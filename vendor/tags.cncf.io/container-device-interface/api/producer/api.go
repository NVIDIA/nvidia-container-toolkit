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

import cdi "tags.cncf.io/container-device-interface/specs-go"

const (
	// DefaultSpecFormat defines the default encoding used to write CDI specs.
	DefaultSpecFormat = SpecFormatYAML

	// SpecFormatJSON defines a CDI spec formatted as JSON.
	SpecFormatJSON = SpecFormat(".json")
	// SpecFormatYAML defines a CDI spec formatted as YAML.
	SpecFormatYAML = SpecFormat(".yaml")
)

// A SpecValidator is used to validate a CDI spec.
type SpecValidator interface {
	Validate(*cdi.Spec) error
}
