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
	"io/fs"
)

// An Option defines a functional option for constructing a producer.
type Option func(*options) error

type options struct {
	specFormat           SpecFormat
	overwrite            bool
	permissions          fs.FileMode
	detectMinimumVersion bool
}

// WithDetectMinimumVersion toggles whether a minimum version should be detected for a CDI specification.
func WithDetectMinimumVersion(detectMinimumVersion bool) Option {
	return func(o *options) error {
		o.detectMinimumVersion = detectMinimumVersion
		return nil
	}
}

// WithSpecFormat sets the output format of a CDI specification.
func WithSpecFormat(format SpecFormat) Option {
	return func(o *options) error {
		switch format {
		case SpecFormatJSON, SpecFormatYAML:
			o.specFormat = format
		default:
			return fmt.Errorf("invalid CDI spec format %v", format)
		}
		return nil
	}
}

// WithOverwrite specifies whether a producer should overwrite a CDI spec when
// saving to file.
func WithOverwrite(overwrite bool) Option {
	return func(o *options) error {
		o.overwrite = overwrite
		return nil
	}
}

// WithPermissions sets the file mode to be used for a saved CDI spec.
func WithPermissions(permissions fs.FileMode) Option {
	return func(o *options) error {
		o.permissions = permissions
		return nil
	}
}
