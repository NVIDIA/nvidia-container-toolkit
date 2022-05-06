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

	"github.com/sirupsen/logrus"
)

type factory struct {
	logger     *logrus.Logger
	properties map[string]Property
}

// New creates a new constraint for the supplied requirements and properties
func New(logger *logrus.Logger, requirements []string, properties map[string]Property) (Constraint, error) {
	if len(requirements) == 0 {
		return &always{}, nil
	}

	f := factory{
		logger:     logger,
		properties: properties,
	}

	var constraints []Constraint
	for _, r := range requirements {
		c, err := f.newConstraintFromRequirement(r)
		if err != nil {
			return nil, err
		}
		if c == nil {
			continue
		}
		constraints = append(constraints, c)
	}

	return AND(constraints), nil
}

// newConstraintFromRequirement takes a requirement string and generates
// the associated constraint(s). Invalid constraints are ignored.
// Each requirement can consist of multiple constraints, with space-separated constraints being ORed
// together and comma-separated constraints being ANDed together.
func (r factory) newConstraintFromRequirement(requirement string) (Constraint, error) {
	const (
		orSeparator  = " "
		andSeparator = ","
	)
	if strings.TrimSpace(requirement) == "" {
		return nil, nil
	}

	var terms []Constraint
	for _, term := range strings.Split(requirement, orSeparator) {
		var factors []Constraint
		for _, factor := range strings.Split(term, andSeparator) {
			f, err := r.parse(factor)
			if err != nil {
				return nil, err
			}
			if f == nil {
				r.logger.Debugf("Skipping unsupported constraint: %v", factor)
				continue
			}
			factors = append(factors, f)
		}

		if len(factors) == 0 {
			continue
		}

		if len(factors) == 1 {
			terms = append(terms, factors[0])
		} else {
			terms = append(terms, and(factors))
		}
	}

	return OR(terms), nil
}

// parse constructs a constraint from the specified string.
// The string is expected to be of the form [PROPERTY][OPERATOR][VALUE]
func (r factory) parse(condition string) (Constraint, error) {
	if strings.TrimSpace(condition) == "" {
		return nil, nil
	}

	operators := []string{
		notEqual,
		lessEqual,
		greaterEqual,
		equal,
		less,
		greater,
	}

	propertyEnd := strings.IndexAny(condition, "<>=!")
	if propertyEnd == -1 {
		return nil, fmt.Errorf("invalid constraint: %v", condition)
	}

	property := condition[:propertyEnd]
	condition = strings.TrimPrefix(condition, property)

	p, ok := r.properties[property]
	if !ok || p == nil {
		return nil, nil
	}

	var op string
	for _, o := range operators {
		if strings.HasPrefix(condition, o) {
			op = o
			break
		}
	}
	value := strings.TrimPrefix(condition, op)

	c := binary{
		left:     p,
		right:    value,
		operator: op,
	}
	return c, p.Validate(value)
}
