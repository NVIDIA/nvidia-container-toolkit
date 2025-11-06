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
	"strings"
	"text/template"
)

var prepareInstallerCacheTemplate = `
set -xe

mkdir -p {{.CacheDir}}

docker run --rm -v {{.CacheDir}}:/cache --entrypoint="sh" {{.ToolkitImage}}-packaging -c "cp -p -R /artifacts/* /cache/"
`

var installFromImageTemplate = `
set -xe

arch="$(uname -m)"
case "${arch##*-}" in
	x86_64 | amd64) ARCH='amd64' ;;
	ppc64el | ppc64le) ARCH='ppc64le' ;;
	aarch64 | arm64) ARCH='arm64' ;;
	*) echo "unsupported architecture" ; exit 1 ;;
esac

cd {{.CacheDir}}/packages/ubuntu18.04/${ARCH}

{{if .WithSudo }}sudo {{end}}dpkg -i libnvidia-container1_*_${ARCH}.deb \
	libnvidia-container-tools_*_${ARCH}.deb \
	nvidia-container-toolkit-base_*_${ARCH}.deb \
	nvidia-container-toolkit_*_${ARCH}.deb

cd -

nvidia-container-cli --version
`

type ToolkitInstaller struct {
	ToolkitImage string
	CacheDir     string
}

type installerOption func(*ToolkitInstaller)

func WithToolkitImage(image string) installerOption {
	return func(i *ToolkitInstaller) {
		i.ToolkitImage = image
	}
}

func WithCacheDir(cacheDir string) installerOption {
	return func(i *ToolkitInstaller) {
		i.CacheDir = cacheDir
	}
}

func NewToolkitInstaller(opts ...installerOption) (*ToolkitInstaller, error) {
	i := &ToolkitInstaller{}

	for _, opt := range opts {
		opt(i)
	}

	if i.ToolkitImage == "" {
		return nil, fmt.Errorf("image is required")
	}

	return i, nil
}

// PrepareCache ensures that the installer (package) cache is created on the runner.
// The can be used to ensure that docker is not REQUIRED in an inner container.
func (i *ToolkitInstaller) PrepareCache(runner Runner) (string, string, error) {
	if i.ToolkitImage == "disabled:disabled" {
		return "", "", nil
	}
	renderedScript, err := i.renderScript(prepareInstallerCacheTemplate, false)
	if err != nil {
		return "", "", err
	}

	return runner.Run(renderedScript)
}

func (i *ToolkitInstaller) Install(runner Runner) (string, string, error) {
	uid, _, err := runner.Run("id -u")
	if err != nil {
		return "", "", err
	}
	withSudo := false
	if strings.TrimSpace(uid) != "0" {
		withSudo = true
	}
	renderedScript, err := i.renderScript(installFromImageTemplate, withSudo)
	if err != nil {
		return "", "", err
	}

	return runner.Run(renderedScript)
}

func (i *ToolkitInstaller) renderScript(scriptTemplate string, withSudo bool) (string, error) {
	// Parse the combined template
	tmpl, err := template.New("template").Parse(scriptTemplate)
	if err != nil {
		return "", fmt.Errorf("error parsing template: %w", err)
	}

	templateInfo := struct {
		*ToolkitInstaller
		WithSudo bool
	}{
		ToolkitInstaller: i,
		WithSudo:         withSudo,
	}
	// Execute the template
	var renderedScript bytes.Buffer
	err = tmpl.Execute(&renderedScript, templateInfo)
	if err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}

	return renderedScript.String(), nil
}
