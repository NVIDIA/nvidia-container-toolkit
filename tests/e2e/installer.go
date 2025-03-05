/*
* Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
*     http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package e2e

import (
	"bytes"
	"fmt"
	"text/template"
)

// dockerInstallTemplate is a template for installing the NVIDIA Container Toolkit
// on a host using Docker.
var dockerInstallTemplate = `
#! /usr/bin/env bash
set -xe

: ${IMAGE:={{.Image}}}

# Create a temporary directory
TEMP_DIR="/tmp/ctk_e2e.$(date +%s)_$RANDOM"
mkdir -p "$TEMP_DIR"

# Given that docker has an init function that checks for the existence of the
# nvidia-container-toolkit, we need to create a symlink to the nvidia-container-runtime-hook
# in the /usr/bin directory.
# See https://github.com/moby/moby/blob/20a05dabf44934447d1a66cdd616cc803b81d4e2/daemon/nvidia_linux.go#L32-L46
sudo rm -f /usr/bin/nvidia-container-runtime-hook
sudo ln -s "$TEMP_DIR/toolkit/nvidia-container-runtime-hook" /usr/bin/nvidia-container-runtime-hook

docker run --pid=host --rm -i --privileged	\
	-v /:/host	\
	-v /var/run/docker.sock:/var/run/docker.sock	\
	-v "$TEMP_DIR:$TEMP_DIR"	\
	-v /etc/docker:/config-root	\
	${IMAGE}	\
	--root "$TEMP_DIR"	\
	--runtime=docker	\
	--config=/config-root/daemon.json	\
	--driver-root=/	\
	--no-daemon	\
	--restart-mode=systemd
`

type ToolkitInstaller struct {
	runner   Runner
	template string

	Image string
}

type installerOption func(*ToolkitInstaller)

func WithRunner(r Runner) installerOption {
	return func(i *ToolkitInstaller) {
		i.runner = r
	}
}

func WithImage(image string) installerOption {
	return func(i *ToolkitInstaller) {
		i.Image = image
	}
}

func WithTemplate(template string) installerOption {
	return func(i *ToolkitInstaller) {
		i.template = template
	}
}

func NewToolkitInstaller(opts ...installerOption) (*ToolkitInstaller, error) {
	i := &ToolkitInstaller{
		runner:   localRunner{},
		template: dockerInstallTemplate,
	}

	for _, opt := range opts {
		opt(i)
	}

	if i.Image == "" {
		return nil, fmt.Errorf("image is required")
	}

	return i, nil
}

func (i *ToolkitInstaller) Install() error {
	// Parse the combined template
	tmpl, err := template.New("installScript").Parse(i.template)
	if err != nil {
		return fmt.Errorf("error parsing template: %w", err)
	}

	// Execute the template
	var renderedScript bytes.Buffer
	err = tmpl.Execute(&renderedScript, i)
	if err != nil {
		return fmt.Errorf("error executing template: %w", err)
	}

	_, _, err = i.runner.Run(renderedScript.String())
	return err
}
