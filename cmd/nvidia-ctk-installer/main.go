package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	nriapi "github.com/containerd/nri/pkg/api"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
	"golang.org/x/sys/unix"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/container/runtime"
	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/container/runtime/nri"
	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/toolkit"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/info"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
)

const (
	toolkitPidFilename = "toolkit.pid"
	defaultPidFile     = "/run/nvidia/toolkit/" + toolkitPidFilename

	defaultToolkitInstallDir = "/usr/local/nvidia"
	toolkitSubDir            = "toolkit"

	defaultRuntime = "docker"

	defaultNRIPluginIdx uint = 10
)

var (
	defaultLowLevelRuntimes = []string{"runc", "crun"}
	waitingForSignal        = make(chan bool, 1)
	signalReceived          = make(chan bool, 1)
)

// options stores the command line arguments
type options struct {
	toolkitInstallDir string

	noDaemon    bool
	pidFile     string
	sourceRoot  string
	packageType string

	enableNRIPlugin bool
	nriPluginIndex  uint
	nriSocket       string
	nriNamespace    string

	toolkitOptions toolkit.Options

	debug bool

	noRuntimeConfig bool
	runtime         string
	runtimeOptions  runtime.Options
}

func (o options) toolkitRoot() string {
	return filepath.Join(o.toolkitInstallDir, toolkitSubDir)
}

func main() {
	logger := logrus.New()
	c := NewApp(logger)

	// Run the CLI
	logger.Infof("Starting %v", c.Name)
	if err := c.Run(context.Background(), os.Args); err != nil {
		logger.Errorf("error running %v: %v", c.Name, err)
		os.Exit(1)
	}

	logger.Infof("Completed %v", c.Name)
}

// An app represents the nvidia-ctk-installer.
type app struct {
	logger *logrus.Logger

	toolkit *toolkit.Installer
}

// NewApp creates the CLI app from the specified options.
func NewApp(logger *logrus.Logger) *cli.Command {
	a := app{
		logger: logger,
	}
	return a.build()
}

func (a app) build() *cli.Command {
	options := options{
		toolkitOptions: toolkit.Options{},
	}
	// Create the top-level CLI
	c := cli.Command{
		Name:    "nvidia-ctk-installer",
		Usage:   "Install the NVIDIA Container Toolkit and configure the specified runtime to use the `nvidia` runtime.",
		Version: info.GetVersionString(),
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			return ctx, a.Before(cmd, &options)
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return a.Run(ctx, cmd, &options)
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "no-daemon",
				Aliases:     []string{"n"},
				Usage:       "terminate immediately after setting up the runtime. Note that no cleanup will be performed",
				Destination: &options.noDaemon,
				Sources:     cli.EnvVars("NO_DAEMON"),
			},
			&cli.BoolFlag{
				Name: "no-runtime-config",
				Usage: "Disables the configuration of a container runtime. This is used in cases where the runtime has " +
					"already been configured for use with the toolkit, and the installer is only used to deploy the " +
					"components of the NVIDIA Container Toolkit.",
				Destination: &options.noRuntimeConfig,
				Sources:     cli.EnvVars("NO_RUNTIME_CONFIG"),
			},
			&cli.BoolFlag{
				Name:    "enable-nri-plugin",
				Aliases: []string{"p"},
				Usage: "if set to true, the toolkit will stand up an NRI Plugin server used to inject CDI devices " +
					"to containers. Note that this option will be ignored if --no-daemon is set.",
				Destination: &options.enableNRIPlugin,
				Sources:     cli.EnvVars("ENABLE_NRI_PLUGIN"),
			},
			&cli.UintFlag{
				Name:        "nri-plugin-index",
				Usage:       "Specify the plugin index to register to NRI",
				Value:       defaultNRIPluginIdx,
				Destination: &options.nriPluginIndex,
				Sources:     cli.EnvVars("NRI_PLUGIN_INDEX"),
				Action: func(ctx context.Context, c *cli.Command, u uint) error {
					if u > 99 {
						return fmt.Errorf("nri-plugin-index must be in the range [0,99]")
					}
					return nil
				},
			},
			&cli.StringFlag{
				Name:        "nri-socket",
				Usage:       "Specify the path to the NRI socket file to register the NRI plugin server",
				Value:       nriapi.DefaultSocketPath,
				Destination: &options.nriSocket,
				Sources:     cli.EnvVars("NRI_SOCKET"),
			},
			&cli.StringFlag{
				Name:        "nri-namespace",
				Usage:       "Specify the kubernetes namespace the toolkit's NRI plugin is running in.",
				Destination: &options.nriNamespace,
				Sources:     cli.EnvVars("NRI_NAMESPACE"),
			},
			&cli.StringFlag{
				Name:    "runtime",
				Aliases: []string{"r"},
				Usage: "the runtime to setup on this node. One of {'docker', 'crio', 'containerd'}. " +
					"This setting is ignored if --no-runtime-config is specified.",
				Value:       defaultRuntime,
				Destination: &options.runtime,
				Sources:     cli.EnvVars("RUNTIME"),
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
				Sources:     cli.EnvVars("TOOLKIT_INSTALL_DIR", "ROOT"),
			},
			&cli.StringFlag{
				Name:        "toolkit-source-root",
				Usage:       "The folder where the required toolkit artifacts can be found. If this is not specified, the path /artifacts/{{ .ToolkitPackageType }} is used where ToolkitPackageType is the resolved package type",
				Destination: &options.sourceRoot,
				Sources:     cli.EnvVars("TOOLKIT_SOURCE_ROOT"),
			},
			&cli.StringFlag{
				Name:        "toolkit-package-type",
				Usage:       "specify the package type to use for the toolkit. One of ['deb', 'rpm', 'auto', '']. If 'auto' or '' are used, the type is inferred automatically.",
				Value:       "auto",
				Destination: &options.packageType,
				Sources:     cli.EnvVars("TOOLKIT_PACKAGE_TYPE"),
			},
			&cli.StringFlag{
				Name:        "pid-file",
				Value:       defaultPidFile,
				Usage:       "the path to a toolkit.pid file to ensure that only a single configuration instance is running",
				Destination: &options.pidFile,
				Sources:     cli.EnvVars("TOOLKIT_PID_FILE", "PID_FILE"),
			},
			&cli.BoolFlag{
				Name:        "debug",
				Value:       false,
				Usage:       "enable debug-level log in the toolkit installer",
				Destination: &options.debug,
				Sources:     cli.EnvVars("TOOLKIT_DEBUG"),
			},
		},
	}

	// Add the additional flags specific to the toolkit and runtime config.
	c.Flags = append(c.Flags, toolkit.Flags(&options.toolkitOptions)...)
	c.Flags = append(c.Flags, runtime.Flags(&options.runtimeOptions)...)

	return &c
}

func (a *app) Before(c *cli.Command, o *options) error {
	if o.debug && a.logger != nil {
		a.logger.SetLevel(logrus.DebugLevel)
	}
	if o.sourceRoot == "" {
		sourceRoot, err := a.resolveSourceRoot(o.runtimeOptions.HostRootMount, o.packageType)
		if err != nil {
			return fmt.Errorf("failed to resolve source root: %v", err)
		}
		a.logger.Infof("Resolved source root to %v", sourceRoot)
		o.sourceRoot = sourceRoot
	}

	a.toolkit = toolkit.NewInstaller(
		toolkit.WithLogger(a.logger),
		toolkit.WithSourceRoot(o.sourceRoot),
		toolkit.WithToolkitRoot(o.toolkitRoot()),
	)
	return a.validateFlags(c, o)
}

func (a *app) validateFlags(c *cli.Command, o *options) error {
	if o.toolkitInstallDir == "" {
		return fmt.Errorf("the install root must be specified")
	}
	if filepath.Base(o.pidFile) != toolkitPidFilename {
		return fmt.Errorf("invalid toolkit.pid path %v", o.pidFile)
	}

	if o.enableNRIPlugin && len(o.nriNamespace) == 0 {
		return fmt.Errorf("the NRI namespace must be specified when the NRI plugin is enabled")
	}

	if err := a.toolkit.ValidateOptions(&o.toolkitOptions); err != nil {
		return err
	}

	if err := o.runtimeOptions.Validate(a.logger, c, o.runtime, o.toolkitRoot(), &o.toolkitOptions); err != nil {
		return err
	}

	return nil
}

// Run installs the NVIDIA Container Toolkit and updates the requested runtime.
// If the application is run as a daemon, the application waits and unconfigures
// the runtime on termination.
func (a *app) Run(ctx context.Context, c *cli.Command, o *options) error {
	err := a.initialize(o.pidFile)
	if err != nil {
		return fmt.Errorf("unable to initialize: %v", err)
	}
	defer a.shutdown(o.pidFile)

	runtimeConfigurer := runtime.NewConfigurer(o.runtime, o.noRuntimeConfig, o.enableNRIPlugin)

	if len(o.toolkitOptions.ContainerRuntimeRuntimes) == 0 {
		lowlevelRuntimePaths, err := runtimeConfigurer.GetLowlevelRuntimePaths(&o.runtimeOptions)
		if err != nil {
			return fmt.Errorf("unable to determine runtime options: %w", err)
		}
		lowlevelRuntimePaths = append(lowlevelRuntimePaths, defaultLowLevelRuntimes...)

		o.toolkitOptions.ContainerRuntimeRuntimes = lowlevelRuntimePaths
	}

	err = a.toolkit.Install(c, &o.toolkitOptions, o.runtime)
	if err != nil {
		return fmt.Errorf("unable to install toolkit: %v", err)
	}

	err = runtimeConfigurer.Setup(c, &o.runtimeOptions)
	if err != nil {
		return fmt.Errorf("unable to setup runtime: %w", err)
	}

	if o.noDaemon {
		return nil
	}

	if o.enableNRIPlugin {
		nriPlugin, err := a.startNRIPluginServer(ctx, o)
		if err != nil {
			a.logger.Errorf("unable to start NRI plugin server: %v", err)
		}
		defer nriPlugin.Stop()
	}

	err = a.waitForSignal()
	if err != nil {
		return fmt.Errorf("unable to wait for signal: %v", err)
	}

	err = runtimeConfigurer.Cleanup(c, &o.runtimeOptions)
	if err != nil {
		return fmt.Errorf("unable to cleanup runtime: %v", err)
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

func (a *app) startNRIPluginServer(ctx context.Context, opts *options) (*nri.Plugin, error) {
	a.logger.Infof("Starting the NRI Plugin server....")

	const (
		maxRetryAttempts = 5
		retryBackoff     = 2 * time.Second
	)

	plugin := nri.NewPlugin(ctx, a.logger, opts.nriNamespace)
	retriable := func() error {
		return plugin.Start(ctx, opts.nriSocket, fmt.Sprintf("%02d", opts.nriPluginIndex))
	}
	var err error
	for i := 0; i < maxRetryAttempts; i++ {
		err = retriable()
		if err == nil {
			break
		}
		a.logger.Warningf("Attempt %d - error starting the NRI plugin: %v", i+1, err)
		if i == maxRetryAttempts-1 {
			break
		}
		time.Sleep(retryBackoff)
	}
	if err != nil {
		a.logger.Errorf("Max retries reached %d/%d, aborting", maxRetryAttempts, maxRetryAttempts)
		return nil, err
	}
	return plugin, nil
}

func (a *app) shutdown(pidFile string) {
	a.logger.Infof("Shutting Down")

	err := os.Remove(pidFile)
	if err != nil {
		a.logger.Warningf("Unable to remove pidfile: %v", err)
	}
}

func (a *app) resolveSourceRoot(hostRoot string, packageType string) (string, error) {
	resolvedPackageType, err := a.resolvePackageType(hostRoot, packageType)
	if err != nil {
		return "", err
	}
	switch resolvedPackageType {
	case "deb":
		return "/artifacts/deb", nil
	case "rpm":
		return "/artifacts/rpm", nil
	default:
		return "", fmt.Errorf("invalid package type: %v", resolvedPackageType)
	}
}

func (a *app) resolvePackageType(hostRoot string, packageType string) (rPackageTypes string, rerr error) {
	if packageType != "" && packageType != "auto" {
		return packageType, nil
	}

	locator := lookup.NewExecutableLocator(a.logger, hostRoot)
	if candidates, err := locator.Locate("/usr/bin/rpm"); err == nil && len(candidates) > 0 {
		return "rpm", nil
	}

	if candidates, err := locator.Locate("/usr/bin/dpkg"); err == nil && len(candidates) > 0 {
		return "deb", nil
	}

	return "deb", nil
}
