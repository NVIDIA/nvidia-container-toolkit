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
	"github.com/pelletier/go-toml"
)

type empty string

var _ Loader = (*empty)(nil)

// Load is a no-op for an empty source.
func (e empty) Load() (*Tree, error) {
	return newEmpty(), nil
}

func newEmpty() *Tree {
	tomlTree, _ := toml.TreeFromMap(nil)
	return (*Tree)(tomlTree)
}
