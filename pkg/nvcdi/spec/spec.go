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
	"os"
	"path/filepath"

	"github.com/container-orchestrated-devices/container-device-interface/pkg/cdi"
	"github.com/container-orchestrated-devices/container-device-interface/specs-go"
)

type spec struct {
	*specs.Spec
	format string
}

var _ Interface = (*spec)(nil)

// New creates a new spec with the specified options.
func New(opts ...Option) (Interface, error) {
	return NewBuilder(opts...).Build()
}

// Save writes the spec to the specified path and overwrites the file if it exists.
func (s *spec) Save(path string) error {
	path = s.normalizePath(path)

	specDir := filepath.Dir(path)
	registry := cdi.GetRegistry(
		cdi.WithAutoRefresh(false),
		cdi.WithSpecDirs(specDir),
	)

	return registry.SpecDB().WriteSpec(s.Raw(), filepath.Base(path))
}

// WriteTo writes the spec to the specified writer.
func (s *spec) WriteTo(w io.Writer) (int64, error) {
	name, err := cdi.GenerateNameForSpec(s.Raw())
	if err != nil {
		return 0, err
	}

	path := s.normalizePath(name)
	tmpFile, err := os.CreateTemp("", "*"+filepath.Base(path))
	if err != nil {
		return 0, err
	}
	defer os.Remove(tmpFile.Name())

	if err := s.Save(tmpFile.Name()); err != nil {
		return 0, err
	}

	err = tmpFile.Close()
	if err != nil {
		return 0, fmt.Errorf("failed to close temporary file: %w", err)
	}

	r, err := os.Open(tmpFile.Name())
	if err != nil {
		return 0, fmt.Errorf("failed to open temporary file: %w", err)
	}
	defer r.Close()

	return io.Copy(w, r)
}

// Raw returns a pointer to the raw spec.
func (s *spec) Raw() *specs.Spec {
	return s.Spec
}

// normalizePath ensures that the specified path has a supported extension
func (s *spec) normalizePath(path string) string {
	if ext := filepath.Ext(path); ext != ".yaml" && ext != ".json" {
		path += s.extension()
	}

	return path
}

func (s *spec) extension() string {
	switch s.format {
	case "json":
		return ".json"
	case "yaml", "yml":
		return ".yaml"
	}

	return ".yaml"
}
