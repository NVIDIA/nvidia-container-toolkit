package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"

	cli "github.com/urfave/cli/v2"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/info"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
)

func (a *app) recoverIfRequired() error {
	if err := recover(); err != nil {
		rerr, ok := err.(runtime.Error)
		if ok {
			log.Println(err)
		}
		if a.isDebug {
			log.Printf("%s", debug.Stack())
		}
		return rerr
	}
	return nil
}

func getCLIPath(config config.ContainerCLIConfig) string {
	if config.Path != "" {
		return config.Path
	}

	if err := os.Setenv("PATH", lookup.GetPath(config.Root)); err != nil {
		log.Panicln("couldn't set PATH variable:", err)
	}

	path, err := exec.LookPath("nvidia-container-cli")
	if err != nil {
		log.Panicln("couldn't find binary nvidia-container-cli in", os.Getenv("PATH"), ":", err)
	}
	return path
}

// getRootfsPath returns an absolute path. We don't need to resolve symlinks for now.
func getRootfsPath(config containerConfig) string {
	rootfs, err := filepath.Abs(config.Rootfs)
	if err != nil {
		log.Panicln(err)
	}
	return rootfs
}

func (a *app) doPrestart() (rerr error) {
	defer func() {
		rerr = errors.Join(rerr, a.recoverIfRequired())
	}()
	log.SetFlags(0)

	hook, err := a.getHookConfig()
	if err != nil || hook == nil {
		return fmt.Errorf("error getting hook config: %w", err)
	}
	cli := hook.NVIDIAContainerCLIConfig

	container := getContainerConfig(*hook)
	nvidia := container.Nvidia
	if nvidia == nil {
		// Not a GPU container, nothing to do.
		return nil
	}

	if !hook.NVIDIAContainerRuntimeHookConfig.SkipModeDetection && info.ResolveAutoMode(&logInterceptor{}, hook.NVIDIAContainerRuntimeConfig.Mode, container.Image) != "legacy" {
		return fmt.Errorf("invoking the NVIDIA Container Runtime Hook directly (e.g. specifying the docker --gpus flag) is not supported. Please use the NVIDIA Container Runtime (e.g. specify the --runtime=nvidia flag) instead")
	}

	rootfs := getRootfsPath(container)

	args := []string{getCLIPath(cli)}
	if cli.Root != "" {
		args = append(args, fmt.Sprintf("--root=%s", cli.Root))
	}
	if cli.LoadKmods {
		args = append(args, "--load-kmods")
	}
	if hook.Features.DisableImexChannelCreation.IsEnabled() {
		args = append(args, "--no-create-imex-channels")
	}
	if cli.NoPivot {
		args = append(args, "--no-pivot")
	}
	if a.isDebug {
		args = append(args, "--debug=/dev/stderr")
	} else if cli.Debug != "" {
		args = append(args, fmt.Sprintf("--debug=%s", cli.Debug))
	}
	if cli.Ldcache != "" {
		args = append(args, fmt.Sprintf("--ldcache=%s", cli.Ldcache))
	}
	if cli.User != "" {
		args = append(args, fmt.Sprintf("--user=%s", cli.User))
	}
	args = append(args, "configure")

	if ldconfigPath := cli.NormalizeLDConfigPath(); ldconfigPath != "" {
		args = append(args, fmt.Sprintf("--ldconfig=%s", ldconfigPath))
	}
	if cli.NoCgroups {
		args = append(args, "--no-cgroups")
	}
	if devicesString := strings.Join(nvidia.Devices, ","); len(devicesString) > 0 {
		args = append(args, fmt.Sprintf("--device=%s", devicesString))
	}
	if len(nvidia.MigConfigDevices) > 0 {
		args = append(args, fmt.Sprintf("--mig-config=%s", nvidia.MigConfigDevices))
	}
	if len(nvidia.MigMonitorDevices) > 0 {
		args = append(args, fmt.Sprintf("--mig-monitor=%s", nvidia.MigMonitorDevices))
	}
	if imexString := strings.Join(nvidia.ImexChannels, ","); len(imexString) > 0 {
		args = append(args, fmt.Sprintf("--imex-channel=%s", imexString))
	}

	for _, cap := range strings.Split(nvidia.DriverCapabilities, ",") {
		if len(cap) == 0 {
			break
		}
		args = append(args, capabilityToCLI(cap))
	}

	for _, req := range nvidia.Requirements {
		args = append(args, fmt.Sprintf("--require=%s", req))
	}

	args = append(args, fmt.Sprintf("--pid=%s", strconv.FormatUint(uint64(container.Pid), 10)))
	args = append(args, rootfs)

	env := append(os.Environ(), cli.Environment...)
	//nolint:gosec // TODO: Can we harden this so that there is less risk of command injection?
	return syscall.Exec(args[0], args, env)
}

type options struct {
	isDebug    bool
	configFile string
}
type app struct {
	options
}

func main() {
	a := &app{}
	// Create the top-level CLI
	c := cli.NewApp()
	c.Name = "NVIDIA Container Runtime Hook"
	c.Version = info.GetVersionString()

	c.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:        "debug",
			Destination: &a.isDebug,
			Usage:       "Enabled debug output",
		},
		&cli.StringFlag{
			Name:        "config",
			Destination: &a.configFile,
			Usage:       "The path to the configuration file to use",
			EnvVars:     []string{config.FilePathOverrideEnvVar},
		},
	}

	c.Commands = []*cli.Command{
		{
			Name:  "prestart",
			Usage: "run the prestart hook",
			Action: func(ctx *cli.Context) error {
				return a.doPrestart()
			},
		},
		{
			Name:    "poststart",
			Aliases: []string{"poststop"},
			Usage:   "no-op",
			Action: func(ctx *cli.Context) error {
				return nil
			},
		},
	}
	c.DefaultCommand = "prestart"

	// Run the CLI
	err := c.Run(os.Args)
	if err != nil {
		os.Exit(1)
	}
}

// logInterceptor implements the logger.Interface to allow for logging from executable.
type logInterceptor struct {
	logger.NullLogger
}

func (l *logInterceptor) Infof(format string, args ...interface{}) {
	log.Printf(format, args...)
}
