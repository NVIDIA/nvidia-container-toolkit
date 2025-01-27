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
	"text/template"
	"time"

	"golang.org/x/crypto/ssh"
)

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

type Installer struct {
	Image    string
	Template string

	SshKey     string
	SshUser    string
	RemoteHost string
}

func newInstaller(templateStr, image, sshKey, sshUser, remoteHost string) (Installer, error) {
	i := Installer{
		Image:      image,
		Template:   templateStr,
		SshKey:     sshKey,
		SshUser:    sshUser,
		RemoteHost: remoteHost,
	}

	return i, nil
}

func (i *Installer) Install() error {
	// Parse the combined template
	tmpl, err := template.New("dockerScript").Parse(i.Template)
	if err != nil {
		return fmt.Errorf("error parsing template: %w", err)
	}

	// Execute the template
	var renderedScript bytes.Buffer
	err = tmpl.Execute(&renderedScript, i)
	if err != nil {
		return fmt.Errorf("error executing template: %w", err)
	}

	var s Script
	switch {
	case i.RemoteHost == "":
		s = localRun{}
	default:
		s = remoteRun{i.SshKey, i.SshUser, i.RemoteHost}
	}

	_, err = s.Run(renderedScript.String())
	return err
}

type Script interface {
	Run(script string) (string, error)
}

type localRun struct{}
type remoteRun struct {
	SshKey     string
	SshUser    string
	RemoteHost string
}

func (l localRun) Run(script string) (string, error) {
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
		return "", fmt.Errorf("script execution failed: %v\nSTDOUT: %s\nSTDERR: %s", err, stdout.String(), stderr.String())
	}

	// Return the captured stdout and nil error
	return stdout.String(), nil
}

func (r remoteRun) Run(script string) (string, error) {
	// Create a new SSH connection
	client, err := connectOrDie(r.SshKey, r.SshUser, r.RemoteHost)
	if err != nil {
		return "", fmt.Errorf("failed to connect to %s: %v", r.RemoteHost, err)
	}
	defer client.Close()

	// Create a session
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	// Run the script
	err = session.Run(script)
	if err != nil {
		return "", fmt.Errorf("script execution failed: %v\nSTDOUT: %s\nSTDERR: %s", err, stdout.String(), stderr.String())
	}

	// Return stdout as string if no errors
	return stdout.String(), nil
}

// createSshClient creates a ssh client, and retries if it fails to connect
func connectOrDie(sshKey, sshUser, remoteHost string) (*ssh.Client, error) {
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
		client, err = ssh.Dial("tcp", remoteHost+":22", sshConfig)
		if err == nil {
			return client, nil // Connection succeeded, return the client.
		}
		connectionFailed = true
		// Sleep for a brief moment before retrying.
		// You can adjust the duration based on your requirements.
		time.Sleep(1 * time.Second)
	}

	if connectionFailed {
		return nil, fmt.Errorf("failed to connect to %s after 10 retries, giving up", remoteHost)
	}

	return client, nil
}
