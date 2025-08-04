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

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFeatureIsEnabled(t *testing.T) {
	testCases := []struct {
		name     string
		feature  *feature
		expected bool
	}{
		{
			name:     "nil feature returns false",
			feature:  nil,
			expected: false,
		},
		{
			name:     "false feature returns false",
			feature:  (*feature)(ptr(false)),
			expected: false,
		},
		{
			name:     "true feature returns true",
			feature:  (*feature)(ptr(true)),
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.feature.IsEnabled())
		})
	}
}

func TestEnableChmodHookFeature(t *testing.T) {
	testCases := []struct {
		name     string
		config   features
		expected bool
	}{
		{
			name:     "default (nil) is disabled",
			config:   features{},
			expected: false,
		},
		{
			name: "explicitly disabled",
			config: features{
				EnableChmodHook: (*feature)(ptr(false)),
			},
			expected: false,
		},
		{
			name: "explicitly enabled",
			config: features{
				EnableChmodHook: (*feature)(ptr(true)),
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.config.EnableChmodHook.IsEnabled())
		})
	}
}
