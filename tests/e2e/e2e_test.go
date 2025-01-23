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
	"context"
	"fmt"
	"os/exec"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Test context
var (
	ctx context.Context
)

func TestMain(t *testing.T) {
	suiteName := "NVIDIA Container Toolkit E2E"

	RegisterFailHandler(Fail)
	RunSpecs(t,
		suiteName,
	)
}

// BeforeSuite runs before the test suite
var _ = BeforeSuite(func() {
	ctx = context.Background()
})

func runScript(script string) (string, error) {
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
