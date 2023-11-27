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
	"path/filepath"
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/transform"
)

// transformer transforms roots of paths.
type transformer struct {
	root       string
	targetRoot string
}

// New creates a root transformer using the specified options.
func New(opts ...Option) transform.Transformer {
	b := &builder{}
	for _, opt := range opts {
		opt(b)
	}
	return b.build()
}

func (t transformer) transformPath(path string) string {
	if !strings.HasPrefix(path, t.root) {
		return path
	}

	return filepath.Join(t.targetRoot, strings.TrimPrefix(path, t.root))
}
