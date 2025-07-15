/*
# Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
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

package oci

import (
	"reflect"
	"testing"
)

func TestEscape(t *testing.T) {
	testCases := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "Empty Slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "Slice with no Metacharacters",
			input:    []string{"ls", "-l", "/home/user"},
			expected: []string{"ls", "-l", "/home/user"},
		},
		{
			name:     "Slice with some Metacharacters",
			input:    []string{"echo", "Hello World", "and", "goodbye | cat"},
			expected: []string{"echo", `"Hello\ World"`, "and", `"goodbye\ \|\ cat"`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := Escape(tc.input)
			if !reflect.DeepEqual(actual, tc.expected) {
				t.Errorf("Escape(%q) = %q; want %q", tc.input, actual, tc.expected)
			}
		})
	}
}
