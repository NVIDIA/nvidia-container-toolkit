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
	"strings"

	oci "github.com/opencontainers/runtime-spec/specs-go"
)

type fileSpec struct {
	*oci.Spec
	path string
}

var _ Spec = (*fileSpec)(nil)

// NewSpecFromArgs creates fileSpec based on the command line arguments passed to the
// application
func NewSpecFromArgs(args []string) (Spec, string, error) {
	bundleDir, err := GetBundleDir(args)
	if err != nil {
		return nil, "", fmt.Errorf("error getting bundle directory: %v", err)
	}

	ociSpecPath := GetSpecFilePath(bundleDir)

	ociSpec := NewSpecFromFile(ociSpecPath)

	return ociSpec, bundleDir, nil
}

// NewSpecFromFile creates an object that encapsulates a file-backed OCI spec.
// This can be used to read from the file, modify the spec, and write to the
// same file.
func NewSpecFromFile(filepath string) Spec {
	oci := fileSpec{
		path: filepath,
	}

	return &oci
}

// Load reads the contents of an OCI spec from file to be referenced internally.
// The file is opened "read-only"
func (s *fileSpec) Load() error {
	specFile, err := os.Open(s.path)
	if err != nil {
		return fmt.Errorf("error opening OCI specification file: %v", err)
	}
	defer specFile.Close()

	return s.loadFrom(specFile)
}

// loadFrom reads the contents of the OCI spec from the specified io.Reader.
func (s *fileSpec) loadFrom(reader io.Reader) error {
	decoder := json.NewDecoder(reader)

	var spec oci.Spec
	err := decoder.Decode(&spec)
	if err != nil {
		return fmt.Errorf("error reading OCI specification: %v", err)
	}

	s.Spec = &spec
	return nil
}

// Modify applies the specified SpecModifier to the stored OCI specification.
func (s *fileSpec) Modify(f SpecModifier) error {
	if s.Spec == nil {
		return fmt.Errorf("no spec loaded for modification")
	}
	return f(s.Spec)
}

// Flush writes the stored OCI specification to the filepath specifed by the path member.
// The file is truncated upon opening, overwriting any existing contents.
func (s fileSpec) Flush() error {
	if s.Spec == nil {
		return fmt.Errorf("no OCI specification loaded")
	}

	specFile, err := os.Create(s.path)
	if err != nil {
		return fmt.Errorf("error opening OCI specification file: %v", err)
	}
	defer specFile.Close()

	return s.flushTo(specFile)
}

// flushTo writes the stored OCI specification to the specified io.Writer.
func (s fileSpec) flushTo(writer io.Writer) error {
	if s.Spec == nil {
		return nil
	}
	encoder := json.NewEncoder(writer)

	err := encoder.Encode(s.Spec)
	if err != nil {
		return fmt.Errorf("error writing OCI specification: %v", err)
	}

	return nil
}

// LookupEnv mirrors os.LookupEnv for the OCI specification. It
// retrieves the value of the environment variable named
// by the key. If the variable is present in the environment the
// value (which may be empty) is returned and the boolean is true.
// Otherwise the returned value will be empty and the boolean will
// be false.
func (s fileSpec) LookupEnv(key string) (string, bool) {
	if s.Spec == nil || s.Spec.Process == nil {
		return "", false
	}

	for _, env := range s.Spec.Process.Env {
		if !strings.HasPrefix(env, key) {
			continue
		}

		parts := strings.SplitN(env, "=", 2)
		if parts[0] == key {
			if len(parts) < 2 {
				return "", true
			}
			return parts[1], true
		}
	}

	return "", false
}
