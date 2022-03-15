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
		logger.Errorf("Error running %v: %v", os.Args, err)
		os.Exit(1)
	}
}

// run is an entry point that allows for idiomatic handling of errors
// when calling from the main function.
func run(argv []string) (rerr error) {
	cfg, err := config.GetRuntimeConfig()
	if err != nil {
		return fmt.Errorf("error loading config: %v", err)
	}

	err = logger.LogToFile(cfg.DebugFilePath)
	if err != nil {
		return fmt.Errorf("error opening debug log file: %v", err)
	}
	defer func() {
		// We capture and log a returning error before closing the log file.
		if rerr != nil {
			logger.Errorf("Error running %v: %v", argv, rerr)
		}
		logger.CloseFile()
	}()

	runtime, err := newNVIDIAContainerRuntime(logger.Logger, cfg, argv)
	if err != nil {
		return fmt.Errorf("failed to create NVIDIA Container Runtime: %v", err)
	}

	return runtime.Exec(argv)
}
