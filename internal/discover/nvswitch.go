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

import "github.com/NVIDIA/nvidia-container-toolkit/internal/logger"

// NewNvSwitchDiscoverer creates a discoverer for NVSWITCH devices.
func NewNvSwitchDiscoverer(logger logger.Interface, devRoot string) (Discover, error) {
	devices := NewCharDeviceDiscoverer(
		logger,
		devRoot,
		[]string{
			"/dev/nvidia-nvswitchctl",
			"/dev/nvidia-nvswitch*",
		},
	)

	return devices, nil
}
