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

// Option defines a functional option for configuring a transormer.
type Option func(*builder)

// WithRoot sets the (from) root for the root transformer.
func WithRoot(root string) Option {
	return func(b *builder) {
		b.root = root
	}
}

// WithTargetRoot sets the (to) target root for the root transformer.
func WithTargetRoot(root string) Option {
	return func(b *builder) {
		b.targetRoot = root
	}
}

// WithRelativeTo sets whether the specified root is relative to the host or container.
func WithRelativeTo(relativeTo string) Option {
	return func(b *builder) {
		b.relativeTo = relativeTo
	}
}
