package main

import (
	"fmt"
	"os"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
)

var logger = NewLogger()

func main() {
	err := run(os.Args)
	if err != nil {
		logger.Errorf("%v", err)
		os.Exit(1)
	}
}

// run is an entry point that allows for idiomatic handling of errors
// when calling from the main function.
func run(argv []string) (rerr error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return fmt.Errorf("error loading config: %v", err)
	}

	logger, err = UpdateLogger(
		cfg.NVIDIAContainerRuntimeConfig.DebugFilePath,
		cfg.NVIDIAContainerRuntimeConfig.LogLevel,
		argv,
	)
	if err != nil {
		return fmt.Errorf("failed to set up logger: %v", err)
	}

	logger.Debugf("Command line arguments: %v", argv)
	runtime, err := newNVIDIAContainerRuntime(logger.Logger, cfg, argv)
	if err != nil {
		return fmt.Errorf("failed to create NVIDIA Container Runtime: %v", err)
	}

	return runtime.Exec(argv)
}
