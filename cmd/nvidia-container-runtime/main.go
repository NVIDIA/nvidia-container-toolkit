package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/info"
	"github.com/opencontainers/runtime-spec/specs-go"
)

// version must be set by go build's -X main.version= option in the Makefile.
var version = "unknown"

// gitCommit will be the hash that the binary was built from
// and will be populated by the Makefile
var gitCommit = ""

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
	printVersion := hasVersionFlag(argv)
	if printVersion {
		fmt.Printf("%v version %v\n", "NVIDIA Container Runtime", info.GetVersionString(fmt.Sprintf("spec: %v", specs.Version)))
	}

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
	defer logger.Reset()

	logger.Debugf("Command line arguments: %v", argv)
	runtime, err := newNVIDIAContainerRuntime(logger.Logger, cfg, argv)
	if err != nil {
		return fmt.Errorf("failed to create NVIDIA Container Runtime: %v", err)
	}

	if printVersion {
		fmt.Print("\n")
	}
	return runtime.Exec(argv)
}

// TODO: This should be refactored / combined with parseArgs in logger.
func hasVersionFlag(args []string) bool {
	for i := 0; i < len(args); i++ {
		param := args[i]

		parts := strings.SplitN(param, "=", 2)
		trimmed := strings.TrimLeft(parts[0], "-")
		// If this is not a flag we continue
		if parts[0] == trimmed {
			continue
		}

		// Check the version flag
		if trimmed == "version" {
			return true
		}
	}

	return false
}
