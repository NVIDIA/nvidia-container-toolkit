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

	"github.com/stretchr/testify/require"
)

func TestGetSearchPrefixes(t *testing.T) {
	testCases := []struct {
		root             string
		prefixes         []string
		expectedPrefixes []string
	}{
		{
			root:             "",
			expectedPrefixes: []string{""},
		},
		{
			root:             "/",
			expectedPrefixes: []string{"/"},
		},
		{
			root:             "/some/root",
			expectedPrefixes: []string{"/some/root"},
		},
		{
			root:             "",
			prefixes:         []string{"foo", "bar"},
			expectedPrefixes: []string{"foo", "bar"},
		},
		{
			root:             "/",
			prefixes:         []string{"foo", "bar"},
			expectedPrefixes: []string{"/foo", "/bar"},
		},
		{
			root:             "/",
			prefixes:         []string{"/foo", "/bar"},
			expectedPrefixes: []string{"/foo", "/bar"},
		},
		{
			root:             "/some/root",
			prefixes:         []string{"foo", "bar"},
			expectedPrefixes: []string{"/some/root/foo", "/some/root/bar"},
		},
		{
			root:             "",
			prefixes:         []string{"foo", "bar", "bar", "foo"},
			expectedPrefixes: []string{"foo", "bar"},
		},
		{
			root:             "/some/root",
			prefixes:         []string{"foo", "bar", "foo", "bar"},
			expectedPrefixes: []string{"/some/root/foo", "/some/root/bar"},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			prefixes := getSearchPrefixes(tc.root, tc.prefixes...)
			require.EqualValues(t, tc.expectedPrefixes, prefixes)
		})
	}
}
