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
	"bytes"
	"fmt"
	"html/template"
	"io"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/container/operator"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
)

type executable struct {
	requiresKernelModule bool
	path                 string
	symlink              string
	env                  map[string]string
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
			env: map[string]string{
				config.FilePathOverrideEnvVar: configFilePath,
			},
		}
		executables = append(executables, e)
	}
	executables = append(executables,
		executable{
			path: "nvidia-container-cli",
			env:  map[string]string{"LD_LIBRARY_PATH": destDir + ":$LD_LIBRARY_PATH"},
		},
	)

	executables = append(executables,
		executable{
			path:    "nvidia-container-runtime-hook",
			symlink: "nvidia-container-toolkit",
			env: map[string]string{
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

		wrappedExecutableFilename := filepath.Base(executablePath)
		dotRealFilename := wrappedExecutableFilename + ".real"

		w := &wrapper{
			Source:            executablePath,
			WrappedExecutable: dotRealFilename,
			CheckModules:      executable.requiresKernelModule,
			Envvars: map[string]string{
				"PATH": strings.Join([]string{destDir, "$PATH"}, ":"),
			},
		}
		for k, v := range executable.env {
			w.Envvars[k] = v
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

type wrapper struct {
	Source            string
	Envvars           map[string]string
	WrappedExecutable string
	CheckModules      bool
}

type render struct {
	*wrapper
	DestDir string
}

func (w *wrapper) Install(destDir string) error {
	// Copy the executable with a .real extension.
	mode, err := installFile(w.Source, filepath.Join(destDir, w.WrappedExecutable))
	if err != nil {
		return err
	}

	// Create a wrapper file.
	r := render{
		wrapper: w,
		DestDir: destDir,
	}
	content, err := r.render()
	if err != nil {
		return fmt.Errorf("failed to render wrapper: %w", err)
	}
	wrapperFile := filepath.Join(destDir, filepath.Base(w.Source))
	return installContent(content, wrapperFile, mode|0111)
}

func (w *render) render() (io.Reader, error) {
	wrapperTemplate := `#! /bin/sh
{{- if (.CheckModules) }}
cat /proc/modules | grep -e "^nvidia " >/dev/null 2>&1
if [ "${?}" != "0" ]; then
	echo "nvidia driver modules are not yet loaded, invoking runc directly"
	exec runc "$@"
fi
{{- end }}
{{- range $key, $value := .Envvars }}
{{$key}}={{$value}} \
{{- end }}
	{{ .DestDir }}/{{ .WrappedExecutable }} \
		"$@"
`

	var content bytes.Buffer
	tmpl, err := template.New("wrapper").Parse(wrapperTemplate)
	if err != nil {
		return nil, err
	}
	if err := tmpl.Execute(&content, w); err != nil {
		return nil, err
	}

	return &content, nil
}
