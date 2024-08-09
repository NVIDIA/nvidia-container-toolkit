/**
# Copyright 2024 NVIDIA CORPORATION
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
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/pelletier/go-toml"
)

const (
	Empty = empty("")
)

// Loader represents a source for a toml config.
type Loader interface {
	Load() (*Toml, error)
}

type empty string

var _ Loader = (*empty)(nil)

// Load is a no-op for an empty source.
func (e empty) Load() (*Toml, error) {
	tomlTree, err := toml.TreeFromMap(nil)
	if err != nil {
		return nil, err
	}

	t := Toml{
		Tree: tomlTree,
	}
	return &t, nil
}

type tomlFile string

var _ Loader = (*tomlFile)(nil)

// FromFile creates a TOML source from the specified file.
// If an empty string is passed an empty toml config is used.
func FromFile(path string) Loader {
	if path == "" {
		return Empty
	}
	return tomlFile(path)
}

// Load loads the contents of the specified TOML file as a map.
func (f tomlFile) Load() (*Toml, error) {
	info, err := os.Stat(string(f))
	if os.IsExist(err) && info.IsDir() {
		return nil, fmt.Errorf("config file %s is a directory", string(f))
	}

	if os.IsNotExist(err) {
		return Empty.Load()
	}

	return LoadFile(string(f))
}

type tomlFromCommandOutput struct {
	command string
	args    []string
}

var _ Loader = (*tomlFromCommandOutput)(nil)

// Load runs the specified command and returns the TOML output as a map.
func (c *tomlFromCommandOutput) Load() (*Toml, error) {
	//nolint:gosec  // Subprocess launched with a potential tainted input or cmd arguments
	cmd := exec.Command(c.command, c.args...)

	var outb bytes.Buffer
	var errb bytes.Buffer

	cmd.Stdout = &outb
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to run command %v %v: %w", c.command, c.args, err)
	}

	return LoadBytes(outb.Bytes())
}

type tomlFromCRI struct {
	socket string
}

var _ Loader = (*tomlFromCRI)(nil)

// Load queries the specified CRI socket to get the runtime information.
// See for example: https://github.com/elezar/cri-tools/blob/cd486308b37ca13c5b5293b234876fd03d6f069a/cmd/crictl/info.go#L70
func (c *tomlFromCRI) Load() (*Toml, error) {
	return nil, fmt.Errorf("not implemented for socket %v", c.socket)
}
