/**
# SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
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

package ldcache

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/test"
)

func TestNewDoesNotLeaveCacheMapped(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	moduleRoot, err := test.GetModuleRoot()
	require.NoError(t, err)
	root := filepath.Join(moduleRoot, "testdata", "lookup", "rootfs-1")

	cache, err := New(logger, root)
	require.NoError(t, err)
	_, libs64 := cache.List()
	require.NotEmpty(t, libs64)

	maps, err := os.ReadFile("/proc/self/maps")
	require.NoError(t, err)
	require.False(t, strings.Contains(string(maps), filepath.Join(root, ldcachePath)), "ld.so.cache is still mapped")
}
