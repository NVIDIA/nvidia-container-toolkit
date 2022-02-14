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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

const (
	ociSpecFileName          = "config.json"
	dockerRuncExecutableName = "docker-runc"
	runcExecutableName       = "runc"
	nvidiaRuntimeName        = "nvidia"
	runcRuntimeName          = "runc"
	dockerDefaultConfig      = "/etc/docker/daemon.json"
)

type dockerDaemon struct {
	Runtimes map[string]dockerRuntime `json:"runtimes,omitempty"`
}

type dockerRuntime struct {
	Path *string `json:"path,omitempty"`
}

// newRuntime is a factory method that constructs a runtime based on the selected configuration.
func newRuntime(argv []string) (oci.Runtime, error) {
	ociSpec, err := newOCISpec(argv)
	if err != nil {
		return nil, fmt.Errorf("error constructing OCI specification: %v", err)
	}

	runtime, err := newDefaultRuntime()
	if err != nil {
		logger.Errorf("Error constructing default runtime: %v", err)

		runc, err := newRuncRuntime()
		if err != nil {
			return nil, fmt.Errorf("error constructing runc runtime: %v", err)
		}
		runtime = runc
	}

	r, err := newNvidiaContainerRuntimeWithLogger(logger.Logger, runtime, ociSpec)
	if err != nil {
		return nil, fmt.Errorf("error constructing NVIDIA Container Runtime: %v", err)
	}

	return r, nil
}

// newOCISpec constructs an OCI spec for the provided arguments
func newOCISpec(argv []string) (oci.Spec, error) {
	bundleDir, err := oci.GetBundleDir(argv)
	if err != nil {
		return nil, fmt.Errorf("error parsing command line arguments: %v", err)
	}
	logger.Infof("Using bundle directory: %v", bundleDir)

	ociSpecPath := oci.GetSpecFilePath(bundleDir)
	logger.Infof("Using OCI specification file path: %v", ociSpecPath)

	ociSpec := oci.NewSpecFromFile(ociSpecPath)

	return ociSpec, nil
}

// newRuncRuntime locates the runc binary and wraps it in a SyscallExecRuntime
func newRuncRuntime() (oci.Runtime, error) {
	return oci.NewLowLevelRuntimeWithLogger(
		logger.Logger,
		dockerRuncExecutableName,
		runcExecutableName,
	)
}

func dockerRuntimeExecutablePath(name string) (string, error) {
	file, err := os.Open(dockerDefaultConfig)
	if err != nil {
		return "", err
	}

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}

	daemon := dockerDaemon{}
	if err := json.Unmarshal(bytes, &daemon); err != nil {
		return "", err
	}

	return *daemon.Runtimes[name].Path, nil
}

// newDefaultRuntime locates the default runtime binary and wraps it in a SyscallExecRuntime
func newDefaultRuntime() (oci.Runtime, error) {
	cmd := exec.Command("docker", "info", "--format", "{{.DefaultRuntime}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error getting docker default runtime: %v", err)
	}
	defaultRuntimeName := strings.TrimSpace(string(output))
	if defaultRuntimeName == nvidiaRuntimeName || defaultRuntimeName == runcRuntimeName {
		return nil, fmt.Errorf("docker default runtime is %v, bail out: %v", defaultRuntimeName, err)
	}
	candidates := []string{}
	defaultRuntimeExecutablePath, err := dockerRuntimeExecutablePath(defaultRuntimeName)
	if err != nil {
		logger.Errorf("Error getting docker default runtime (%v)'s executable path: %v", defaultRuntimeName, err)
		candidates = append(candidates, defaultRuntimeName)
	} else {
		candidates = append(candidates, defaultRuntimeExecutablePath)
	}
	return oci.NewLowLevelRuntimeWithLogger(
		logger.Logger,
		candidates...,
	)
}
