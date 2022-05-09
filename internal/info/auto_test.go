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

package info

import (
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestResolveAutoMode(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	testCases := []struct {
		description  string
		mode         string
		expectedMode string
	}{
		{
			description:  "non-auto resolves to input",
			mode:         "not-auto",
			expectedMode: "not-auto",
		},
		// TODO: The following test is brittle in that it will break on Tegra-based systems.
		// {
		// 	description:  "auto resolves to legacy",
		// 	mode:         "auto",
		// 	expectedMode: "legacy",
		// },
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			mode := ResolveAutoMode(logger, tc.mode)
			require.EqualValues(t, tc.expectedMode, mode)
		})
	}
}
