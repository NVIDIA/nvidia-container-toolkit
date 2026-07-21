/**
# Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
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

// CheckRequirementsHookOptions defines the options that can be specified when
// creating the check-requirements hook.
type CheckRequirementsHookOptions struct {
	DriverRoot string
}

// NewCheckRequirementsHookDiscoverer creates a discoverer for a
// check-requirements hook.
func NewCheckRequirementsHookDiscoverer(hookCreator HookCreator, o *CheckRequirementsHookOptions) Discover {
	hook := hookCreator.Create(CheckRequirementsHook, o.args()...)
	if hook == nil {
		return None{}
	}
	return hook
}

func (o *CheckRequirementsHookOptions) args() []string {
	if o == nil {
		return nil
	}
	if o.DriverRoot == "" || o.DriverRoot == "/" {
		return nil
	}
	return []string{"--driver-root=" + o.DriverRoot}
}
