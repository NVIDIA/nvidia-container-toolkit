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
	"fmt"
	"testing"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	"github.com/stretchr/testify/require"
)

func TestMountsReturnsErrorForNoLookup(t *testing.T) {
	d := mounts{}
	mounts, err := d.Mounts()

	require.Error(t, err)
	require.Len(t, mounts, 0)

	devices, err := d.Devices()
	require.NoError(t, err)
	require.Empty(t, devices)

	hooks, err := d.Hooks()
	require.NoError(t, err)
	require.Empty(t, hooks)
}

func NewLocatorMockFromMap(lookupMap map[string]string) *lookup.LocatorMock {
	return &lookup.LocatorMock{
		LocateFunc: func(key string) ([]string, error) {
			value, exists := lookupMap[key]
			if !exists {
				return nil, fmt.Errorf("key %v not found", key)
			}
			return []string{value}, nil
		},
	}
}
