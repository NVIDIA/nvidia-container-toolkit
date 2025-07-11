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

	"github.com/pelletier/go-toml"

	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config"
)

type Tree toml.Tree

// Copy produces a copy of the contents of the Tree.
func (t *Tree) Copy() *Tree {
	copy, _ := Load(t.String())
	return copy
}

func (t *Tree) GetSubtreeByPath(keys []string) *Tree {
	subtree := t.GetPath(keys)
	if subtree == nil {
		return nil
	}

	switch subtree := subtree.(type) {
	case *toml.Tree:
		return (*Tree)(subtree)
	case *Tree:
		return subtree
	default:
		panic(fmt.Errorf("invalid subtree type %T", subtree))
	}
}

func (t *Tree) DeletePath(keys []string) error {
	return (*toml.Tree)(t).DeletePath(keys)
}

func (t *Tree) HasPath(keys []string) bool {
	return (*toml.Tree)(t).HasPath(keys)
}

func (t *Tree) Get(key string) interface{} {
	return toTreeFromRaw((*toml.Tree)(t).Get(key))
}

func (t *Tree) GetPath(keys []string) interface{} {
	return toTreeFromRaw((*toml.Tree)(t).GetPath(keys))
}

func (t *Tree) SetPath(keys []string, value interface{}) {
	(*toml.Tree)(t).SetPath(keys, toRawFromTree(value))
}

func (t *Tree) Set(key string, value interface{}) {
	(*toml.Tree)(t).Set(key, toRawFromTree(value))
}

func (t *Tree) Delete(key string) error {
	return (*toml.Tree)(t).Delete(key)
}

func (t *Tree) Keys() []string {
	return (*toml.Tree)(t).Keys()
}

func (t *Tree) String() string {
	return (*toml.Tree)(t).String()
}

func (t *Tree) ToMap() map[string]interface{} {
	return (*toml.Tree)(t).ToMap()
}

func (t *Tree) Raw() *toml.Tree {
	return (*toml.Tree)(t)
}

func toRawFromTree(value interface{}) interface{} {
	if tree, ok := value.(*Tree); ok {
		return (*toml.Tree)(tree)
	}
	return value
}

func toTreeFromRaw(value interface{}) interface{} {
	if tree, ok := value.(*toml.Tree); ok {
		return (*Tree)(tree)
	}
	return value
}

func TreeFromMap(m map[string]interface{}) (*Tree, error) {
	return new(func() (*toml.Tree, error) {
		return toml.TreeFromMap(m)
	})
}

func Load(content string) (*Tree, error) {
	return new(func() (*toml.Tree, error) {
		return toml.Load(content)
	})
}

func LoadBytes(b []byte) (*Tree, error) {
	return new(func() (*toml.Tree, error) {
		return toml.LoadBytes(b)
	})
}

func LoadFile(path string) (*Tree, error) {
	return new(func() (*toml.Tree, error) {
		return toml.LoadFile(path)
	})
}

func LoadMap(m map[string]interface{}) (*Tree, error) {
	return TreeFromMap(m)
}

func Marshal(v interface{}) ([]byte, error) {
	return toml.Marshal(v)
}

func new(construct func() (*toml.Tree, error)) (*Tree, error) {
	tomlTree, err := construct()
	if err != nil {
		return nil, err
	}
	return (*Tree)(tomlTree), nil
}

// Save writes the config to the specified path
func (t *Tree) Save(path string) (int64, error) {
	cfg := (*toml.Tree)(t)
	output, err := cfg.Marshal()
	if err != nil {
		return 0, fmt.Errorf("unable to convert to TOML: %v", err)
	}

	n, err := config.Raw(path).Write(output)
	return int64(n), err
}
