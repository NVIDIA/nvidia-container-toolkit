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

package flags

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Options stores options for the config commands
type Options struct {
	Config  string
	Output  string
	InPlace bool
}

// Validate checks whether the options are valid.
func (o Options) Validate() error {
	if o.InPlace && o.Output != "" {
		return fmt.Errorf("cannot specify both --in-place and --output")
	}

	return nil
}

// GetOutput returns the effective output
func (o Options) GetOutput() string {
	if o.InPlace {
		return o.Config
	}

	return o.Output
}

// EnsureOutputFolder creates the output folder if it does not exist.
// If the output folder is not specified (i.e. output to STDOUT), it is ignored.
func (o Options) EnsureOutputFolder() error {
	output := o.GetOutput()
	if output == "" {
		return nil
	}
	if dir := filepath.Dir(output); dir != "" {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// CreateOutput creates the writer for the output.
func (o Options) CreateOutput() (io.WriteCloser, error) {
	output := o.GetOutput()
	if output == "" {
		return nullCloser{os.Stdout}, nil
	}

	return os.Create(output)
}

// nullCloser is a writer that does nothing on Close.
type nullCloser struct {
	io.Writer
}

// Close is a no-op for a nullCloser.
func (d nullCloser) Close() error {
	return nil
}
