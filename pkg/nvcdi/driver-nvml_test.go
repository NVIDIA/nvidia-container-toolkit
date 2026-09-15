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

package nvcdi

import (
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/root"
)

func TestNewIPCDiscoverer(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	t.Run("default enables IPC discoverer", func(t *testing.T) {
		l := &nvcdilib{
			logger:       logger,
			driver:       root.New(root.WithDriverRoot("/")),
			featureFlags: make(map[FeatureFlag]bool),
		}
		d, err := l.newIPCDiscoverer()
		require.NoError(t, err)
		require.NotNil(t, d)
	})

	t.Run("FeatureDisableIPCDiscoverer disables IPC discoverer", func(t *testing.T) {
		l := &nvcdilib{
			logger: logger,
			driver: root.New(root.WithDriverRoot("/")),
			featureFlags: map[FeatureFlag]bool{
				FeatureDisableIPCDiscoverer: true,
			},
		}
		d, err := l.newIPCDiscoverer()
		require.NoError(t, err)
		require.Nil(t, d)
	})
}
