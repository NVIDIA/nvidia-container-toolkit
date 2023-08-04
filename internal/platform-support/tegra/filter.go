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

package tegra

import "path/filepath"

type ignoreFilenamePatterns []string

func (d ignoreFilenamePatterns) Match(name string) bool {
	for _, pattern := range d {
		if match, _ := filepath.Match(pattern, filepath.Base(name)); match {
			return true
		}
	}
	return false
}

func (d ignoreFilenamePatterns) Apply(input ...string) []string {
	var filtered []string
	for _, name := range input {
		if d.Match(name) {
			continue
		}
		filtered = append(filtered, name)
	}
	return filtered
}
