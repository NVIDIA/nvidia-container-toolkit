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

package discover

import "errors"

type firstOf []Discover

// FirstValid returns a discoverer that returns the first non-error result from a list of discoverers.
func FirstValid(discoverers ...Discover) Discover {
	var f firstOf
	for _, d := range discoverers {
		if d == nil {
			continue
		}
		f = append(f, d)
	}
	return f
}

func (f firstOf) Devices() ([]Device, error) {
	var errs error
	for _, d := range f {
		devices, err := d.Devices()
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		return devices, nil
	}
	return nil, errs
}

func (f firstOf) Hooks() ([]Hook, error) {
	var errs error
	for _, d := range f {
		hooks, err := d.Hooks()
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		return hooks, nil
	}
	return nil, errs
}

func (f firstOf) Mounts() ([]Mount, error) {
	var errs error
	for _, d := range f {
		mounts, err := d.Mounts()
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		return mounts, nil
	}
	return nil, nil
}
