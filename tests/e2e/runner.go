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
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	outerContainerTemplate = `docker run -d --name {{.ContainerName}} --privileged --runtime=nvidia \
    -e NVIDIA_VISIBLE_DEVICES=runtime.nvidia.com/gpu=all \
    -e NVIDIA_DRIVER_CAPABILITIES=all \
	{{ range $i, $a := .AdditionalArguments -}}
	{{ $a }} \
	{{ end -}}
	ubuntu sleep infinity`

	installDockerTemplate = `
	export DEBIAN_FRONTEND=noninteractive
	
	# Add Docker official GPG key:
	apt-get update
	apt-get install -y ca-certificates curl apt-utils gnupg2
	install -m 0755 -d /etc/apt/keyrings
	curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
	chmod a+r /etc/apt/keyrings/docker.asc
	
	# Add the repository to Apt sources:
	echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo \"${UBUNTU_CODENAME:-$VERSION_CODENAME}\") stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
	apt-get update
	
	apt-get install -y docker-ce docker-ce-cli containerd.io
	
	# start dockerd in the background
	dockerd &
	
	# wait for dockerd to be ready with timeout
	timeout=30
	elapsed=0
	while ! docker info > /dev/null 2>&1 && [ $elapsed -lt $timeout ]; do
		echo "Waiting for dockerd to be ready..."
		sleep 1
		elapsed=$((elapsed + 1))
	done
	if [ $elapsed -ge $timeout ]; then
		echo "Docker failed to start within $timeout seconds"
		exit 1
	fi
	`

	installCTKTemplate = `
	# Create a temporary directory and rootfs path
	TMPDIR="$(mktemp -d)"
	
	# Expose TMPDIR for the child namespace
	export TMPDIR
	
	docker run --rm -v ${TMPDIR}:/host-tmpdir --entrypoint="sh" {{.ToolkitImage}}-packaging -c "cp -p -R /artifacts/* /host-tmpdir/"
	dpkg -i ${TMPDIR}/packages/ubuntu18.04/amd64/libnvidia-container1_*_amd64.deb ${TMPDIR}/packages/ubuntu18.04/amd64/nvidia-container-toolkit-base_*_amd64.deb ${TMPDIR}/packages/ubuntu18.04/amd64/libnvidia-container-tools_*_amd64.deb

	nvidia-ctk --version
	`
)

type localRunner struct{}
type remoteRunner struct {
	sshKey  string
	sshUser string
	host    string
	port    string
}

type nestedContainerRunner struct {
	runner        Runner
	containerName string
}

type runnerOption func(*remoteRunner)
type nestedContainerOption func(*nestedContainerRunner)

type Runner interface {
	Run(script string) (string, string, error)
}

func WithSshKey(key string) runnerOption {
	return func(r *remoteRunner) {
		r.sshKey = key
	}
}

func WithSshUser(user string) runnerOption {
	return func(r *remoteRunner) {
		r.sshUser = user
	}
}

func WithHost(host string) runnerOption {
	return func(r *remoteRunner) {
		r.host = host
	}
}

func WithPort(port string) runnerOption {
	return func(r *remoteRunner) {
		r.port = port
	}
}

func NewRunner(opts ...runnerOption) Runner {
	r := &remoteRunner{}
	for _, opt := range opts {
		opt(r)
	}

	// If the Host is empty, return a local runner
	if r.host == "" {
		return localRunner{}
	}

	// Otherwise, return a remote runner
	return r
}

// NewNestedContainerRunner creates a new nested container runner.
// A nested container runs a container inside another container based on a
// given runner (remote or local).
func NewNestedContainerRunner(runner Runner, installCTK bool, image string, containerName string, opts ...nestedContainerOption) (Runner, error) {
	additionalContainerArguments := []string{}

	// If a container with the same name exists from a previous test run, remove it first.
	// Ignore errors as container might not exist
	_, _, err := runner.Run(fmt.Sprintf("docker rm -f %s 2>/dev/null || true", containerName)) //nolint:errcheck
	if err != nil {
		return nil, fmt.Errorf("failed to remove container: %w", err)
	}

	// If installCTK is true, install the toolkit on the host. before creating
	// the nested container.
	if installCTK {
		installer, err := NewToolkitInstaller(
			WithRunner(runner),
			WithImage(image),
			WithTemplate(dockerInstallTemplate),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create toolkit installer: %w", err)
		}

		err = installer.Install()
		if err != nil {
			return nil, fmt.Errorf("failed to install toolkit: %w", err)
		}
	} else {
		// If installCTK is false, we use the preinstalled toolkit.
		output, _, err := runner.Run("ls /lib/**/libnvidia-container*.so.*.*")
		if err != nil {
			return nil, fmt.Errorf("failed to list toolkit libraries: %w", err)
		}

		output = strings.TrimSpace(output)
		if output == "" {
			return nil, fmt.Errorf("no toolkit libraries found") //nolint:goerr113
		}

		for _, lib := range strings.Split(output, "\n") {
			additionalContainerArguments = append(additionalContainerArguments, "-v "+lib+":"+lib)
		}

		// Look for NVIDIA binaries in standard locations and mount them as volumes
		nvidiaBinaries := []string{
			"nvidia-container-cli",
			"nvidia-container-runtime",
			"nvidia-container-runtime-hook",
			"nvidia-ctk",
			"nvidia-cdi-hook",
			"nvidia-container-runtime.cdi",
			"nvidia-container-runtime.legacy",
		}

		searchPaths := []string{
			"/usr/bin",
			"/usr/sbin",
			"/usr/local/bin",
			"/usr/local/sbin",
		}

		for _, binary := range nvidiaBinaries {
			for _, searchPath := range searchPaths {
				binaryPath := searchPath + "/" + binary
				// Check if the binary exists at this path
				checkCmd := fmt.Sprintf("test -f %s && echo 'exists'", binaryPath)
				output, _, err := runner.Run(checkCmd)
				if err == nil && strings.TrimSpace(output) == "exists" {
					// Binary found, add it as a volume mount
					additionalContainerArguments = append(additionalContainerArguments,
						fmt.Sprintf("-v %s:%s", binaryPath, binaryPath))
					break // Move to the next binary once found
				}
			}
		}
	}

	// Launch the container in detached mode.
	var outerContainerScriptBuilder strings.Builder
	outerContainerTemplate, err := template.New("outerContainer").Parse(outerContainerTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse start container template: %w", err)
	}
	err = outerContainerTemplate.Execute(&outerContainerScriptBuilder, struct {
		ContainerName       string
		AdditionalArguments []string
	}{
		ContainerName:       containerName,
		AdditionalArguments: additionalContainerArguments,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to execute start container template: %w", err)
	}

	outerContainerScript := outerContainerScriptBuilder.String()
	_, _, err = runner.Run(outerContainerScript)
	if err != nil {
		return nil, fmt.Errorf("failed to run start container script: %w", err)
	}

	// install docker in the nested container
	_, _, err = runner.Run(fmt.Sprintf("docker exec -u root "+containerName+" bash -c '%s'", installDockerTemplate))
	if err != nil {
		return nil, fmt.Errorf("failed to install docker: %w", err)
	}

	if installCTK {
		// Install nvidia-container-cli in the container.
		tmpl, err := template.New("toolkitInstall").Parse(installCTKTemplate)
		if err != nil {
			return nil, fmt.Errorf("failed to parse installCTK template: %w", err)
		}

		var toolkitInstall strings.Builder
		err = tmpl.Execute(&toolkitInstall, struct {
			ToolkitImage string
		}{
			ToolkitImage: image,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to execute installCTK template: %w", err)
		}

		_, _, err = runner.Run(fmt.Sprintf("docker exec -u root "+containerName+" bash -c '%s'", toolkitInstall.String()))
		if err != nil {
			return nil, fmt.Errorf("failed to install nvidia-container-cli: %w", err)
		}
	}

	nc := &nestedContainerRunner{
		runner:        runner,
		containerName: containerName,
	}

	return nc, nil
}

func (l localRunner) Run(script string) (string, string, error) {
	// Create a command to run the script using bash
	cmd := exec.Command("bash", "-c", script)

	// Buffer to capture standard output
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	// Buffer to capture standard error
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Run the command
	err := cmd.Run()
	if err != nil {
		return "", "", fmt.Errorf("script execution failed: %v\nSTDOUT: %s\nSTDERR: %s", err, stdout.String(), stderr.String())
	}

	// Return the captured stdout and nil error
	return stdout.String(), "", nil
}

func (r remoteRunner) Run(script string) (string, string, error) {
	// Create a new SSH connection
	client, err := connectOrDie(r.sshKey, r.sshUser, r.host, r.port)
	if err != nil {
		return "", "", fmt.Errorf("failed to connect to %s: %v", r.host, err)
	}
	defer client.Close()

	// Create a session
	session, err := client.NewSession()
	if err != nil {
		return "", "", fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	// Run the script
	err = session.Run(script)
	if err != nil {
		return "", stderr.String(), fmt.Errorf("script execution failed: %v\nSTDOUT: %s\nSTDERR: %s", err, stdout.String(), stderr.String())
	}

	// Return stdout as string if no errors
	return stdout.String(), "", nil
}

func (r nestedContainerRunner) Run(script string) (string, string, error) {
	return r.runner.Run(fmt.Sprintf("docker exec -u root "+r.containerName+" bash -c '%s'", script))
}

// createSshClient creates a ssh client, and retries if it fails to connect
func connectOrDie(sshKey, sshUser, host, port string) (*ssh.Client, error) {
	var client *ssh.Client
	var err error
	key, err := os.ReadFile(sshKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %v", err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}
	sshConfig := &ssh.ClientConfig{
		User: sshUser,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	connectionFailed := false
	for i := 0; i < 20; i++ {
		client, err = ssh.Dial("tcp", host+":"+port, sshConfig)
		if err == nil {
			return client, nil // Connection succeeded, return the client.
		}
		connectionFailed = true
		// Sleep for a brief moment before retrying.
		// You can adjust the duration based on your requirements.
		time.Sleep(1 * time.Second)
	}

	if connectionFailed {
		return nil, fmt.Errorf("failed to connect to %s after 10 retries, giving up", host)
	}

	return client, nil
}
