/**
# Copyright (c) NVIDIA CORPORATION.  All rights reserved.
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

package dgpu

import (
	"fmt"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
)

// newNvsandboxutilsDriverDiscoverer constructs a discoverer from the specified nvsandboxutils library.
func (o *options) newNvsandboxutilsDriverDiscoverer() (discover.Discover, error) {
	if o.nvsandboxutilslib == nil {
		return nil, nil
	}
	return nil, fmt.Errorf("nvsandboxutils driver discovery is not implemented")
}
