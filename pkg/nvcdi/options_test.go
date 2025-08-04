/**
# Copyright 2025 NVIDIA CORPORATION
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

package nvcdi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithEnableChmodHook(t *testing.T) {
	testCases := []struct {
		name     string
		opts     []Option
		expected bool
	}{
		{
			name:     "default is disabled",
			opts:     []Option{},
			expected: false,
		},
		{
			name:     "explicitly disabled",
			opts:     []Option{WithEnableChmodHook(false)},
			expected: false,
		},
		{
			name:     "explicitly enabled",
			opts:     []Option{WithEnableChmodHook(true)},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			lib := &nvcdilib{}
			for _, opt := range tc.opts {
				opt(lib)
			}
			assert.Equal(t, tc.expected, lib.enableChmodHook)
		})
	}
}
