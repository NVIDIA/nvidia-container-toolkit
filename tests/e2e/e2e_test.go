/*
 * SPDX-FileCopyrightText: Copyright (c) 2025 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
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
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Test context
var (
	ctx context.Context

	installCTK bool

	ImageRepo string
	ImageTag  string

	sshKey      string
	sshUser     string
	host        string
	sshPort     string
	cwd         string
	packagePath string

	runner Runner
)

func TestMain(t *testing.T) {
	suiteName := "E2E NVIDIA Container Toolkit"

	RegisterFailHandler(Fail)

	ctx = context.Background()
	getTestEnv()

	RunSpecs(t,
		suiteName,
	)
}

// BeforeSuite runs before the test suite
var _ = BeforeSuite(func() {
	runner = NewRunner(
		WithHost(host),
		WithPort(sshPort),
		WithSshKey(sshKey),
		WithSshUser(sshUser),
	)

	if installCTK {
		installer, err := NewToolkitInstaller(
			WithRunner(runner),
			WithImage(ImageRepo+":"+ImageTag),
			WithTemplate(dockerInstallTemplate),
		)
		Expect(err).ToNot(HaveOccurred())

		err = installer.Install()
		Expect(err).ToNot(HaveOccurred())
	}

	_, _, err := runner.Run("docker pull ubuntu")
	Expect(err).ToNot(HaveOccurred())

	_, _, err = runner.Run("docker pull nvcr.io/nvidia/k8s/cuda-sample:vectoradd-cuda12.5.0")
	Expect(err).ToNot(HaveOccurred())

	_, _, err = runner.Run("docker pull nvcr.io/nvidia/cuda:12.8.0-base-ubi8")
	Expect(err).ToNot(HaveOccurred())
})

// getTestEnv gets the test environment variables
func getTestEnv() {
	defer GinkgoRecover()
	var err error

	_, thisFile, _, _ := runtime.Caller(0)
	packagePath = filepath.Dir(thisFile)

	installCTK = getBoolEnvVar("INSTALL_CTK", false)

	ImageRepo = os.Getenv("E2E_IMAGE_REPO")
	Expect(ImageRepo).NotTo(BeEmpty(), "E2E_IMAGE_REPO environment variable must be set")

	ImageTag = os.Getenv("E2E_IMAGE_TAG")
	Expect(ImageTag).NotTo(BeEmpty(), "E2E_IMAGE_TAG environment variable must be set")

	sshKey = os.Getenv("SSH_KEY")
	Expect(sshKey).NotTo(BeEmpty(), "SSH_KEY environment variable must be set")

	sshUser = os.Getenv("SSH_USER")
	Expect(sshUser).NotTo(BeEmpty(), "SSH_USER environment variable must be set")

	host = os.Getenv("REMOTE_HOST")
	Expect(host).NotTo(BeEmpty(), "REMOTE_HOST environment variable must be set")

	sshPort = os.Getenv("REMOTE_PORT")
	Expect(sshPort).NotTo(BeEmpty(), "REMOTE_PORT environment variable must be set")

	// Get current working directory
	cwd, err = os.Getwd()
	Expect(err).NotTo(HaveOccurred())
}

// getBoolEnvVar returns the boolean value of the environment variable or the default value if not set.
func getBoolEnvVar(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return boolValue
}
