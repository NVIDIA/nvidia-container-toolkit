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
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml"
)

type Tree *toml.Tree

// Toml represents a generic toml config with a logger.
type Toml struct {
	*toml.Tree
}

func Load(content string) (*Toml, error) {
	return new(func() (*toml.Tree, error) {
		return toml.Load(content)
	})
}

func LoadBytes(b []byte) (*Toml, error) {
	return new(func() (*toml.Tree, error) {
		return toml.LoadBytes(b)
	})
}

func LoadFile(path string) (*Toml, error) {
	return new(func() (*toml.Tree, error) {
		return toml.LoadFile(path)
	})
}

func LoadMap(m map[string]interface{}) (*Toml, error) {
	return new(func() (*toml.Tree, error) {
		return toml.TreeFromMap(m)
	})
}

func new(construct func() (*toml.Tree, error)) (*Toml, error) {
	tomlTree, err := construct()
	if err != nil {
		return nil, err
	}
	t := &Toml{
		Tree: tomlTree,
	}

	return t, nil
}

// Save writes the config to the specified path
func (c Toml) Save(path string) (int64, error) {
	config := c.Tree
	output, err := config.Marshal()
	if err != nil {
		return 0, fmt.Errorf("unable to convert to TOML: %v", err)
	}

	n, err := Raw(path).Write(output)
	return int64(n), err
}

// Raw represents a raw config file
type Raw string

// Write writes the specified contents to a config file.
func (c Raw) Write(output []byte) (int, error) {
	path := string(c)
	if path == "" {
		n, err := os.Stdout.Write(output)
		if err == nil {
			os.Stdout.WriteString("\n")
		}
		return n, err
	}

	if len(output) == 0 {
		err := os.Remove(path)
		if err != nil {
			return 0, fmt.Errorf("unable to remove empty file: %v", err)
		}
		return 0, nil
	}

	if dir := filepath.Dir(path); dir != "" {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return 0, fmt.Errorf("unable to create directory %v: %v", dir, err)
		}
	}

	f, err := os.Create(path)
	if err != nil {
		return 0, fmt.Errorf("unable to open %v for writing: %v", path, err)
	}
	defer f.Close()

	return f.Write(output)
}
