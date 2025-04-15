/**
# SPDX-FileCopyrightText: Copyright (c) 2025 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
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

package disabledevicenodemodification

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestGetModifiedParamsFileContentsFromReader(t *testing.T) {
	logger, _ := testlog.NewNullLogger()
	testCases := map[string]struct {
		contents         []byte
		expectedError    error
		expectedContents []byte
	}{
		"no contents": {
			contents:         nil,
			expectedError:    nil,
			expectedContents: nil,
		},
		"other contents are ignored": {
			contents: []byte(`# Some other content
			that we don't care about
			`),
			expectedError:    nil,
			expectedContents: nil,
		},
		"already zero requires no modification": {
			contents:         []byte("ModifyDeviceFiles: 0"),
			expectedError:    nil,
			expectedContents: nil,
		},
		"leading spaces require no modification": {
			contents: []byte("  ModifyDeviceFiles: 1"),
		},
		"Trailing spaces require no modification": {
			contents: []byte("ModifyDeviceFiles: 1  "),
		},
		"Not 1 require no modification": {
			contents: []byte("ModifyDeviceFiles: 11"),
		},
		"single line requires modification": {
			contents:         []byte("ModifyDeviceFiles: 1"),
			expectedError:    nil,
			expectedContents: []byte("ModifyDeviceFiles: 0\n"),
		},
		"single line with trailing newline requires modification": {
			contents:         []byte("ModifyDeviceFiles: 1\n"),
			expectedError:    nil,
			expectedContents: []byte("ModifyDeviceFiles: 0\n"),
		},
		"other content is maintained": {
			contents: []byte(`ModifyDeviceFiles: 1
			other content
			that
			is maintained`),
			expectedError: nil,
			expectedContents: []byte(`ModifyDeviceFiles: 0
			other content
			that
			is maintained
`),
		},
	}

	for description, tc := range testCases {
		t.Run(description, func(t *testing.T) {
			c := command{
				logger: logger,
			}
			contents, err := c.getModifiedParamsFileContentsFromReader(bytes.NewReader(tc.contents))
			require.EqualValues(t, tc.expectedError, err)
			require.EqualValues(t, string(tc.expectedContents), string(contents))
		})
	}

}
