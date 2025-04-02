package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/urfave/cli/v2"
	"golang.org/x/sys/unix"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/container/runtime"
	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/container/toolkit"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/info"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

const (
	toolkitPidFilename = "toolkit.pid"
	defaultPidFile     = "/run/nvidia/toolkit/" + toolkitPidFilename

	defaultToolkitInstallDir = "/usr/local/nvidia"
	toolkitSubDir            = "toolkit"

	defaultRuntime = "docker"
)

var availableRuntimes = map[string]struct{}{"docker": {}, "crio": {}, "containerd": {}}
var defaultLowLevelRuntimes = []string{"docker-runc", "runc", "crun"}

var waitingForSignal = make(chan bool, 1)
var signalReceived = make(chan bool, 1)

// options stores the command line arguments
type options struct {
	toolkitInstallDir string

	noDaemon   bool
	runtime    string
	pidFile    string
	sourceRoot string

	toolkitOptions toolkit.Options
	runtimeOptions runtime.Options
}

func (o options) toolkitRoot() string {
	return filepath.Join(o.toolkitInstallDir, toolkitSubDir)
}

func main() {
	logger := logger.New()
	c := NewApp(logger)

	// Run the CLI
	logger.Infof("Starting %v", c.Name)
	if err := c.Run(os.Args); err != nil {
		logger.Errorf("error running %v: %v", c.Name, err)
		os.Exit(1)
	}

	logger.Infof("Completed %v", c.Name)
}

// An app represents the nvidia-ctk-installer.
type app struct {
	logger logger.Interface

	toolkit *toolkit.Installer
}

// NewApp creates the CLI app fro the specified options.
func NewApp(logger logger.Interface) *cli.App {
	a := app{
		logger: logger,
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
	c.Usage = "Install the NVIDIA Container Toolkit and configure the specified runtime to use the `nvidia` runtime."
	c.Version = info.GetVersionString()
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
		&cli.StringFlag{
			Name:    "toolkit-install-dir",
			Aliases: []string{"root"},
			Usage: "The directory where the NVIDIA Container Toolkit is to be installed. " +
				"The components of the toolkit will be installed to `ROOT`/toolkit. " +
				"Note that in the case of a containerized installer, this is the path in the container and it is " +
				"recommended that this match the path on the host.",
			Value:       defaultToolkitInstallDir,
			Destination: &options.toolkitInstallDir,
			EnvVars:     []string{"TOOLKIT_INSTALL_DIR", "ROOT"},
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
	if o.toolkitInstallDir == "" {
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

	_, err = fmt.Fprintf(f, "%v\n", os.Getpid())
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
