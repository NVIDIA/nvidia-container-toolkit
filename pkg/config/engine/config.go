/**
# Copyright (c) NVIDIA CORPORATION.  All rights reserved.
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

package engine

import (
	"fmt"
	"os"
	"path/filepath"
)

// Config represents a runtime config
type Config string

// Write writes the specified contents to a config file.
func (c Config) Write(output []byte) (int, error) {
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
