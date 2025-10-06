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

const (
	InstallUsingNVIDIACTKInstaller = "nvidia-ctk-installer"
	InstallUsingPackagingImage     = "packaging-image"
)

// dockerInstallTemplate is a template for installing the NVIDIA Container Toolkit
// on a host using Docker.
var dockerInstallTemplate = `
#! /usr/bin/env bash
set -xe

# if the TEMP_DIR is already set, use it
if [ -f /tmp/ctk_e2e_temp_dir.txt ]; then
    TEMP_DIR=$(cat /tmp/ctk_e2e_temp_dir.txt)
else
    TEMP_DIR="/tmp/ctk_e2e.$(date +%s)_$RANDOM"
    echo "$TEMP_DIR" > /tmp/ctk_e2e_temp_dir.txt
fi

# if TEMP_DIR does not exist, create it
if [ ! -d "$TEMP_DIR" ]; then
    mkdir -p "$TEMP_DIR"
fi

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
	{{.ToolkitImage}}	\
	--root "$TEMP_DIR"	\
	--runtime=docker	\
	--config=/config-root/daemon.json	\
	--driver-root=/	\
	--no-daemon	\
	--restart-mode=systemd
`

var installFromImageTemplate = `
# Create a temporary directory and rootfs path
TMPDIR="$(mktemp -d)"

# Expose TMPDIR for the child namespace
export TMPDIR

docker run --rm -v ${TMPDIR}:/host-tmpdir --entrypoint="sh" {{.ToolkitImage}}-packaging -c "cp -p -R /artifacts/* /host-tmpdir/"
dpkg -i ${TMPDIR}/packages/ubuntu18.04/amd64/libnvidia-container1_*_amd64.deb ${TMPDIR}/packages/ubuntu18.04/amd64/nvidia-container-toolkit-base_*_amd64.deb ${TMPDIR}/packages/ubuntu18.04/amd64/libnvidia-container-tools_*_amd64.deb

nvidia-container-cli --version
{{if .ConfigureDocker}}
nvidia-ctk runtime configure
{{end}}
{{if .RestartDocker}}
systemctl restart docker
{{end}}
`

type toolkitInstallerOptions struct {
	mode  string
	Image string
}

type toolkitInstallerTemplateOptions struct {
	ToolkitImage    string
	ConfigureDocker bool
	RestartDocker   bool
}

type ToolkitInstaller struct {
	template string
	toolkitInstallerTemplateOptions
}

type installerOption func(*toolkitInstallerOptions)

func WithImage(image string) installerOption {
	return func(i *toolkitInstallerOptions) {
		i.Image = image
	}
}

func WithMode(mode string) installerOption {
	return func(i *toolkitInstallerOptions) {
		i.mode = mode
	}
}

func NewToolkitInstaller(opts ...installerOption) (*ToolkitInstaller, error) {
	i := &toolkitInstallerOptions{
		mode: InstallUsingNVIDIACTKInstaller,
	}

	for _, opt := range opts {
		opt(i)
	}

	if i.Image == "" {
		return nil, fmt.Errorf("image is required")
	}

	template, err := i.getTemplate()
	if err != nil {
		return nil, err
	}

	ti := &ToolkitInstaller{
		template: template,
		toolkitInstallerTemplateOptions: toolkitInstallerTemplateOptions{
			ToolkitImage: i.Image,
		},
	}
	return ti, nil
}

func (i *ToolkitInstaller) Install(runner Runner) (string, string, error) {
	renderedScript, err := i.Render()
	if err != nil {
		return "", "", err
	}

	return runner.Run(renderedScript)
}

func (i *ToolkitInstaller) Render() (string, error) {
	// Parse the combined template
	tmpl, err := template.New("installScript").Parse(i.template)
	if err != nil {
		return "", fmt.Errorf("error parsing template: %w", err)
	}

	// Execute the template
	var renderedScript bytes.Buffer
	err = tmpl.Execute(&renderedScript, i.toolkitInstallerTemplateOptions)
	if err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}

	return renderedScript.String(), nil
}

func (i *toolkitInstallerOptions) getTemplate() (string, error) {
	switch i.mode {
	case InstallUsingNVIDIACTKInstaller:
		return dockerInstallTemplate, nil
	case InstallUsingPackagingImage:
		return installFromImageTemplate, nil
	default:
		return "", fmt.Errorf("unrecognized mode %q", i.mode)
	}
}
