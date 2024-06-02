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

package containerd

import (
	"fmt"

	"github.com/pelletier/go-toml"
)

// tomlTree is an alias for toml.Tree that allows for extensions.
type tomlTree toml.Tree

func subtreeAtPath(c toml.Tree, path ...string) *tomlTree {
	tree := c.GetPath(path).(*toml.Tree)
	return (*tomlTree)(tree)
}

func (t *tomlTree) insert(other map[string]interface{}) error {

	for key, value := range other {
		if insertsubtree, ok := value.(map[string]interface{}); ok {
			subtree := (*toml.Tree)(t).Get(key).(*toml.Tree)
			return (*tomlTree)(subtree).insert(insertsubtree)
		}
		(*toml.Tree)(t).Set(key, value)
	}
	return nil
}

func (t *tomlTree) applyOverrides(overrides ...map[string]interface{}) error {
	for _, override := range overrides {
		subconfig, err := toml.TreeFromMap(override)
		if err != nil {
			return fmt.Errorf("invalid toml config: %w", err)
		}
		if err := t.insert(subconfig.ToMap()); err != nil {
			return err
		}
	}
	return nil
}
