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

package lookup

import (
	"fmt"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestCharDeviceLocator(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	testCases := []struct {
		root             string
		expectedPrefixes []string
	}{
		{
			root:             "",
			expectedPrefixes: []string{"", "/dev"},
		},
		{
			root:             "/",
			expectedPrefixes: []string{"/", "/dev"},
		},
		{
			root:             "/some/root",
			expectedPrefixes: []string{"/some/root", "/some/root/dev"},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			f := NewCharDeviceLocator(
				WithLogger(logger),
				WithRoot(tc.root),
			).(*file)

			require.EqualValues(t, tc.expectedPrefixes, f.prefixes)
		})
	}
}
