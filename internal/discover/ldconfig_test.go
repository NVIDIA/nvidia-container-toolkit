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

package discover

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsLibName(t *testing.T) {
	testCases := []struct {
		name  string
		isLib bool
	}{
		{
			name:  "",
			isLib: false,
		},
		{
			name:  "lib/not/.so",
			isLib: false,
		},
		{
			name:  "lib.so",
			isLib: false,
		},
		{
			name:  "notlibcuda.so",
			isLib: false,
		},
		{
			name:  "libcuda.so",
			isLib: true,
		},
		{
			name:  "libcuda.so.1",
			isLib: true,
		},
		{
			name:  "libcuda.soNOT",
			isLib: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.isLib, isLibName(tc.name))
		})
	}
}
