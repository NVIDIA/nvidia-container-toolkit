package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/urfave/cli/v2"
	"golang.org/x/sys/unix"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/container/runtime"
	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/container/toolkit"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

const (
	toolkitPidFilename = "toolkit.pid"
	defaultPidFile     = "/run/nvidia/toolkit/" + toolkitPidFilename
	toolkitSubDir      = "toolkit"

	defaultRuntime     = "docker"
	defaultRuntimeArgs = ""
)

var availableRuntimes = map[string]struct{}{"docker": {}, "crio": {}, "containerd": {}}
var defaultLowLevelRuntimes = []string{"docker-runc", "runc", "crun"}

var waitingForSignal = make(chan bool, 1)
var signalReceived = make(chan bool, 1)

// options stores the command line arguments
type options struct {
	noDaemon    bool
	runtime     string
	runtimeArgs string
	root        string
	pidFile     string
	sourceRoot  string

	toolkitOptions toolkit.Options
	runtimeOptions runtime.Options
}

func (o options) toolkitRoot() string {
	return filepath.Join(o.root, toolkitSubDir)
}

// Version defines the CLI version. This is set at build time using LD FLAGS
var Version = "development"

func main() {
	logger := logger.New()

	remainingArgs, root, err := ParseArgs(logger, os.Args)
	if err != nil {
		logger.Errorf("Error: unable to parse arguments: %v", err)
		os.Exit(1)
	}

	c := NewApp(logger, root)

	// Run the CLI
	logger.Infof("Starting %v", c.Name)
	if err := c.Run(remainingArgs); err != nil {
		logger.Errorf("error running nvidia-toolkit: %v", err)
		os.Exit(1)
	}

	logger.Infof("Completed %v", c.Name)
}

// An app represents the nvidia-ctk-installer.
type app struct {
	logger logger.Interface
	// defaultRoot stores the root to use if the --root flag is not specified.
	defaultRoot string

	toolkit *toolkit.Installer
}

// NewApp creates the CLI app fro the specified options.
// defaultRoot is used as the root if not specified via the --root flag.
func NewApp(logger logger.Interface, defaultRoot string) *cli.App {
	a := app{
		logger:      logger,
		defaultRoot: defaultRoot,
	}
	return a.build()
}

func (a app) build() *cli.App {
	options := options{
		toolkitOptions: toolkit.Options{},
	}
	// Create the top-level CLI
	c := cli.NewApp()
	c.Name = "nvidia-ctk-installer"
	c.Usage = "Install the nvidia-container-toolkit for use by a given runtime"
	c.UsageText = "[DESTINATION] [-n | --no-daemon] [-r | --runtime] [-u | --runtime-args]"
	c.Description = "DESTINATION points to the host path underneath which the nvidia-container-toolkit should be installed.\nIt will be installed at ${DESTINATION}/toolkit"
	c.Version = Version
	c.Before = func(ctx *cli.Context) error {
		return a.Before(ctx, &options)
	}
	c.Action = func(ctx *cli.Context) error {
		return a.Run(ctx, &options)
	}

	// Setup flags for the CLI
	c.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:        "no-daemon",
			Aliases:     []string{"n"},
			Usage:       "terminate immediately after setting up the runtime. Note that no cleanup will be performed",
			Destination: &options.noDaemon,
			EnvVars:     []string{"NO_DAEMON"},
		},
		&cli.StringFlag{
			Name:        "runtime",
			Aliases:     []string{"r"},
			Usage:       "the runtime to setup on this node. One of {'docker', 'crio', 'containerd'}",
			Value:       defaultRuntime,
			Destination: &options.runtime,
			EnvVars:     []string{"RUNTIME"},
		},
		// TODO: Remove runtime-args
		&cli.StringFlag{
			Name:        "runtime-args",
			Aliases:     []string{"u"},
			Usage:       "arguments to pass to 'docker', 'crio', or 'containerd' setup command",
			Value:       defaultRuntimeArgs,
			Destination: &options.runtimeArgs,
			EnvVars:     []string{"RUNTIME_ARGS"},
		},
		&cli.StringFlag{
			Name:        "root",
			Value:       a.defaultRoot,
			Usage:       "the folder where the NVIDIA Container Toolkit is to be installed. It will be installed to `ROOT`/toolkit",
			Destination: &options.root,
			EnvVars:     []string{"ROOT"},
		},
		&cli.StringFlag{
			Name:        "source-root",
			Value:       "/",
			Usage:       "The folder where the required toolkit artifacts can be found",
			Destination: &options.sourceRoot,
			EnvVars:     []string{"SOURCE_ROOT"},
		},
		&cli.StringFlag{
			Name:        "pid-file",
			Value:       defaultPidFile,
			Usage:       "the path to a toolkit.pid file to ensure that only a single configuration instance is running",
			Destination: &options.pidFile,
			EnvVars:     []string{"TOOLKIT_PID_FILE", "PID_FILE"},
		},
	}

	c.Flags = append(c.Flags, toolkit.Flags(&options.toolkitOptions)...)
	c.Flags = append(c.Flags, runtime.Flags(&options.runtimeOptions)...)

	return c
}

func (a *app) Before(c *cli.Context, o *options) error {
	a.toolkit = toolkit.NewInstaller(
		toolkit.WithLogger(a.logger),
		toolkit.WithSourceRoot(o.sourceRoot),
		toolkit.WithToolkitRoot(o.toolkitRoot()),
	)
	return a.validateFlags(c, o)
}

func (a *app) validateFlags(c *cli.Context, o *options) error {
	if o.root == "" {
		return fmt.Errorf("the install root must be specified")
	}
	if _, exists := availableRuntimes[o.runtime]; !exists {
		return fmt.Errorf("unknown runtime: %v", o.runtime)
	}
	if filepath.Base(o.pidFile) != toolkitPidFilename {
		return fmt.Errorf("invalid toolkit.pid path %v", o.pidFile)
	}

	if err := a.toolkit.ValidateOptions(&o.toolkitOptions); err != nil {
		return err
	}
	if err := runtime.ValidateOptions(c, &o.runtimeOptions, o.runtime, o.toolkitRoot(), &o.toolkitOptions); err != nil {
		return err
	}
	return nil
}

// Run installs the NVIDIA Container Toolkit and updates the requested runtime.
// If the application is run as a daemon, the application waits and unconfigures
// the runtime on termination.
func (a *app) Run(c *cli.Context, o *options) error {
	err := a.initialize(o.pidFile)
	if err != nil {
		return fmt.Errorf("unable to initialize: %v", err)
	}
	defer a.shutdown(o.pidFile)

	if len(o.toolkitOptions.ContainerRuntimeRuntimes.Value()) == 0 {
		lowlevelRuntimePaths, err := runtime.GetLowlevelRuntimePaths(&o.runtimeOptions, o.runtime)
		if err != nil {
			return fmt.Errorf("unable to determine runtime options: %w", err)
		}
		lowlevelRuntimePaths = append(lowlevelRuntimePaths, defaultLowLevelRuntimes...)

		o.toolkitOptions.ContainerRuntimeRuntimes = *cli.NewStringSlice(lowlevelRuntimePaths...)
	}

	err = a.toolkit.Install(c, &o.toolkitOptions)
	if err != nil {
		return fmt.Errorf("unable to install toolkit: %v", err)
	}

	err = runtime.Setup(c, &o.runtimeOptions, o.runtime)
	if err != nil {
		return fmt.Errorf("unable to setup runtime: %v", err)
	}

	if !o.noDaemon {
		err = a.waitForSignal()
		if err != nil {
			return fmt.Errorf("unable to wait for signal: %v", err)
		}

		err = runtime.Cleanup(c, &o.runtimeOptions, o.runtime)
		if err != nil {
			return fmt.Errorf("unable to cleanup runtime: %v", err)
		}
	}

	return nil
}

// ParseArgs checks if a single positional argument was defined and extracts this the root.
// If no positional arguments are defined, it is assumed that the root is specified as a flag.
func ParseArgs(logger logger.Interface, args []string) ([]string, string, error) {
	logger.Infof("Parsing arguments")

	if len(args) < 2 {
		return args, "", nil
	}

	var lastPositionalArg int
	for i, arg := range args {
		if strings.HasPrefix(arg, "-") {
			break
		}
		lastPositionalArg = i
	}

	if lastPositionalArg == 0 {
		return args, "", nil
	}

	if lastPositionalArg == 1 {
		return append([]string{args[0]}, args[2:]...), args[1], nil
	}

	return nil, "", fmt.Errorf("unexpected positional argument(s) %v", args[2:lastPositionalArg+1])
}

func (a *app) initialize(pidFile string) error {
	a.logger.Infof("Initializing")

	if dir := filepath.Dir(pidFile); dir != "" {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return fmt.Errorf("unable to create folder for pidfile: %w", err)
		}
	}

	f, err := os.Create(pidFile)
	if err != nil {
		return fmt.Errorf("unable to create pidfile: %v", err)
	}

	err = unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err != nil {
		a.logger.Warningf("Unable to get exclusive lock on '%v'", pidFile)
		a.logger.Warningf("This normally means an instance of the NVIDIA toolkit Container is already running, aborting")
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
			a.logger.Infof("Signal received, exiting early")
			a.shutdown(pidFile)
			os.Exit(0)
		}
	}()

	return nil
}

func (a *app) waitForSignal() error {
	a.logger.Infof("Waiting for signal")
	waitingForSignal <- true
	<-signalReceived
	return nil
}

func (a *app) shutdown(pidFile string) {
	a.logger.Infof("Shutting Down")

	err := os.Remove(pidFile)
	if err != nil {
		a.logger.Warningf("Unable to remove pidfile: %v", err)
	}
}
