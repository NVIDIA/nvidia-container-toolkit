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

package image

import (
	"fmt"
	"strings"
)

type builder struct {
	env            []string
	disableRequire bool
}

// New creates a new CUDA image from the input options.
func New(opt ...Option) (CUDA, error) {
	b := &builder{}
	for _, o := range opt {
		o(b)
	}

	return b.build()
}

// build creates a CUDA image from the builder.
func (b builder) build() (CUDA, error) {
	c := make(CUDA)

	for _, e := range b.env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid environment variable: %v", e)
		}
		c[parts[0]] = parts[1]
	}

	if b.disableRequire {
		c[envNVDisableRequire] = "true"
	}

	return c, nil
}

// Option is a functional option for creating a CUDA image.
type Option func(*builder)

// WithDisableRequire sets the disable require option.
func WithDisableRequire(disableRequire bool) Option {
	return func(b *builder) {
		b.disableRequire = disableRequire
	}
}

// WithEnv sets the environment variables to use when creating the CUDA image.
func WithEnv(env []string) Option {
	return func(b *builder) {
		b.env = env
	}
}
