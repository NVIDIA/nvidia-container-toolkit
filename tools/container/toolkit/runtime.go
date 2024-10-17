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

package toolkit

import (
	"fmt"
	"path/filepath"

	"github.com/NVIDIA/nvidia-container-toolkit/tools/container/operator"
)

const (
	nvidiaContainerRuntimeSource = "/usr/bin/nvidia-container-runtime"
)

// installContainerRuntimes sets up the NVIDIA container runtimes, copying the executables
// and implementing the required wrapper
func installContainerRuntimes(toolkitDir string, driverRoot string) error {
	runtimes := operator.GetRuntimes()
	for _, runtime := range runtimes {
		r := newNvidiaContainerRuntimeInstaller(runtime.Path)

		_, err := r.install(toolkitDir)
		if err != nil {
			return fmt.Errorf("error installing NVIDIA container runtime: %v", err)
		}
	}
	return nil
}

// newNVidiaContainerRuntimeInstaller returns a new executable installer for the NVIDIA container runtime.
// This installer will copy the specified source executable to the toolkit directory.
// The executable is copied to a file with the same name as the source, but with a ".real" suffix and a wrapper is
// created to allow for the configuration of the runtime environment.
func newNvidiaContainerRuntimeInstaller(source string) *executable {
	wrapperName := filepath.Base(source)
	target := executableTarget{
		wrapperName: wrapperName,
	}
	return newRuntimeInstaller(source, target, nil)
}

func newRuntimeInstaller(source string, target executableTarget, env map[string]string) *executable {
	runtimeEnv := make(map[string]string)
	runtimeEnv["XDG_CONFIG_HOME"] = filepath.Join(destDirPattern, ".config")
	for k, v := range env {
		runtimeEnv[k] = v
	}
	r := executable{
		source: source,
		target: target,
		envm:   runtimeEnv,
	}
	return &r
}
