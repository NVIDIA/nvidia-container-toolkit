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

package modifier

import (
	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
	"github.com/sirupsen/logrus"
	"gitlab.com/nvidia/cloud-native/go-nvlib/pkg/nvlib/info"
)

// NewTegraPlatformFiles creates a modifier to inject the Tegra platform files into a container.
func NewTegraPlatformFiles(logger *logrus.Logger) (oci.SpecModifier, error) {
	isTegra, _ := info.New().IsTegraSystem()
	if !isTegra {
		return nil, nil
	}

	tegraSystemMounts := discover.NewMounts(
		logger,
		lookup.NewFileLocator(lookup.WithLogger(logger)),
		"",
		[]string{
			"/etc/nv_tegra_release",
			"/sys/devices/soc0/family",
		},
	)

	return NewModifierFromDiscoverer(logger, tegraSystemMounts)
}
