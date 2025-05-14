/**
# SPDX-FileCopyrightText: Copyright (c) 2025 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
**/

package e2e

import (
	"context"
	"errors"
	"os"
	"strconv"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Test context
var (
	ctx context.Context

	installCTK bool

	imageName string
	imageTag  string

	sshKey  string
	sshUser string
	sshHost string
	sshPort string
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

// getTestEnv gets the test environment variables
func getTestEnv() {
	defer GinkgoRecover()

	installCTK = getEnvVarOrDefault("E2E_INSTALL_CTK", false)

	if installCTK {
		imageName = getRequiredEnvvar[string]("E2E_IMAGE_NAME")

		imageTag = getRequiredEnvvar[string]("E2E_IMAGE_TAG")

	}

	sshKey = getRequiredEnvvar[string]("E2E_SSH_KEY")
	sshUser = getRequiredEnvvar[string]("E2E_SSH_USER")
	sshHost = getRequiredEnvvar[string]("E2E_SSH_HOST")

	sshPort = getEnvVarOrDefault("E2E_SSH_PORT", "22")
}

// getRequiredEnvvar returns the specified envvar if set or raises an error.
func getRequiredEnvvar[T any](key string) T {
	v, err := getEnvVarAs[T](key)
	Expect(err).To(BeNil(), "required environement variable not set", key)
	return v
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
