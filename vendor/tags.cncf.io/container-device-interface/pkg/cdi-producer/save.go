/*
   Copyright © 2026 The CDI Authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package producer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	orderedyaml "gopkg.in/yaml.v3"

	cdi "tags.cncf.io/container-device-interface/specs-go"
)

type validator interface {
	Validate(*cdi.Spec) error
}

type options struct {
	format      string
	overwrite   bool
	permissions os.FileMode
	validator   validator
}

// An Option is used to supply additional configuration to the functions of the
// producer API.
type Option func(*options)

// Save a CDI specification to the requested path.
func Save(raw *cdi.Spec, path string, opts ...Option) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("a path is required")
	}
	dir, filename, err := splitPath(path)
	if err != nil {
		return err
	}
	if filename == "" {
		return fmt.Errorf("unexpected empty filename")
	}
	if dir == "" {
		return fmt.Errorf("unexpected empty directory name")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create Spec dir: %w", err)
	}

	o := populateOptions(opts...)

	tmpFile, err := os.CreateTemp(dir, "spec.*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create Spec file: %w", err)
	}
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	if o.permissions != 0 {
		err = tmpFile.Chmod(o.permissions)
		if err != nil {
			return fmt.Errorf("failed to set file permissions: %w", err)
		}
	}

	format := o.formatFromFilename(filename)
	_, err = WriteTo(raw, tmpFile, append(opts, WithOutputFormat(format))...)
	_ = tmpFile.Close()
	if err != nil {
		return fmt.Errorf("failed to write Spec file: %w", err)
	}

	return renameIn(dir, filepath.Base(tmpFile.Name()), filename, o.overwrite)
}

// WriteTo writes a CDI specification to a writer.
func WriteTo(raw *cdi.Spec, w io.Writer, opts ...Option) (int64, error) {
	o := populateOptions(opts...)

	if err := o.Validate(raw); err != nil {
		return 0, fmt.Errorf("spec validation failed: %w", err)
	}

	data, err := o.marshal(raw)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal spec: %w", err)
	}

	n, err := w.Write(data)
	if err != nil {
		return 0, err
	}
	return int64(n), nil
}

// EnsureExtension adds an extension defined by the specified options if
// required.
func EnsureExtension(path string, opts ...Option) string {
	o := populateOptions(opts...)
	ext := filepath.Ext(path)
	switch ext {
	case ".yaml", ".json":
		return path
	default:
		return path + "." + o.format
	}
}

func populateOptions(opts ...Option) *options {
	o := &options{
		format:      "yaml",
		overwrite:   false,
		permissions: 0644,
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// WithOutputFormat sets the format (e.g. YAML or JSON) to use when outputting a
// CDI specification.
func WithOutputFormat(format string) Option {
	return func(o *options) {
		o.format = format
	}
}

// WithOverwrite defines whether an existing CDI specification file should be
// overwritten if it exists at the specified path.
func WithOverwrite(overwrite bool) Option {
	return func(o *options) {
		o.overwrite = overwrite
	}
}

// WithPermissions sets the file permissions for the file created when
// outputting a CDI specification.
func WithPermissions(permissions os.FileMode) Option {
	return func(o *options) {
		o.permissions = permissions
	}
}

// WithValidator sets a validator to apply to a CDI specification before
// outputting it.
func WithValidator(validator validator) Option {
	return func(o *options) {
		o.validator = validator
	}
}

func (o *options) formatFromFilename(path string) string {
	ext := filepath.Ext(path)
	switch ext {
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	default:
		return o.format
	}
}

// splitPath separates a path into a directory and a filename.
// If the directory is unspecified or '.' the current working directory is
// returned instead.
func splitPath(path string) (string, string, error) {
	path = filepath.Clean(path)
	if err := assertNotDirectory(path); err != nil {
		return "", "", err
	}
	dir, filename := filepath.Split(path)
	if dir != "." {
		return dir, filename, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("error getting current directory: %w", err)
	}
	return cwd, filename, nil
}

func assertNotDirectory(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("specified path is a directory")
	}
	return nil
}

func (o *options) marshal(v any) ([]byte, error) {
	switch o.format {
	case "yaml":
		data, err := orderedyaml.Marshal(v)
		if err != nil {
			return nil, err
		}
		data = append([]byte("---\n"), data...)
		return data, err
	case "json":
		return json.Marshal(v)
	default:
		return nil, fmt.Errorf("invalid output format: %v", o.format)
	}
}

// Validate a CDI specification using the supplied options.
// If no validator is specified, validation always succeeds.
func (o *options) Validate(raw *cdi.Spec) error {
	if o == nil || o.validator == nil {
		return nil
	}
	return o.validator.Validate(raw)
}

type validatorFunction func(*cdi.Spec) error

func (v validatorFunction) Validate(raw *cdi.Spec) error {
	if v == nil {
		return nil
	}
	return v(raw)
}
