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
	oci "github.com/opencontainers/runtime-spec/specs-go"
)

// SpecModifier is a function that accepts a pointer to an OCI Srec and returns an
// error. The intention is that the function would modify the spec in-place.
type SpecModifier func(*oci.Spec) error

//go:generate moq -stub -out spec_mock.go . Spec

// Spec defines the operations to be performed on an OCI specification
type Spec interface {
	Load() error
	Flush() error
	Modify(SpecModifier) error
	LookupEnv(string) (string, bool)
}
