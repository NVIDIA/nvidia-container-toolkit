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
	"time"

	"golang.org/x/crypto/ssh"
)

type localRunner struct{}
type remoteRunner struct {
	sshKey  string
	sshUser string
	host    string
	port    string
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
		return "", "", fmt.Errorf("script execution failed: %v\nSTDOUT: %s\nSTDERR: %s", err, stdout.String(), stderr.String())
	}

	// Return stdout as string if no errors
	return stdout.String(), "", nil
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
