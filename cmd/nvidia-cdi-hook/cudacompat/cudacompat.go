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
	"strings"

	"github.com/moby/sys/symlink"
	"github.com/urfave/cli/v2"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

const (
	cudaCompatPath = "/usr/local/cuda/compat"
)

type command struct {
	logger logger.Interface
}

type options struct {
	hostDriverVersion string
	containerSpec     string
}

// NewCommand constructs an cuda-compat command with the specified logger
func NewCommand(logger logger.Interface) *cli.Command {
	c := command{
		logger: logger,
	}
	return c.build()
}

// build the cuda-compat command
func (m command) build() *cli.Command {
	cfg := options{}

	// Create the 'cuda-compat' command
	c := cli.Command{
		Name:  "cuda-compat",
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

func (m command) validateFlags(c *cli.Context, cfg *options) error {
	return nil
}

func (m command) run(c *cli.Context, cfg *options) error {
	if cfg.hostDriverVersion == "" {
		return nil
	}
	s, err := oci.LoadContainerState(cfg.containerSpec)
	if err != nil {
		return fmt.Errorf("failed to load container state: %w", err)
	}

	containerRoot, err := s.GetContainerRoot()
	if err != nil {
		return fmt.Errorf("failed to determined container root: %w", err)
	}

	return m.updateCUDACompatLibs(root(containerRoot), cfg.hostDriverVersion)
}

func (m command) updateCUDACompatLibs(containerRoot root, hostDriverVersion string) error {
	if hostDriverVersion == "" {
		return nil
	}
	if !containerRoot.hasPath(cudaCompatPath) {
		return nil
	}
	if !containerRoot.hasPath("/etc/ld.so.cache") {
		// If there is no ldcache in the container, the hook is a no-op.
		return nil
	}

	libs, err := containerRoot.globFiles(filepath.Join(cudaCompatPath, "libcuda.so.*.*"))
	if err != nil {
		m.logger.Warningf("Failed to find CUDA compat library: %w", err)
		return nil
	}

	if len(libs) == 0 {
		return nil
	}

	if len(libs) != 1 {
		m.logger.Warningf("Unexpected number of CUDA compat libraries: %v", libs)
		return nil
	}

	compatDriverVersion := strings.TrimPrefix(filepath.Base(libs[0]), "libcuda.so.")
	compatMajor := strings.SplitN(compatDriverVersion, ".", 2)[0]

	driverMajor := strings.SplitN(hostDriverVersion, ".", 2)[0]

	if driverMajor < compatMajor {
		return m.createLdsoconfdFile(containerRoot, []string{filepath.Dir(libs[0])})
	}
	return nil
}

// A root is used to add basic path functionality to a string.
type root string

// hasPath checks whether the specified path exists in the root.
func (r root) hasPath(path string) bool {
	resolved, err := r.resolve(path)
	if err != nil {
		return false
	}
	if _, err := os.Stat(resolved); err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

// globFiles matches the specified pattern in the root.
// The files that match must be regular files.
func (r root) globFiles(pattern string) ([]string, error) {
	patternPath, err := r.resolve(pattern)
	if err != nil {
		return nil, err
	}
	matches, err := filepath.Glob(patternPath)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, match := range matches {
		info, err := os.Lstat(match)
		if err != nil {
			return nil, err
		}
		// Ignore symlinks.
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		// Ignore directories.
		if info.IsDir() {
			continue
		}
		files = append(files, match)
	}
	return files, nil
}

// resolve returns the absolute path including root path.
// Symlinks are resolved, but are guaranteed to resolve in the root.
func (r root) resolve(path string) (string, error) {
	absolute := filepath.Clean(filepath.Join(string(r), path))
	return symlink.FollowSymlinkInScope(absolute, string(r))
}

// createLdsoconfdFile creates (or updates) /etc/ld.so.conf.d/00-compat-<RANDOM_STRING>.conf in the container
// to include the required paths.
// Note that the 00-compat prefix is chosen to ensure that these libraries have
// a higher precedence than other libraries on the system.
func (m command) createLdsoconfdFile(in root, folders []string) error {
	if len(folders) == 0 {
		m.logger.Debugf("No folders to add to /etc/ld.so.conf")
		return nil
	}

	ldsoconfdDir, err := in.resolve("/etc/ld.so.conf.d")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(ldsoconfdDir, 0755); err != nil {
		return fmt.Errorf("failed to create ld.so.conf.d: %w", err)
	}

	configFile, err := os.CreateTemp(ldsoconfdDir, "00-compat-*.conf")
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer configFile.Close()

	m.logger.Debugf("Adding folders %v to %v", folders, configFile.Name())

	configured := make(map[string]bool)
	for _, folder := range folders {
		if configured[folder] {
			continue
		}
		_, err = configFile.WriteString(fmt.Sprintf("%s\n", folder))
		if err != nil {
			return fmt.Errorf("failed to update config file: %w", err)
		}
		configured[folder] = true
	}

	// The created file needs to be world readable for the cases where the container is run as a non-root user.
	if err := os.Chmod(configFile.Name(), 0644); err != nil {
		return fmt.Errorf("failed to chmod config file: %w", err)
	}

	return nil
}
