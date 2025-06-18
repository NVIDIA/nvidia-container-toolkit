/**
# SPDX-FileCopyrightText: Copyright (c) 2025 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
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

package discover

var _ Discover = (*EnvVar)(nil)

// Devices returns an empty list of devices for a EnvVar discoverer.
func (e EnvVar) Devices() ([]Device, error) {
	return nil, nil
}

// EnvVars returns an empty list of envs for a EnvVar discoverer.
func (e EnvVar) EnvVars() ([]EnvVar, error) {
	return []EnvVar{e}, nil
}

// Mounts returns an empty list of mounts for a EnvVar discoverer.
func (e EnvVar) Mounts() ([]Mount, error) {
	return nil, nil
}

// Hooks allows the Hook type to also implement the Discoverer interface.
// It returns a single hook
func (e EnvVar) Hooks() ([]Hook, error) {
	return nil, nil
}
