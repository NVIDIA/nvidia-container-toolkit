/*
# Copyright (c) 2021, NVIDIA CORPORATION.  All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
*/

package main

import (
	"fmt"
	"os"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/ensure"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/filter"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/modify"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/runtime"
	log "github.com/sirupsen/logrus"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-container-runtime.experimental/config"
)

const (
	visibleDevicesEnvvar = "NVIDIA_VISIBLE_DEVICES"
	visibleDevicesVoid   = "void"
)

var logger = log.New()

func main() {
	cfg, err := config.GetConfig(logger)
	if err != nil {
		logger.Errorf("Error loading config: %v", err)
		os.Exit(1)
	}

	if cfg.DebugFilePath != "" && cfg.DebugFilePath != "/dev/nul" {
		logFile, err := os.OpenFile(cfg.DebugFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			logger.Errorf("Error opening debug log file: %v", err)
			os.Exit(1)
		}
		defer logFile.Close()

		logger.SetOutput(logFile)
	}

	logLevel, err := log.ParseLevel(cfg.LogLevel)
	if err == nil {
		logger.SetLevel(logLevel)
	} else {
		logger.Warnf("Invalid log-level '%v'; using '%v'", cfg.LogLevel, logger.Level.String())
	}

	logger.Infof("Starting nvidia-container-runtime: %v", os.Args)
	logger.Debugf("Using config=%+v", cfg)

	if err := run(cfg, os.Args); err != nil {
		logger.Errorf("Error running runtime: %v", err)
		os.Exit(1)
	}
}

func run(cfg *config.Config, args []string) error {
	logger.Debugf("running with args=%v", args)

	// We create a low-level runtime
	lowLevelRuntime, err := createLowLevelRuntime(logger, cfg)
	if err != nil {
		return fmt.Errorf("error constructing low-level runtime: %v", err)
	}

	if !oci.HasCreateSubcommand(args) {
		logger.Infof("No modification of OCI specification required")
		logger.Infof("Forwarding command to runtime")
		return lowLevelRuntime.Exec(args)
	}

	// We create the OCI spec that is to be modified
	ociSpec, bundleDir, err := oci.NewSpecFromArgs(args)
	if err != nil {
		return fmt.Errorf("error constructing OCI spec: %v", err)
	}

	err = ociSpec.Load()
	if err != nil {
		return fmt.Errorf("error loading OCI specification: %v", err)
	}

	visibleDevices, exists := ociSpec.LookupEnv(visibleDevicesEnvvar)
	if !exists || visibleDevices == "" || visibleDevices == visibleDevicesVoid {
		logger.Infof("Using low-level runtime: %v=%v (exists=%v)", visibleDevicesEnvvar, visibleDevices, exists)
		return lowLevelRuntime.Exec(os.Args)
	}

	// We create the modifier that will be applied by the Modifying Runtime Wrapper
	modifier, err := createModifier(cfg.Root, bundleDir, visibleDevices, ociSpec)
	if err != nil {
		return fmt.Errorf("error constructing modifer: %v", err)
	}

	// We construct the Modifying runtime
	r := runtime.NewModifyingRuntimeWrapperWithLogger(logger, lowLevelRuntime, ociSpec, modifier)

	return r.Exec(os.Args)
}

func createLowLevelRuntime(logger *log.Logger, cfg *config.Config) (oci.Runtime, error) {
	if cfg.RuntimePath == "" {
		return oci.NewLowLevelRuntimeWithLogger(logger, "docker-runc", "runc")
	}
	logger.Infof("Creating runtime with path %v", cfg.RuntimePath)
	return oci.NewRuntimeForPathWithLogger(logger, cfg.RuntimePath)
}

func createModifier(root string, bundleDir string, visibleDevices string, env filter.EnvLookup) (modify.Modifier, error) {
	// We set up the modifier using discovery
	discovered, err := discover.NewNVMLServerWithLogger(logger, root)
	if err != nil {
		return nil, fmt.Errorf("error discovering devices: %v", err)
	}

	// We apply a filter to the discovered devices
	selected := filter.NewSelectDevicesFromWithLogger(logger, discovered, visibleDevices, env)

	// We ensure that the selected devices are available
	available := ensure.NewEnsureDevicesWithLogger(logger, selected, root)

	// We construct the modifer for the OCI spec
	modifier := modify.NewModifierWithLoggerFor(logger, available, root, bundleDir)

	return modifier, nil
}
