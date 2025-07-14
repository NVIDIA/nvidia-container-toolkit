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

import (
	"fmt"
	"os"
)

type tomlFile string

var _ Loader = (*tomlFile)(nil)

// Load loads the contents of the specified TOML file as a map.
func (f tomlFile) Load() (*Tree, error) {
	info, err := os.Stat(string(f))
	if os.IsExist(err) && info.IsDir() {
		return nil, fmt.Errorf("config file %s is a directory", string(f))
	}

	if os.IsNotExist(err) {
		return Empty.Load()
	}

	return LoadFile(string(f))
}
