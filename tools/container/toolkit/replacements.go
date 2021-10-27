/**
# Copyright (c) 2021, NVIDIA CORPORATION.  All rights reserved.
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
*/

package main

import "strings"

const (
	destDirPattern = "@destDir@"
)

type replacements map[string]string

func newReplacements(rules ...string) replacements {
	r := make(replacements)
	for i := 0; i < len(rules)-1; i += 2 {
		old := rules[i]
		new := rules[i+1]

		r[old] = new
	}

	return r
}

func (r replacements) apply(input string) string {
	output := input
	for old, new := range r {
		output = strings.ReplaceAll(output, old, new)
	}
	return output
}
