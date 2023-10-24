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
	"fmt"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestResolveMode(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	testCases := []struct {
		mode      string
		isTegra   bool
		hasDXCore bool
		hasNVML   bool
		expected  string
	}{
		{
			mode:      "auto",
			hasDXCore: true,
			expected:  "wsl",
		},
		{
			mode:      "auto",
			hasDXCore: false,
			isTegra:   true,
			hasNVML:   false,
			expected:  "csv",
		},
		{
			mode:      "auto",
			hasDXCore: false,
			isTegra:   false,
			hasNVML:   false,
			expected:  "nvml",
		},
		{
			mode:      "auto",
			hasDXCore: false,
			isTegra:   true,
			hasNVML:   true,
			expected:  "nvml",
		},
		{
			mode:      "auto",
			hasDXCore: false,
			isTegra:   false,
			expected:  "nvml",
		},
		{
			mode:      "nvml",
			hasDXCore: true,
			isTegra:   true,
			expected:  "nvml",
		},
		{
			mode:      "wsl",
			hasDXCore: false,
			expected:  "wsl",
		},
		{
			mode:      "not-auto",
			hasDXCore: true,
			expected:  "not-auto",
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			l := nvcdilib{
				logger:  logger,
				mode:    tc.mode,
				infolib: infoMock{hasDXCore: tc.hasDXCore, isTegra: tc.isTegra, hasNVML: tc.hasNVML},
			}

			require.Equal(t, tc.expected, l.resolveMode())
		})
	}
}

type infoMock struct {
	hasDXCore bool
	isTegra   bool
	hasNVML   bool
}

func (i infoMock) HasDXCore() (bool, string) {
	return i.hasDXCore, ""
}

func (i infoMock) HasNvml() (bool, string) {
	return i.hasNVML, ""
}

func (i infoMock) IsTegraSystem() (bool, string) {
	return i.isTegra, ""
}
