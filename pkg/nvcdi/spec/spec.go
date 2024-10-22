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

package spec

import (
	"io"

	"tags.cncf.io/container-device-interface/api/producer"
	cdi "tags.cncf.io/container-device-interface/specs-go"

	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/transform"
)

type spec struct {
	raw             *cdi.Spec
	transformOnSave transform.Transformer
	*producer.SpecWriter
}

var _ Interface = (*spec)(nil)

// New creates a new spec with the specified options.
func New(opts ...Option) (Interface, error) {
	return newBuilder(opts...).Build()
}

// Raw returns a pointer to the raw spec.
func (s *spec) Raw() *cdi.Spec {
	return s.raw
}

// Save writes the spec to the specified path and overwrites the file if it exists.
func (s *spec) Save(path string) error {
	if s.transformOnSave != nil {
		err := s.transformOnSave.Transform(s.raw)
		if err != nil {
			return err
		}
	}
	// TODO: We should add validation here.
	_, err := s.SpecWriter.Save(s.raw, path)
	return err
}

// WriteTo writes the configured spec to the specified writer.
func (s *spec) WriteTo(w io.Writer) (int64, error) {
	if s.transformOnSave != nil {
		err := s.transformOnSave.Transform(s.raw)
		if err != nil {
			return 0, err
		}
	}
	// TODO: We should add validation here.
	return s.SpecWriter.WriteSpecTo(s.raw, w)
}
