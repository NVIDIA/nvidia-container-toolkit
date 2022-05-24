/**
# Copyright (c) 2022, NVIDIA CORPORATION.  All rights reserved.
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

package image

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/mod/semver"
)

const (
	envCUDAVersion      = "CUDA_VERSION"
	envNVRequirePrefix  = "NVIDIA_REQUIRE_"
	envNVRequireCUDA    = envNVRequirePrefix + "CUDA"
	envNVRequireJetpack = envNVRequirePrefix + "JETPACK"
	envNVDisableRequire = "NVIDIA_DISABLE_REQUIRE"
)

// CUDA represents a CUDA image that can be used for GPU computing. This wraps
// a map of environment variable to values that can be used to perform lookups
// such as requirements.
type CUDA map[string]string

// NewCUDAImageFromSpec creates a CUDA image from the input OCI runtime spec.
// The process environment is read (if present) to construc the CUDA Image.
func NewCUDAImageFromSpec(spec *specs.Spec) (CUDA, error) {
	if spec == nil || spec.Process == nil {
		return NewCUDAImageFromEnv(nil)
	}

	return NewCUDAImageFromEnv(spec.Process.Env)
}

// NewCUDAImageFromEnv creates a CUDA image from the input environment. The environment
// is a list of strings of the form ENVAR=VALUE.
func NewCUDAImageFromEnv(env []string) (CUDA, error) {
	c := make(CUDA)

	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid environment variable: %v", e)
		}
		c[parts[0]] = parts[1]
	}

	return c, nil
}

// IsLegacy returns whether the associated CUDA image is a "legacy" image. An
// image is considered legacy if it has a CUDA_VERSION environment variable defined
// and no NVIDIA_REQUIRE_CUDA environment variable defined.
func (i CUDA) IsLegacy() bool {
	legacyCudaVersion := i[envCUDAVersion]
	cudaRequire := i[envNVRequireCUDA]
	return len(legacyCudaVersion) > 0 && len(cudaRequire) == 0
}

// GetRequirements returns the requirements from all NVIDIA_REQUIRE_ environment
// variables.
func (i CUDA) GetRequirements() ([]string, error) {
	// TODO: We need not process this if disable require is set, but this will be done
	// in a single follow-up to ensure that the behavioural change is accurately captured.
	// if i.HasDisableRequire() {
	// 	return nil, nil
	// }

	// All variables with the "NVIDIA_REQUIRE_" prefix are passed to nvidia-container-cli
	var requirements []string
	for name, value := range i {
		if strings.HasPrefix(name, envNVRequirePrefix) && !strings.HasPrefix(name, envNVRequireJetpack) {
			requirements = append(requirements, value)
		}
	}
	if i.IsLegacy() {
		v, err := i.legacyVersion()
		if err != nil {
			return nil, fmt.Errorf("failed to get version: %v", err)
		}
		cudaRequire := fmt.Sprintf("cuda>=%s", v)
		requirements = append(requirements, cudaRequire)
	}
	return requirements, nil
}

// HasDisableRequire checks for the value of the NVIDIA_DISABLE_REQUIRE. If set
// to a valid (true) boolean value this can be used to disable the requirement checks
func (i CUDA) HasDisableRequire() bool {
	if disable, exists := i[envNVDisableRequire]; exists {
		// i.logger.Debugf("NVIDIA_DISABLE_REQUIRE=%v; skipping requirement checks", disable)
		d, _ := strconv.ParseBool(disable)
		return d
	}

	return false
}

func (i CUDA) legacyVersion() (string, error) {
	majorMinor, err := parseMajorMinorVersion(i[envCUDAVersion])
	if err != nil {
		return "", fmt.Errorf("invalid CUDA version: %v", err)
	}

	return majorMinor, nil
}

func parseMajorMinorVersion(version string) (string, error) {
	vVersion := "v" + strings.TrimPrefix(version, "v")

	if !semver.IsValid(vVersion) {
		return "", fmt.Errorf("invalid version string")
	}

	majorMinor := strings.TrimPrefix(semver.MajorMinor(vVersion), "v")
	parts := strings.Split(majorMinor, ".")

	var err error
	_, err = strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		return "", fmt.Errorf("invalid major version")
	}
	_, err = strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return "", fmt.Errorf("invalid minor version")
	}
	return majorMinor, nil
}
