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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFeatures(t *testing.T) {
	testCases := []struct {
		description string
		features    features
		expected    map[featureName]bool
		envs        []getenver
	}{
		{
			description: "empty features",
			expected: map[featureName]bool{
				FeatureAllowAdditionalGIDs: false,
				FeatureGDRCopy:             false,
				FeatureGDS:                 false,
				FeatureMOFED:               false,
				FeatureNVSWITCH:            false,
			},
		},
		{
			description: "envvar sets features if enabled",
			expected: map[featureName]bool{
				FeatureAllowAdditionalGIDs: true,
				FeatureGDRCopy:             false,
				FeatureGDS:                 false,
				FeatureMOFED:               false,
				FeatureNVSWITCH:            false,
			},
			envs: []getenver{
				mockEnver{
					"NVIDIA_ALLOW_ADDITIONAL_GIDS": "enabled",
				},
			},
		},
		{
			description: "envvar sets features if true",
			expected: map[featureName]bool{
				FeatureAllowAdditionalGIDs: true,
				FeatureGDRCopy:             false,
				FeatureGDS:                 false,
				FeatureMOFED:               false,
				FeatureNVSWITCH:            false,
			},
			envs: []getenver{
				mockEnver{
					"NVIDIA_ALLOW_ADDITIONAL_GIDS": "true",
				},
			},
		},
		{
			description: "feature sets feature",
			features: features{
				AllowAdditionalGIDs: ptr(featureEnabled),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			for n, v := range tc.expected {
				t.Run(tc.description+"-"+string(n), func(t *testing.T) {
					require.EqualValues(t, v, tc.features.IsEnabled(n, tc.envs...))
				})
			}

		})
	}
}

func TestFeature(t *testing.T) {
	testCases := []struct {
		description string
		feature     *feature
		envvar      string
		envs        []getenver
		expected    bool
	}{
		{
			description: "nil feature is false",
			feature:     nil,
			expected:    false,
		},
		{
			description: "feature enables",
			feature:     ptr(featureEnabled),
			expected:    true,
		},
		{
			description: "feature disabled",
			feature:     ptr(featureDisabled),
			expected:    false,
		},
		{
			description: "envvar overrides feature disabled",
			feature:     ptr(featureDisabled),
			envvar:      "FEATURE",
			envs: []getenver{
				mockEnver{"FEATURE": "enabled"},
			},
			expected: true,
		},
		{
			description: "envvar overrides feature enabled",
			feature:     ptr(featureEnabled),
			envvar:      "FEATURE",
			envs: []getenver{
				mockEnver{"FEATURE": "disabled"},
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			require.EqualValues(t, tc.expected, tc.feature.isEnabled(tc.envvar, tc.envs...))
		})
	}
}

type mockEnver map[string]string

func (m mockEnver) Getenv(k string) string {
	return m[k]
}

func ptr[T any](x T) *T {
	return &x
}
