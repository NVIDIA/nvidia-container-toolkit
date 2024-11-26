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

package config

import (
	"fmt"
	"os"
	"strings"
)

// ContainerCLIConfig stores the options for the nvidia-container-cli
type ContainerCLIConfig struct {
	Root        string   `toml:"root"`
	Path        string   `toml:"path"`
	Environment []string `toml:"environment"`
	Debug       string   `toml:"debug"`
	Ldcache     string   `toml:"ldcache"`
	LoadKmods   bool     `toml:"load-kmods"`
	// NoPivot disables the pivot root operation in the NVIDIA Container CLI.
	// This is not exposed in the config if not set.
	NoPivot   bool   `toml:"no-pivot,omitempty"`
	NoCgroups bool   `toml:"no-cgroups"`
	User      string `toml:"user"`
	// Ldconfig represents the path to the ldconfig binary to be used to update
	// the ldcache in a container as it is being created.
	// If this path starts with a '@' the path is relative to the host and if
	// not it is treated as a container path.
	//
	// Note that the use of container paths are disabled by default and if this
	// is required, the features.allow-ldconfig-from-container feature gate must
	// be enabled explicitly.
	Ldconfig ldconfigPath `toml:"ldconfig"`
}

// NormalizeLDConfigPath returns the resolved path of the configured LDConfig binary.
// This is only done for host LDConfigs and is required to handle systems where
// /sbin/ldconfig is a wrapper around /sbin/ldconfig.real.
func (c *ContainerCLIConfig) NormalizeLDConfigPath() string {
	return string(c.Ldconfig.normalize())
}

// An ldconfigPath is used to represent the path to ldconfig.
type ldconfigPath string

func (p ldconfigPath) assertValid(allowContainerRelativePath bool) error {
	if p.isHostRelative() {
		return nil
	}
	if allowContainerRelativePath {
		return nil
	}
	return fmt.Errorf("nvidia-container-cli.ldconfig value %q is not host-relative (does not start with a '@')", p)
}

func (p ldconfigPath) isHostRelative() bool {
	return strings.HasPrefix(string(p), "@")
}

// normalize returns the resolved path of the configured LDConfig binary.
// This is only done for host LDConfigs and is required to handle systems where
// /sbin/ldconfig is a wrapper around /sbin/ldconfig.real.
func (p ldconfigPath) normalize() ldconfigPath {
	if !p.isHostRelative() {
		return p
	}

	path := string(p)
	trimmedPath := strings.TrimSuffix(strings.TrimPrefix(path, "@"), ".real")
	// If the .real path exists, we return that.
	if _, err := os.Stat(trimmedPath + ".real"); err == nil {
		return ldconfigPath("@" + trimmedPath + ".real")
	}
	// If the .real path does not exists (or cannot be read) we return the non-.real path.
	return ldconfigPath("@" + trimmedPath)
}

// NormalizeLDConfigPath returns the resolved path of the configured LDConfig binary.
// This is only done for host LDConfigs and is required to handle systems where
// /sbin/ldconfig is a wrapper around /sbin/ldconfig.real.
func NormalizeLDConfigPath(path string) string {
	return string(ldconfigPath(path).normalize())
}
