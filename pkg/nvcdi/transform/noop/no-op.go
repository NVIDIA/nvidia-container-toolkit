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

package noop

import (
	"tags.cncf.io/container-device-interface/specs-go"

	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/transform"
)

type noop struct{}

var _ transform.Transformer = (*noop)(nil)

// New returns a no-op transformer.
func New() transform.Transformer {
	return noop{}
}

// Transform is a no-op for a noop transformer.
func (n noop) Transform(spec *specs.Spec) error {
	return nil
}
