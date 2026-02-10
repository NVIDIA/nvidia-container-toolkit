/**
# SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
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

package lookup

import (
	"errors"
)

type optionalLocator struct {
	wraps Locator
}

// AsOptional converts the specified Locator to a Locator that does not raise an
// error if no candidate can be found for a specified pattern.
func AsOptional(l Locator) Locator {
	return &optionalLocator{
		wraps: l,
	}
}

func (l *optionalLocator) Locate(pattern string) ([]string, error) {
	candidates, err := l.wraps.Locate(pattern)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	return candidates, err
}

type requiredLocator struct {
	wraps Locator
}

func asRequired(l Locator) Locator {
	return &requiredLocator{
		wraps: l,
	}
}

func (l *requiredLocator) Locate(pattern string) ([]string, error) {
	candidates, err := l.wraps.Locate(pattern)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, notFound.NewError(pattern)
	}
	return candidates, nil
}
