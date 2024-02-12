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

package devices

type builder struct {
	asMap  devices
	filter func(string) bool
}

// New creates a new devices struct with the specified options.
func New(opts ...Option) Devices {
	b := &builder{}
	for _, opt := range opts {
		opt(b)
	}

	if b.filter == nil {
		b.filter = func(string) bool { return false }
	}

	devices := make(devices)
	for k, v := range b.asMap {
		if b.filter(string(k)) {
			continue
		}
		devices[k] = v
	}
	return devices
}

type Option func(*builder)

// WithDeviceToMajor specifies an explicit device name to major number map.
func WithDeviceToMajor(deviceToMajor map[string]int) Option {
	return func(b *builder) {
		b.asMap = make(devices)
		for name, major := range deviceToMajor {
			b.asMap[Name(name)] = Major(major)
		}
	}
}

// WithFilter specifies a filter to exclude devices.
func WithFilter(filter func(string) bool) Option {
	return func(b *builder) {
		b.filter = filter
	}
}
