/**
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
**/

package lookup

import (
	"github.com/NVIDIA/nvidia-container-toolkit/internal/devices"
)

// NewCharDeviceLocator creates a Locator that can be used to find char devices at the specified root. A logger is
// also specified.
func NewCharDeviceLocator(opts ...Option) Locator {
	filter := devices.AssertCharDevice

	opts = append(opts,
		// Device nodes can be specified by their full path e.g. /dev/nvidia0 or
		// by the name of the device node e.g nvidia0.
		// We thus set the search path to include "/" and "/dev" to cover both
		// cases.
		WithSearchPaths("/", "/dev"),
		WithFilter(filter),
	)
	return NewFactory(opts...).NewFileLocator()
}
