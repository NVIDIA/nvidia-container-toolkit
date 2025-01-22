/**
# Copyright (c) 2022, NVIDIA CORPORATION.  All rights reserved.
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

package modifier

import (
	"github.com/opencontainers/runtime-spec/specs-go"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

type List []oci.SpecModifier

// Merge merges a set of OCI specification modifiers as a list.
// This can be used to compose modifiers.
func Merge(modifiers ...oci.SpecModifier) oci.SpecModifier {
	var filteredModifiers List
	for _, m := range modifiers {
		if m == nil {
			continue
		}
		filteredModifiers = append(filteredModifiers, m)
	}

	return filteredModifiers
}

// Modify applies a list of modifiers in sequence and returns on any errors encountered.
func (m List) Modify(spec *specs.Spec) error {
	for _, mm := range m {
		if mm == nil {
			continue
		}
		err := mm.Modify(spec)
		if err != nil {
			return err
		}
	}
	return nil
}
