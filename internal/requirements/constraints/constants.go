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

import "fmt"

const (
	equal        = "="
	notEqual     = "!="
	less         = "<"
	lessEqual    = "<="
	greater      = ">"
	greaterEqual = ">="
)

// always is a constraint that is always met
type always struct{}

func (c always) Assert() error {
	return nil
}

func (c always) String() string {
	return "true"
}

// invalid is an invalid constraint and can never be met
type invalid string

func (c invalid) Assert() error {
	return fmt.Errorf("invalid constraint: %v", c.String())
}

// String returns the string representation of the contraint
func (c invalid) String() string {
	return string(c)
}
