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

package config

// features specifies a set of named features.
type features struct {
	// DisableImexChannelCreation ensures that the implicit creation of
	// requested IMEX channels is skipped when invoking the nvidia-container-cli.
	DisableImexChannelCreation *feature `toml:"disable-imex-channel-creation,omitempty"`
}

//nolint:unused
type feature bool

// IsEnabled checks whether a feature is explicitly enabled.
//
//nolint:unused
func (f *feature) IsEnabled() bool {
	if f != nil {
		return bool(*f)
	}
	return false
}
