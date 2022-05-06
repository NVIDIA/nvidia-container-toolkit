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
)

// binary represents a binary operation. This can be used to compare a specified
// property to a value
type binary struct {
	left     Property
	operator string
	right    string
}

// String returns the string representation of the binary comparator
func (c binary) String() string {
	return fmt.Sprintf("%v%v%v", c.left.Name(), c.operator, c.right)
}

// Assert compares the property to the required value using the supplied comparator
func (c binary) Assert() error {
	satisfied, err := c.eval()
	if err != nil {
		return err
	}
	if satisfied {
		return nil
	}

	// error_setx(err, "unsatisfied condition: %s, please update your driver to a newer version, or use an earlier cuda container", predicate_format);
	return fmt.Errorf("unsatisfied condition: %v (%v)", c.String(), c.left.String())
}

func (c binary) eval() (bool, error) {
	if c.left == nil {
		return true, nil
	}

	compare, err := c.left.CompareTo(c.right)
	if err != nil {
		return false, err
	}

	switch string(c.operator) {
	case equal:
		return compare == 0, nil
	case notEqual:
		return compare != 0, nil
	case less:
		return compare < 0, nil
	case lessEqual:
		return compare <= 0, nil
	case greater:
		return compare > 0, nil
	case greaterEqual:
		return compare >= 0, nil
	}

	return false, fmt.Errorf("invalid operator %v", c.operator)
}
