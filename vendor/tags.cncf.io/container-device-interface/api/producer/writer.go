/*
   Copyright © 2024 The CDI Authors

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
	"fmt"
	"io"
	"os"
	"path/filepath"

	cdi "tags.cncf.io/container-device-interface/specs-go"
)

// A SpecWriter defines a structure for outputting CDI specifications.
type SpecWriter struct {
	options
}

// NewSpecWriter creates a spec writer with the supplied options.
func NewSpecWriter(opts ...Option) (*SpecWriter, error) {
	sw := &SpecWriter{
		options: options{
			overwrite: true,
			// TODO: This could be updated to 0644 to be world-readable.
			permissions:          0600,
			specFormat:           DefaultSpecFormat,
			detectMinimumVersion: false,
		},
	}
	for _, opt := range opts {
		err := opt(&sw.options)
		if err != nil {
			return nil, err
		}
	}
	return sw, nil
}

// Save writes a CDI spec to a file with the specified name.
// If the filename ends in a supported extension, the format implied by the
// extension takes precedence over the format with which the SpecWriter was
// configured.
func (p *SpecWriter) Save(spec *cdi.Spec, filename string) (string, error) {
	filename, outputFormat := p.specFormat.normalizeFilename(filename)

	options := p.options
	options.specFormat = outputFormat
	specFormatter := specFormatter{
		Spec:    spec,
		options: options,
	}

	dir := filepath.Dir(filename)
	if dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("failed to create Spec dir: %w", err)
		}
	}

	tmp, err := os.CreateTemp(dir, "spec.*.tmp")
	if err != nil {
		return "", fmt.Errorf("failed to create Spec file: %w", err)
	}

	_, err = specFormatter.WriteTo(tmp)
	tmp.Close()
	if err != nil {
		return "", fmt.Errorf("failed to write Spec file: %w", err)
	}

	if err := os.Chmod(tmp.Name(), p.permissions); err != nil {
		return "", fmt.Errorf("failed to set permissions on spec file: %w", err)
	}

	err = renameIn(dir, filepath.Base(tmp.Name()), filepath.Base(filename), p.overwrite)
	if err != nil {
		_ = os.Remove(tmp.Name())
		return "", fmt.Errorf("failed to write Spec file: %w", err)
	}
	return filename, nil
}

// WriteSpecTo writes the specified spec to the specified writer.
func (p *SpecWriter) WriteSpecTo(spec *cdi.Spec, w io.Writer) (int64, error) {
	specFormatter := specFormatter{
		Spec:    spec,
		options: p.options,
	}

	return specFormatter.WriteTo(w)
}
