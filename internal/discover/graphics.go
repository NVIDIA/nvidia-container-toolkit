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

package discover

import (
	"fmt"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	"github.com/sirupsen/logrus"
)

// NewGraphicsDiscoverer returns the discoverer for graphics tools such as Vulkan.
func NewGraphicsDiscoverer(logger *logrus.Logger, root string) (Discover, error) {
	locator, err := lookup.NewLibraryLocator(logger, root)
	if err != nil {
		return nil, fmt.Errorf("failed to construct library locator: %v", err)
	}
	libraries := NewMounts(
		logger,
		locator,
		root,
		[]string{
			"libnvidia-egl-gbm.so",
		},
	)

	jsonMounts := NewMounts(
		logger,
		lookup.NewFileLocator(logger, root),
		root,
		[]string{
			// TODO: We should handle this more cleanly
			"/etc/glvnd/egl_vendor.d/10_nvidia.json",
			"/etc/vulkan/icd.d/nvidia_icd.json",
			"/etc/vulkan/implicit_layer.d/nvidia_layers.json",
			"/usr/share/glvnd/egl_vendor.d/10_nvidia.json",
			"/usr/share/vulkan/icd.d/nvidia_icd.json",
			"/usr/share/vulkan/implicit_layer.d/nvidia_layers.json",
			"/usr/share/egl/egl_external_platform.d/15_nvidia_gbm.json",
		},
	)

	discover := Merge(
		libraries,
		jsonMounts,
	)

	return discover, nil
}
