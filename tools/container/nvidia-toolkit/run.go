package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"
	unix "golang.org/x/sys/unix"
)

const (
	runDir         = "/run/nvidia"
	pidFile        = runDir + "/toolkit.pid"
	toolkitCommand = "toolkit"
	toolkitSubDir  = "toolkit"

	defaultToolkitArgs = ""
	defaultRuntime     = "docker"
	defaultRuntimeArgs = ""
)

var availableRuntimes = map[string]struct{}{"docker": {}, "crio": {}, "containerd": {}}

var waitingForSignal = make(chan bool, 1)
var signalReceived = make(chan bool, 1)

var destinationArg string
var noDaemonFlag bool
var toolkitArgsFlag string
var runtimeFlag string
var runtimeArgsFlag string

// Version defines the CLI version. This is set at build time using LD FLAGS
var Version = "development"

func main() {
	// Create the top-level CLI
	c := cli.NewApp()
	c.Name = "nvidia-toolkit"
	c.Usage = "Install the nvidia-container-toolkit for use by a given runtime"
	c.UsageText = "DESTINATION [-n | --no-daemon] [-t | --toolkit-args] [-r | --runtime] [-u | --runtime-args]"
	c.Description = "DESTINATION points to the host path underneath which the nvidia-container-toolkit should be installed.\nIt will be installed at ${DESTINATION}/toolkit"
	c.Version = Version
	c.Action = Run

	// Setup flags for the CLI
	c.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:        "no-daemon",
			Aliases:     []string{"n"},
			Usage:       "terminate immediatly after setting up the runtime. Note that no cleanup will be performed",
			Destination: &noDaemonFlag,
			EnvVars:     []string{"NO_DAEMON"},
		},
		&cli.StringFlag{
			Name:        "toolkit-args",
			Aliases:     []string{"t"},
			Usage:       "arguments to pass to the underlying 'toolkit' command",
			Value:       defaultToolkitArgs,
			Destination: &toolkitArgsFlag,
			EnvVars:     []string{"TOOLKIT_ARGS"},
		},
		&cli.StringFlag{
			Name:        "runtime",
			Aliases:     []string{"r"},
			Usage:       "the runtime to setup on this node. One of {'docker', 'crio', 'containerd'}",
			Value:       defaultRuntime,
			Destination: &runtimeFlag,
			EnvVars:     []string{"RUNTIME"},
		},
		&cli.StringFlag{
			Name:        "runtime-args",
			Aliases:     []string{"u"},
			Usage:       "arguments to pass to 'docker', 'crio', or 'containerd' setup command",
			Value:       defaultRuntimeArgs,
			Destination: &runtimeArgsFlag,
			EnvVars:     []string{"RUNTIME_ARGS"},
		},
	}

	// Run the CLI
	log.Infof("Starting %v", c.Name)

	remainingArgs, err := ParseArgs(os.Args)
	if err != nil {
		log.Errorf("Error: unable to parse arguments: %v", err)
		os.Exit(1)
	}

	if err := c.Run(remainingArgs); err != nil {
		log.Errorf("error running nvidia-toolkit: %v", err)
		os.Exit(1)
	}

	log.Infof("Completed %v", c.Name)
}

// Run runs the core logic of the CLI
func Run(c *cli.Context) error {
	err := verifyFlags()
	if err != nil {
		return fmt.Errorf("unable to verify flags: %v", err)
	}

	err = initialize()
	if err != nil {
		return fmt.Errorf("unable to initialize: %v", err)
	}
	defer shutdown()

	err = installToolkit()
	if err != nil {
		return fmt.Errorf("unable to install toolkit: %v", err)
	}

	err = setupRuntime()
	if err != nil {
		return fmt.Errorf("unable to setup runtime: %v", err)
	}

	if !noDaemonFlag {
		err = waitForSignal()
		if err != nil {
			return fmt.Errorf("unable to wait for signal: %v", err)
		}

		err = cleanupRuntime()
		if err != nil {
			return fmt.Errorf("unable to cleanup runtime: %v", err)
		}
	}

	return nil
}

// ParseArgs parses the command line arguments and returns the remaining arguments
func ParseArgs(args []string) ([]string, error) {
	log.Infof("Parsing arguments")

	numPositionalArgs := 2 // Includes command itself

	if len(args) < numPositionalArgs {
		return nil, fmt.Errorf("missing arguments")
	}

	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return []string{args[0], arg}, nil
		}
		if arg == "--version" || arg == "-v" {
			return []string{args[0], arg}, nil
		}
	}

	for _, arg := range args[:numPositionalArgs] {
		if strings.HasPrefix(arg, "-") {
			return nil, fmt.Errorf("unexpected flag where argument should be")
		}
	}

	for _, arg := range args[numPositionalArgs:] {
		if !strings.HasPrefix(arg, "-") {
			return nil, fmt.Errorf("unexpected argument where flag should be")
		}
	}

	destinationArg = args[1]

	return append([]string{args[0]}, args[numPositionalArgs:]...), nil
}

func verifyFlags() error {
	log.Infof("Verifying Flags")
	if _, exists := availableRuntimes[runtimeFlag]; !exists {
		return fmt.Errorf("unknown runtime: %v", runtimeFlag)
	}
	return nil
}

func initialize() error {
	log.Infof("Initializing")

	f, err := os.Create(pidFile)
	if err != nil {
		return fmt.Errorf("unable to create pidfile: %v", err)
	}

	err = unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err != nil {
		log.Warnf("Unable to get exclusive lock on '%v'", pidFile)
		log.Warnf("This normally means an instance of the NVIDIA toolkit Container is already running, aborting")
		return fmt.Errorf("unable to get flock on pidfile: %v", err)
	}

	_, err = f.WriteString(fmt.Sprintf("%v\n", os.Getpid()))
	if err != nil {
		return fmt.Errorf("unable to write PID to pidfile: %v", err)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGPIPE, syscall.SIGTERM)
	go func() {
		<-sigs
		select {
		case <-waitingForSignal:
			signalReceived <- true
		default:
			log.Infof("Signal received, exiting early")
			shutdown()
			os.Exit(0)
		}
	}()

	return nil
}

func installToolkit() error {
	toolkitDir := filepath.Join(destinationArg, toolkitSubDir)

	log.Infof("Installing toolkit")

	cmdline := fmt.Sprintf("%v install %v %v\n", toolkitCommand, toolkitArgsFlag, toolkitDir)
	cmd := exec.Command("sh", "-c", cmdline)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error running %v command: %v", toolkitCommand, err)
	}

	return nil
}

func setupRuntime() error {
	toolkitDir := filepath.Join(destinationArg, toolkitSubDir)

	log.Infof("Setting up runtime")

	cmdline := fmt.Sprintf("%v setup %v %v\n", runtimeFlag, runtimeArgsFlag, toolkitDir)

	cmd := exec.Command("sh", "-c", cmdline)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error running %v command: %v", runtimeFlag, err)
	}

	return nil
}

func waitForSignal() error {
	log.Infof("Waiting for signal")
	waitingForSignal <- true
	<-signalReceived
	return nil
}

func cleanupRuntime() error {
	toolkitDir := filepath.Join(destinationArg, toolkitSubDir)

	log.Infof("Cleaning up Runtime")

	cmdline := fmt.Sprintf("%v cleanup %v %v\n", runtimeFlag, runtimeArgsFlag, toolkitDir)

	cmd := exec.Command("sh", "-c", cmdline)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error running %v command: %v", runtimeFlag, err)
	}

	return nil
}

func shutdown() {
	log.Infof("Shutting Down")

	err := os.Remove(pidFile)
	if err != nil {
		log.Warnf("Unable to remove pidfile: %v", err)
	}
}
