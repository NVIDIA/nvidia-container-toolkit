/**
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
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	nvidiaContainerRuntimeSource  = "/usr/bin/nvidia-container-runtime"
	nvidiaContainerRuntimeTarget  = "nvidia-container-runtime.real"
	nvidiaContainerRuntimeWrapper = "nvidia-container-runtime"

	nvidiaExperimentalContainerRuntimeSource  = "nvidia-container-runtime.experimental"
	nvidiaExperimentalContainerRuntimeTarget  = nvidiaExperimentalContainerRuntimeSource
	nvidiaExperimentalContainerRuntimeWrapper = "nvidia-container-runtime-experimental"
)

// installContainerRuntimes sets up the NVIDIA container runtimes, copying the executables
// and implementing the required wrapper
func installContainerRuntimes(toolkitDir string, driverRoot string) error {
	r := newNvidiaContainerRuntimeInstaller()

	_, err := r.install(toolkitDir)
	if err != nil {
		return fmt.Errorf("error installing NVIDIA container runtime: %v", err)
	}

	// Install the experimental runtime and treat failures as non-fatal.
	err = installExperimentalRuntime(toolkitDir, driverRoot)
	if err != nil {
		log.Warnf("Could not install experimental runtime: %v", err)
	}

	return nil
}

// installExperimentalRuntime ensures that the experimental NVIDIA Container runtime is installed
func installExperimentalRuntime(toolkitDir string, driverRoot string) error {
	libraryRoot, err := findLibraryRoot(driverRoot)
	if err != nil {
		log.Warnf("Error finding library path for root %v: %v", driverRoot, err)
	}
	log.Infof("Using library root %v", libraryRoot)

	e := newNvidiaContainerRuntimeExperimentalInstaller(libraryRoot)
	_, err = e.install(toolkitDir)
	if err != nil {
		return fmt.Errorf("error installing experimental NVIDIA Container Runtime: %v", err)
	}

	return nil
}

func newNvidiaContainerRuntimeInstaller() *executable {
	target := executableTarget{
		dotfileName: nvidiaContainerRuntimeTarget,
		wrapperName: nvidiaContainerRuntimeWrapper,
	}
	return newRuntimeInstaller(nvidiaContainerRuntimeSource, target, nil)
}

func newNvidiaContainerRuntimeExperimentalInstaller(libraryRoot string) *executable {
	target := executableTarget{
		dotfileName: nvidiaExperimentalContainerRuntimeTarget,
		wrapperName: nvidiaExperimentalContainerRuntimeWrapper,
	}

	env := make(map[string]string)
	if libraryRoot != "" {
		env["LD_LIBRARY_PATH"] = strings.Join([]string{libraryRoot, "$LD_LIBRARY_PATH"}, ":")
	}
	return newRuntimeInstaller(nvidiaExperimentalContainerRuntimeSource, target, env)
}

func newRuntimeInstaller(source string, target executableTarget, env map[string]string) *executable {
	preLines := []string{
		"",
		"cat /proc/modules | grep -e \"^nvidia \" >/dev/null 2>&1",
		"if [ \"${?}\" != \"0\" ]; then",
		"	echo \"nvidia driver modules are not yet loaded, invoking runc directly\"",
		"	exec runc \"$@\"",
		"fi",
		"",
	}

	runtimeEnv := make(map[string]string)
	runtimeEnv["XDG_CONFIG_HOME"] = filepath.Join(destDirPattern, ".config")
	for k, v := range env {
		runtimeEnv[k] = v
	}

	r := executable{
		source:   source,
		target:   target,
		env:      runtimeEnv,
		preLines: preLines,
	}

	return &r
}

func findLibraryRoot(root string) (string, error) {
	libnvidiamlPath, err := findManagementLibrary(root)
	if err != nil {
		return "", fmt.Errorf("error locating NVIDIA management library: %v", err)
	}

	return filepath.Dir(libnvidiamlPath), nil
}

func findManagementLibrary(root string) (string, error) {
	return findLibrary(root, "libnvidia-ml.so")
}
