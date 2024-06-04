/**
# Copyright 2024 NVIDIA CORPORATION
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

type DriverOption func(*driverOptions)

func WithDriverRoot(root string) DriverOption {
	return func(do *driverOptions) {
		do.driverRoot = root
	}
}

func WithTargetDriverRoot(root string) DriverOption {
	return func(do *driverOptions) {
		do.targetDriverRoot = root
	}
}

func WithDevRoot(root string) DriverOption {
	return func(do *driverOptions) {
		do.devRoot = root
	}
}

func WithTargetDevRoot(root string) DriverOption {
	return func(do *driverOptions) {
		do.targetDevRoot = root
	}
}

type driverOptions struct {
	driverRoot       string
	targetDriverRoot string
	devRoot          string
	targetDevRoot    string
}

// NewDriverTransformer creates a transformer for transforming driver specifications.
func NewDriverTransformer(opts ...DriverOption) transform.Transformer {
	d := &driverOptions{}
	for _, opt := range opts {
		opt(d)
	}
	if d.driverRoot == "" {
		d.driverRoot = "/"
	}
	if d.targetDriverRoot == "" {
		d.targetDriverRoot = "/"
	}
	if d.devRoot == "" {
		d.devRoot = d.driverRoot
	}
	if d.targetDevRoot == "" {
		d.targetDevRoot = d.targetDriverRoot
	}

	var transformers []transform.Transformer

	if d.targetDevRoot != d.targetDriverRoot {
		devRootTransformer := New(
			WithRoot(ensureDev(d.devRoot)),
			WithTargetRoot(ensureDev(d.targetDevRoot)),
		)
		transformers = append(transformers, devRootTransformer)
	}

	driverRootTransformer := New(
		WithRoot(d.driverRoot),
		WithTargetRoot(d.targetDriverRoot),
	)
	transformers = append(transformers, driverRootTransformer)

	return transform.Merge(transformers...)
}

func ensureDev(p string) string {
	return filepath.Join(strings.TrimSuffix(filepath.Clean(p), "/dev"), "/dev")
}
