/**
# Copyright 2024 NVIDIA CORPORATION
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

package toml

import "errors"

type firstOf []Loader

func LoadFirst(loaders ...Loader) Loader {
	return firstOf(loaders)
}

func (loaders firstOf) Load() (*Tree, error) {
	var errs error
	for _, loader := range loaders {
		if loader == nil {
			continue
		}
		tree, err := loader.Load()
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		return tree, nil
	}
	return nil, errs
}
