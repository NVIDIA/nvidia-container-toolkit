/**
# Copyright (c) 2022, NVIDIA CORPORATION.  All rights reserved.
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

package modifier

import (
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/image"
)

func TestNewCSVModifier(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	testCases := []struct {
		description   string
		cfg           *config.Config
		envmap        map[string]string
		expectedError error
		expectedNil   bool
	}{
		{
			description: "visible devices not set returns nil",
			envmap:      map[string]string{},
			expectedNil: true,
		},
		{
			description: "visible devices empty returns nil",
			envmap:      map[string]string{"NVIDIA_VISIBLE_DEVICES": ""},
			expectedNil: true,
		},
		{
			description: "visible devices 'void' returns nil",
			envmap:      map[string]string{"NVIDIA_VISIBLE_DEVICES": "void"},
			expectedNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			image, _ := image.New(
				image.WithEnvMap(tc.envmap),
			)
			m, err := NewCSVModifier(logger, tc.cfg, image)
			if tc.expectedError != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tc.expectedNil || tc.expectedError != nil {
				require.Nil(t, m)
			} else {
				require.NotNil(t, m)
			}
		})
	}
}
