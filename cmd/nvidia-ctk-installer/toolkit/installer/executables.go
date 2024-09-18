/**
# SPDX-FileCopyrightText: Copyright (c) 2025 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
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
**/

package installer

import (
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/container/operator"
	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/toolkit/installer/wrappercore"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
)

type executable struct {
	requiresKernelModule bool
	path                 string
	symlink              string
	argv                 []string
	envm                 map[string]string
}

func (t *ToolkitInstaller) collectExecutables(destDir string) ([]Installer, error) {
	configFilePath := t.ConfigFilePath(destDir)

	executables := []executable{
		{
			path: "nvidia-ctk",
		},
		{
			path: "nvidia-cdi-hook",
		},
	}
	for _, runtime := range operator.GetRuntimes() {
		e := executable{
			path:                 runtime.Path,
			requiresKernelModule: true,
			envm: map[string]string{
				config.FilePathOverrideEnvVar: configFilePath,
			},
		}
		executables = append(executables, e)
	}
	executables = append(executables,
		executable{
			path: "nvidia-container-cli",
			envm: map[string]string{"<LD_LIBRARY_PATH": destDir},
		},
	)

	executables = append(executables,
		executable{
			path:    "nvidia-container-runtime-hook",
			symlink: "nvidia-container-toolkit",
			envm: map[string]string{
				config.FilePathOverrideEnvVar: configFilePath,
			},
		},
	)

	var installers []Installer
	for _, executable := range executables {
		executablePath, err := t.artifactRoot.findExecutable(executable.path)
		if err != nil {
			if t.ignoreErrors {
				log.Errorf("Ignoring error: %v", err)
				continue
			}
			return nil, err
		}

		w := &wrapper{
			Source:             executablePath,
			WrapperProgramPath: t.wrapperProgramPath,
			Config: wrappercore.WrapperConfig{
				Argv: executable.argv,
				Envm: map[string]string{
					"<PATH": destDir,
				},
				RequiresKernelModule: executable.requiresKernelModule,
			},
		}
		for k, v := range executable.envm {
			w.Config.Envm[k] = v
		}

		if len(t.defaultRuntimeExecutablePath) > 0 {
			w.Config.DefaultRuntimeExecutablePath = t.defaultRuntimeExecutablePath
		} else {
			w.Config.DefaultRuntimeExecutablePath = "runc"
		}

		installers = append(installers, w)

		if executable.symlink == "" {
			continue
		}
		link := symlink{
			linkname: executable.symlink,
			target:   filepath.Base(executablePath),
		}
		installers = append(installers, link)
	}

	return installers, nil
}
