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

package constraints

import (
	"fmt"
	"strings"

	"golang.org/x/mod/semver"
)

//go:generate moq -stub -out property_mock.go . Property
// Property represents a property that is used to check requirements
type Property interface {
	Name() string
	Value() (string, error)
	String() string
	CompareTo(string) (int, error)
	Validate(string) error
}

// NewStringProperty creates a string property based on the name-value pair
func NewStringProperty(name string, value string) Property {
	p := stringProperty{
		name:  name,
		value: value,
	}

	return p
}

// NewVersionProperty creates a property representing a semantic version based on the name-value pair
func NewVersionProperty(name string, value string) Property {
	p := versionProperty{
		stringProperty: stringProperty{
			name:  name,
			value: value,
		},
	}

	return p
}

// stringProperty represents a property that is used to check requirements
type stringProperty struct {
	name  string
	value string
}

type versionProperty struct {
	stringProperty
}

// Name returns a stringProperty's name
func (p stringProperty) Name() string {
	return p.name
}

// Value returns a stringProperty's value or an error if this cannot be determined
func (p stringProperty) Value() (string, error) {
	return p.value, nil
}

// CompareTo compares two strings to each other
func (p stringProperty) CompareTo(other string) (int, error) {
	value := p.value

	if value < other {
		return -1, nil
	}

	if value > other {
		return 1, nil
	}

	return 0, nil
}

// Validate returns nil for all input strings
func (p stringProperty) Validate(string) error {
	return nil
}

// String returns the string representation of the name value combination
func (p stringProperty) String() string {
	v, err := p.Value()
	if err != nil {
		return fmt.Sprintf("invalid %v: %v", p.name, err)
	}
	return fmt.Sprintf("%v=%v", p.name, v)
}

// CompareTo compares two versions to each other as semantic versions
func (p versionProperty) CompareTo(other string) (int, error) {
	if err := p.Validate(other); err != nil {
		return 0, fmt.Errorf("invailid value for %v: %v", p.name, err)
	}

	vValue := ensurePrefix(p.value, "v")
	vOther := ensurePrefix(other, "v")
	return semver.Compare(vValue, vOther), nil
}

// Validate checks whether the supplied value is a valid semantic version
func (p versionProperty) Validate(value string) error {
	if !semver.IsValid(ensurePrefix(value, "v")) {
		return fmt.Errorf("invailid value %v; expected a valid version string", value)
	}

	return nil
}

func ensurePrefix(s string, prefix string) string {
	return prefix + strings.TrimPrefix(s, prefix)
}
