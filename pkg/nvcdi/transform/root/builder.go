/**
# Copyright 2023 NVIDIA CORPORATION
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

package root

import (
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/transform"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/transform/noop"
)

type builder struct {
	transformer
	relativeTo string
}

func (b *builder) build() transform.Transformer {
	if b.root == b.targetRoot {
		return noop.New()
	}

	if b.relativeTo == "container" {
		return containerRootTransformer(b.transformer)
	}
	return hostRootTransformer(b.transformer)
}
