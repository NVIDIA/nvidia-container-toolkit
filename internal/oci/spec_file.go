/*
# Copyright (c) 2021, NVIDIA CORPORATION.  All rights reserved.
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
*/

package oci

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/opencontainers/runtime-spec/specs-go"
)

type fileSpec struct {
	memorySpec
	path string
	loader
}

type loader bool

const (
	strictLoader = loader(true)
)

var _ Spec = (*fileSpec)(nil)

// NewFileSpec creates an object that encapsulates a file-backed OCI spec.
// This can be used to read from the file, modify the spec, and write to the
// same file.
func NewFileSpec(filepath string, isStrict bool) Spec {
	var loader loader
	if isStrict {
		loader = strictLoader
	}
	oci := fileSpec{
		path:   filepath,
		loader: loader,
	}

	return &oci
}

// Load reads the contents of an OCI spec from file to be referenced internally.
// The file is opened "read-only"
func (s *fileSpec) Load() (*specs.Spec, error) {
	specFile, err := os.Open(s.path)
	if err != nil {
		return nil, fmt.Errorf("error opening OCI specification file: %v", err)
	}
	defer specFile.Close()

	spec, err := s.loadFrom(specFile)
	if err != nil {
		return nil, fmt.Errorf("error loading OCI specification from file: %v", err)
	}
	s.Spec = spec
	return s.Spec, nil
}

// LoadFrom reads the contents of the OCI spec from the specified io.Reader.
func (isStrict loader) loadFrom(reader io.Reader) (*specs.Spec, error) {
	decoder := json.NewDecoder(reader)
	if isStrict {
		decoder.DisallowUnknownFields()
	}

	var spec specs.Spec

	err := decoder.Decode(&spec)
	if err != nil {
		return nil, fmt.Errorf("error reading OCI specification: %v", err)
	}

	return &spec, nil
}

// Modify applies the specified SpecModifier to the stored OCI specification.
func (s *fileSpec) Modify(m SpecModifier) error {
	return s.memorySpec.Modify(m)
}

// Flush writes the stored OCI specification to the filepath specified by the path member.
// The file is written to a temporary file first and then atomically replaced,
// preventing data loss if the write operation fails mid-way.
func (s fileSpec) Flush() error {
	if s.Spec == nil {
		return fmt.Errorf("no OCI specification loaded")
	}

	// Resolve symlinks so we write through to the target, preserving the
	// os.Create semantics of following symlinks rather than replacing them.
	writePath := s.path
	if info, err := os.Lstat(s.path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		if resolved, err := filepath.EvalSymlinks(s.path); err == nil {
			writePath = resolved
		}
	}

	dir := filepath.Dir(writePath)

	// Stat the original file (if it exists) so its mode can be preserved on the
	// temp file. os.CreateTemp creates the temp with 0600; without this, a
	// successful os.Rename would downgrade the spec (e.g. 0644 -> 0600) and drop
	// the original owner, which is a mode regression on top of the data-loss fix.
	var originalMode os.FileMode
	if info, err := os.Stat(writePath); err == nil {
		originalMode = info.Mode().Perm()
	}

	tmpFile, err := os.CreateTemp(dir, ".oci-spec-*.tmp")
	if err != nil {
		return fmt.Errorf("error creating temp OCI specification file: %v", err)
	}
	tmpPath := tmpFile.Name()

	// Restore the original mode on the temp file before it replaces the
	// original. On Windows only the write bit is meaningful; other bits are
	// ignored by the OS.
	if originalMode != 0 {
		if err := os.Chmod(tmpPath, originalMode); err != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
			return fmt.Errorf("error setting temp OCI specification file mode: %v", err)
		}
	}

	err = flushTo(s.Spec, tmpFile)
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("error writing OCI specification: %v", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("error closing temp OCI specification file: %v", err)
	}

	if err := os.Rename(tmpPath, writePath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("error replacing OCI specification file: %v", err)
	}

	return nil
}

// flushTo writes the stored OCI specification to the specified io.Writer.
func flushTo(spec *specs.Spec, writer io.Writer) error {
	if spec == nil {
		return nil
	}
	encoder := json.NewEncoder(writer)

	err := encoder.Encode(spec)
	if err != nil {
		return fmt.Errorf("error writing OCI specification: %v", err)
	}

	return nil
}
