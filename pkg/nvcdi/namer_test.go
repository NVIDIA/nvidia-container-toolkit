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

package nvcdi

import (
	"testing"

	"github.com/NVIDIA/go-nvlib/pkg/nvml"
	"github.com/stretchr/testify/require"
)

func TestConvert(t *testing.T) {
	testCases := []struct {
		description   string
		nvml          nvmlUUIDer
		expectedError error
		expecteUUID   string
	}{
		{
			description:   "empty UUIDer returns error",
			expectedError: errUUIDUnsupported,
			expecteUUID:   "",
		},
		{
			description: "nvmlUUIDer returns UUID",
			nvml: &nvmlUUIDerMock{
				GetUUIDFunc: func() (string, nvml.Return) {
					return "SOME_UUID", nvml.SUCCESS
				},
			},
			expectedError: nil,
			expecteUUID:   "SOME_UUID",
		},
		{
			description: "nvmlUUIDer returns error",
			nvml: &nvmlUUIDerMock{
				GetUUIDFunc: func() (string, nvml.Return) {
					return "SOME_UUID", nvml.ERROR_UNKNOWN
				},
			},
			expectedError: nvml.ERROR_UNKNOWN,
			expecteUUID:   "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			uuid, err := convert{tc.nvml}.GetUUID()
			require.ErrorIs(t, err, tc.expectedError)
			require.Equal(t, tc.expecteUUID, uuid)
		})
	}
}
