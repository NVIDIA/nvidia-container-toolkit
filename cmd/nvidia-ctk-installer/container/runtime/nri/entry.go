/**
# Copyright (c) NVIDIA CORPORATION.  All rights reserved.
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

package nri

import "context"

// Runner is an NRI plugin implementation that can be registered with the container runtime.
type Runner interface {
	Start(ctx context.Context, cfg RegistrationConfig) error
	Stop()
}

// Entry associates a user-defined name and implementation with NRI registration settings.
type Entry struct {
	Name         string
	Config       RegistrationConfig
	PluginRunner Runner
}
