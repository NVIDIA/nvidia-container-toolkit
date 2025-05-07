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
	"context"
	"errors"
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
})

// getTestEnv gets the test environment variables
func getTestEnv() {
	defer GinkgoRecover()
	var err error

	_, thisFile, _, _ := runtime.Caller(0)
	packagePath = filepath.Dir(thisFile)

	installCTK = getEnvVarOrDefault("E2E_INSTALL_CTK", true)

	if installCTK {
		ImageRepo = os.Getenv("E2E_IMAGE_REPO")
		Expect(ImageRepo).NotTo(BeEmpty(), "E2E_IMAGE_REPO environment variable must be set")

		ImageTag = os.Getenv("E2E_IMAGE_TAG")
		Expect(ImageTag).NotTo(BeEmpty(), "E2E_IMAGE_TAG environment variable must be set")
	}

	sshKey = os.Getenv("E2E_SSH_KEY")
	Expect(sshKey).NotTo(BeEmpty(), "E2E_SSH_KEY environment variable must be set")

	sshUser = os.Getenv("E2E_SSH_USER")
	Expect(sshUser).NotTo(BeEmpty(), "E2E_SSH_USER environment variable must be set")

	host = os.Getenv("E2E_SSH_HOST")
	Expect(host).NotTo(BeEmpty(), "E2E_SSH_HOST environment variable must be set")

	sshPort = getEnvVarOrDefault("E2E_SSH_PORT", "22")

	// Get current working directory
	cwd, err = os.Getwd()
	Expect(err).NotTo(HaveOccurred())
}

func getEnvVarAs[T any](key string) (T, error) {
	var zero T
	value := os.Getenv(key)
	if value == "" {
		return zero, errors.New("env var not set")
	}

	switch any(zero).(type) {
	case bool:
		v, err := strconv.ParseBool(value)
		if err != nil {
			return zero, err
		}
		return any(v).(T), nil
	case int:
		v, err := strconv.Atoi(value)
		if err != nil {
			return zero, err
		}
		return any(v).(T), nil
	case string:
		return any(value).(T), nil
	default:
		return zero, errors.New("unsupported type")
	}
}

func getEnvVarOrDefault[T any](key string, defaultValue T) T {
	val, err := getEnvVarAs[T](key)
	if err != nil {
		return defaultValue
	}
	return val
}
