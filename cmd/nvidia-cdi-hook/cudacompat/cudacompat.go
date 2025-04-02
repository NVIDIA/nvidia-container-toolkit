/**
# Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
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

package cudacompat

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

const (
	cudaCompatPath = "/usr/local/cuda/compat"
	// cudaCompatLdsoconfdFilenamePattern specifies the pattern for the filename
	// in ld.so.conf.d that includes a reference to the CUDA compat path.
	// The 00-compat prefix is chosen to ensure that these libraries have a
	// higher precedence than other libraries on the system.
	cudaCompatLdsoconfdFilenamePattern = "00-compat-*.conf"
)

type command struct {
	logger logger.Interface
}

type options struct {
	hostDriverVersion string
	containerSpec     string
}

// NewCommand constructs a cuda-compat command with the specified logger
func NewCommand(logger logger.Interface) *cli.Command {
	c := command{
		logger: logger,
	}
	return c.build()
}

// build the enable-cuda-compat command
func (m command) build() *cli.Command {
	cfg := options{}

	// Create the 'enable-cuda-compat' command
	c := cli.Command{
		Name:  "enable-cuda-compat",
		Usage: "This hook ensures that the folder containing the CUDA compat libraries is added to the ldconfig search path if required.",
		Before: func(c *cli.Context) error {
			return m.validateFlags(c, &cfg)
		},
		Action: func(c *cli.Context) error {
			return m.run(c, &cfg)
		},
	}

	c.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "host-driver-version",
			Usage:       "Specify the host driver version. If the CUDA compat libraries detected in the container do not have a higher MAJOR version, the hook is a no-op.",
			Destination: &cfg.hostDriverVersion,
		},
		&cli.StringFlag{
			Name:        "container-spec",
			Hidden:      true,
			Category:    "testing-only",
			Usage:       "Specify the path to the OCI container spec. If empty or '-' the spec will be read from STDIN",
			Destination: &cfg.containerSpec,
		},
	}

	return &c
}

func (m command) validateFlags(_ *cli.Context, cfg *options) error {
	return nil
}

func (m command) run(_ *cli.Context, cfg *options) error {
	if cfg.hostDriverVersion == "" {
		return nil
	}

	s, err := oci.LoadContainerState(cfg.containerSpec)
	if err != nil {
		return fmt.Errorf("failed to load container state: %w", err)
	}

	containerRootDir, err := s.GetContainerRoot()
	if err != nil {
		return fmt.Errorf("failed to determined container root: %w", err)
	}

	containerForwardCompatDir, err := m.getContainerForwardCompatDir(containerRoot(containerRootDir), cfg.hostDriverVersion)
	if err != nil {
		return fmt.Errorf("failed to get container forward compat directory: %w", err)
	}
	if containerForwardCompatDir == "" {
		return nil
	}

	return m.createLdsoconfdFile(containerRoot(containerRootDir), cudaCompatLdsoconfdFilenamePattern, containerForwardCompatDir)
}

func (m command) getContainerForwardCompatDir(containerRoot containerRoot, hostDriverVersion string) (string, error) {
	if hostDriverVersion == "" {
		m.logger.Debugf("Host driver version not specified")
		return "", nil
	}
	if !containerRoot.hasPath(cudaCompatPath) {
		m.logger.Debugf("No CUDA forward compatibility libraries directory in container")
		return "", nil
	}
	if !containerRoot.hasPath("/etc/ld.so.cache") {
		m.logger.Debugf("The container does not have an LDCache")
		return "", nil
	}

	libs, err := containerRoot.globFiles(filepath.Join(cudaCompatPath, "libcuda.so.*.*"))
	if err != nil {
		m.logger.Warningf("Failed to find CUDA compat library: %w", err)
		return "", nil
	}

	if len(libs) == 0 {
		m.logger.Debugf("No CUDA forward compatibility libraries container")
		return "", nil
	}

	if len(libs) != 1 {
		m.logger.Warningf("Unexpected number of CUDA compat libraries in container: %v", libs)
		return "", nil
	}

	compatDriverVersion := strings.TrimPrefix(filepath.Base(libs[0]), "libcuda.so.")
	compatMajor, err := extractMajorVersion(compatDriverVersion)
	if err != nil {
		return "", fmt.Errorf("failed to extract major version from %q: %v", compatDriverVersion, err)
	}

	driverMajor, err := extractMajorVersion(hostDriverVersion)
	if err != nil {
		return "", fmt.Errorf("failed to extract major version from %q: %v", hostDriverVersion, err)
	}

	if driverMajor >= compatMajor {
		m.logger.Debugf("Compat major version is not greater than the host driver major version (%v >= %v)", hostDriverVersion, compatDriverVersion)
		return "", nil
	}

	resolvedCompatDir := strings.TrimPrefix(filepath.Dir(libs[0]), string(containerRoot))
	return resolvedCompatDir, nil
}

// createLdsoconfdFile creates a file at /etc/ld.so.conf.d/ in the specified root.
// The file is created at /etc/ld.so.conf.d/{{ .pattern }} using `CreateTemp` and
// contains the specified directories on each line.
func (m command) createLdsoconfdFile(in containerRoot, pattern string, dirs ...string) error {
	if len(dirs) == 0 {
		m.logger.Debugf("No directories to add to /etc/ld.so.conf")
		return nil
	}

	ldsoconfdDir, err := in.resolve("/etc/ld.so.conf.d")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(ldsoconfdDir, 0755); err != nil {
		return fmt.Errorf("failed to create ld.so.conf.d: %w", err)
	}

	configFile, err := os.CreateTemp(ldsoconfdDir, pattern)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer configFile.Close()

	m.logger.Debugf("Adding directories %v to %v", dirs, configFile.Name())

	added := make(map[string]bool)
	for _, dir := range dirs {
		if added[dir] {
			continue
		}
		_, err = fmt.Fprintf(configFile, "%s\n", dir)
		if err != nil {
			return fmt.Errorf("failed to update config file: %w", err)
		}
		added[dir] = true
	}

	// The created file needs to be world readable for the cases where the container is run as a non-root user.
	if err := configFile.Chmod(0644); err != nil {
		return fmt.Errorf("failed to chmod config file: %w", err)
	}

	return nil
}

// extractMajorVersion parses a version string and returns the major version as an int.
func extractMajorVersion(version string) (int, error) {
	majorString := strings.SplitN(version, ".", 2)[0]
	return strconv.Atoi(majorString)
}
