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

package toml

const (
	Empty = empty("")
)

// Loader represents a source for a toml config.
type Loader interface {
	Load() (*Tree, error)
}

// FromFile creates a TOML source from the specified file.
// If an empty string is passed an empty toml config is used.
func FromFile(path string) Loader {
	if path == "" {
		return Empty
	}
	return tomlFile(path)
}

// FromCommandLine creates a TOML source from the output of a shell command and its corresponding args.
// If the command is empty, an empty config is returned.
func FromCommandLine(cmd string, args ...string) Loader {
	if len(cmd) == 0 {
		return Empty
	}
	return &tomlCliSource{
		command: cmd,
		args:    args,
	}
}
