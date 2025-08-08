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
	installPrerequisitesScript = `
	export DEBIAN_FRONTEND=noninteractive
	apt-get update && apt-get install -y curl gnupg2
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
func NewNestedContainerRunner(runner Runner, installCTK bool, containerName string, cacheDir string) (Runner, error) {
	// If a container with the same name exists from a previous test run, remove it first.
	// Ignore errors as container might not exist
	_, _, err := runner.Run(fmt.Sprintf("docker rm -f %s 2>/dev/null || true", containerName))
	if err != nil {
		return nil, fmt.Errorf("failed to remove container: %w", err)
	}

	var additionalContainerArguments []string

	if cacheDir != "" {
		additionalContainerArguments = append(additionalContainerArguments,
			"-v "+cacheDir+":"+cacheDir+":ro",
		)
	}

	if !installCTK {
		// If installCTK is false, we use the preinstalled toolkit.
		// This means we need to add toolkit libraries and binaries from the "host"

		// TODO: This should be updated for other distributions and other components of the toolkit.
		output, _, err := runner.Run("ls /lib/**/libnvidia-container*.so.*.*")
		if err != nil {
			return nil, fmt.Errorf("failed to list toolkit libraries: %w", err)
		}

		output = strings.TrimSpace(output)
		if output == "" {
			return nil, fmt.Errorf("no toolkit libraries found")
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
	container := outerContainer{
		Name:                containerName,
		AdditionalArguments: additionalContainerArguments,
	}

	script, err := container.Render()
	if err != nil {
		return nil, err
	}
	_, _, err = runner.Run(script)
	if err != nil {
		return nil, fmt.Errorf("failed to run start container script: %w", err)
	}

	inContainer := &nestedContainerRunner{
		runner:        runner,
		containerName: containerName,
	}

	_, _, err = inContainer.Run(installPrerequisitesScript)
	if err != nil {
		return nil, fmt.Errorf("failed to install docker: %w", err)
	}

	return inContainer, nil
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

// Run runs teh specified script in the container using the docker exec command.
// The script is is run as the root user.
func (r nestedContainerRunner) Run(script string) (string, string, error) {
	return r.runner.Run(`docker exec -u root "` + r.containerName + `" bash -c '` + script + `'`)
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

// outerContainerTemplate represents a template to start a container with
// a name specified.
// The container is given access to all NVIDIA gpus by explicitly using the
// nvidia runtime and the `runtime.nvidia.com/gpu=all` device to trigger JIT
// CDI spec generation.
// The template also allows for additional arguments to be specified.
type outerContainer struct {
	Name                string
	AdditionalArguments []string
}

func (o *outerContainer) Render() (string, error) {
	tmpl, err := template.New("startContainer").Parse(`docker run -d --name {{.Name}} --privileged --runtime=nvidia \
-e NVIDIA_VISIBLE_DEVICES=runtime.nvidia.com/gpu=all \
-e NVIDIA_DRIVER_CAPABILITIES=all \
{{ range $i, $a := .AdditionalArguments -}}
{{ $a }} \
{{ end -}}
ubuntu sleep infinity`)

	if err != nil {
		return "", err
	}

	var script strings.Builder
	err = tmpl.Execute(&script, o)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return script.String(), nil
}
