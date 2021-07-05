/*
# Copyright (c) 2021, NVIDIA CORPORATION.  All rights reserved.
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
*/

package discover

import (
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestBinaries(t *testing.T) {
	binaryLookup := map[string]string{
		"nvidia-smi":              "/usr/bin/nvidia-smi",
		"nvidia-persistenced":     "/usr/bin/nvidia-persistenced",
		"nvidia-debugdump":        "test-duplicates",
		"nvidia-cuda-mps-control": "test-duplicates",
	}

	logger, _ := testlog.NewNullLogger()

	d := NewBinaryMountsWithLogger(logger, "")

	// Override lookup for testing
	mockLocator := NewLocatorMockFromMap(binaryLookup)
	d.(*mounts).lookup = mockLocator

	mounts, err := d.Mounts()
	require.NoError(t, err)

	expected := []Mount{
		{
			Path: "/usr/bin/nvidia-smi",
		},
		{
			Path: "/usr/bin/nvidia-persistenced",
		},
		{
			Path: "test-duplicates",
		},
	}

	require.Equal(t, len(expected), len(mounts))

	devices, err := d.Devices()
	require.NoError(t, err)
	require.Empty(t, devices)

	hooks, err := d.Hooks()
	require.NoError(t, err)
	require.Empty(t, hooks)
}

func TestNewBinariesConstructor(t *testing.T) {
	b := NewBinaryMounts("").(*mounts)

	require.NotNil(t, b.logger)
	require.NotNil(t, b.lookup)
}
