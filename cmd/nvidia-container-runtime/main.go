package main

import (
	"fmt"
	"os"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/sirupsen/logrus"
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
	logger.Debugf("Running %v", argv)
	cfg, err := config.GetConfig()
	if err != nil {
		return fmt.Errorf("error loading config: %v", err)
	}

	err = logger.LogToFile(cfg.NVIDIAContainerRuntimeConfig.DebugFilePath)
	if err != nil {
		return fmt.Errorf("error opening debug log file: %v", err)
	}
	defer func() {
		// We capture and log a returning error before closing the log file.
		if rerr != nil {
			logger.Errorf("%v", rerr)
		}
		logger.CloseFile()
	}()

	if logLevel, err := logrus.ParseLevel(cfg.NVIDIAContainerRuntimeConfig.LogLevel); err == nil {
		logger.SetLevel(logLevel)
	} else {
		logger.Warnf("Invalid log-level '%v'; using '%v'", cfg.NVIDIAContainerRuntimeConfig.LogLevel, logger.Level.String())
	}

	runtime, err := newNVIDIAContainerRuntime(logger.Logger, cfg, argv)
	if err != nil {
		return fmt.Errorf("failed to create NVIDIA Container Runtime: %v", err)
	}

	return runtime.Exec(argv)
}
