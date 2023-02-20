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

	"github.com/container-orchestrated-devices/container-device-interface/pkg/cdi"
	"github.com/container-orchestrated-devices/container-device-interface/specs-go"
)

type spec specs.Spec

var _ Interface = (*spec)(nil)

// New creates a new spec with the specified deivice specs and edits.
func New(deviceSpecs []specs.Device, edits specs.ContainerEdits) (Interface, error) {
	s := specs.Spec{
		// TODO: Should be set through an option
		Version: "NOT_SET",
		// TODO: Should be set through an option
		Kind: "nvidia.com/gpu",
		// TODO: Should be set through an option
		Devices: deviceSpecs,
		// TODO: Should be set through an option
		ContainerEdits: edits,
	}

	minVersion, err := cdi.MinimumRequiredVersion(&s)
	if err != nil {
		return nil, fmt.Errorf("failed to get minumum required CDI spec version: %v", err)
	}
	s.Version = minVersion

	return (*spec)(&s), nil
}

// Save writes the spec to the specified path and overwrites the file if it exists.
func (s *spec) Save(path string) error {
	return cdi.WriteSpec(s, path, true)
}

// Raw returns a pointer to the raw spec.
func (s *spec) Raw() *specs.Spec {
	return (*specs.Spec)(s)
}
