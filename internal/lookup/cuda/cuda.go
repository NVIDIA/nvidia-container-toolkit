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

package cuda

import (
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
)

type cudaLocator struct {
	lookup.Locator
}

// New creates a new CUDA library locator.
func New(libraries lookup.Locator) lookup.Locator {
	c := cudaLocator{
		Locator: libraries,
	}
	return &c
}

// Locate returns the path to the libcuda.so.RMVERSION file.
// libcuda.so is prefixed to the specified pattern.
func (l *cudaLocator) Locate(pattern string) ([]string, error) {
	return l.Locator.Locate("libcuda.so" + pattern)
}
