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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"tags.cncf.io/container-device-interface/pkg/cdi"
	producer "tags.cncf.io/container-device-interface/pkg/cdi-producer"
	"tags.cncf.io/container-device-interface/specs-go"

	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/transform"
)

type spec struct {
	*specs.Spec
	format          string
	permissions     os.FileMode
	transformOnSave transform.Transformer
}

var _ Interface = (*spec)(nil)

// New creates a new spec with the specified options.
func New(opts ...Option) (Interface, error) {
	return newBuilder(opts...).Build()
}

type Validator interface {
	Validate(*specs.Spec) error
}

type Validators []Validator

func (v Validators) Validate(s *specs.Spec) error {
	for _, vv := range v {
		if vv == nil {
			continue
		}
		err := vv.Validate(s)
		if err != nil {
			return err
		}
	}
	return nil
}

type transfromAsValidator struct {
	transform.Transformer
}

func fromTransform(t transform.Transformer) Validator {
	if t == nil {
		return nil
	}
	return &transfromAsValidator{t}
}

func (t *transfromAsValidator) Validate(s *specs.Spec) error {
	if t == nil || t.Transformer == nil {
		return nil
	}
	if err := t.Transform(s); err != nil {
		return fmt.Errorf("error applying transform: %w", err)
	}
	return nil
}

// Save writes the spec to the specified path and overwrites the file if it exists.
func (s *spec) Save(path string) error {
	pathWithExtension := s.ensureExtension(path)

	return producer.Save(s.Raw(), pathWithExtension,
		s.producerOptions()...,
	)
}

// WriteTo writes the spec to the specified writer.
func (s *spec) WriteTo(w io.Writer) (int64, error) {
	return producer.WriteTo(s.Raw(), w,
		s.producerOptions()...,
	)
}

func (s *spec) producerOptions() []producer.Option {
	var validators Validators
	if s.transformOnSave != nil {
		validators = append(validators, fromTransform(s.transformOnSave))
	}
	validators = append(validators, cdi.SpecContentValidator)

	return []producer.Option{
		producer.WithOutputFormat(s.format),
		producer.WithOverwrite(true),
		producer.WithPermissions(s.permissions),
		producer.WithValidator(validators),
	}
}

// Raw returns a pointer to the raw spec.
func (s *spec) Raw() *specs.Spec {
	return s.Spec
}

func (s *spec) ensureExtension(filename string) string {
	if filename == "" {
		return ""
	}
	ext := filepath.Ext(filename)
	switch ext {
	case ".yaml", ".json":
		return filename
	case ".yml":
		return strings.TrimSuffix(filename, ".yml") + ".yaml"
	default:
		return filename + "." + s.format
	}
}
