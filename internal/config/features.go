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
	// AllowCUDACompatLibsFromContainer allows CUDA compat libs from a container
	// to override certain driver library mounts from the host.
	AllowCUDACompatLibsFromContainer *feature `toml:"allow-cuda-compat-libs-from-container,omitempty"`
	// AllowLDConfigFromContainer allows non-host ldconfig paths to be used.
	// If this feature flag is not set to 'true' only host-rooted config paths
	// (i.e. paths starting with an '@' are considered valid)
	AllowLDConfigFromContainer *feature `toml:"allow-ldconfig-from-container,omitempty"`
	// DisableCUDACompatLibHook, when enabled skips the injection of a specific
	// hook to process CUDA compatibility libraries.
	//
	// Note: Since this mechanism replaces the logic in the `nvidia-container-cli`,
	// toggling this feature has no effect if `allow-cuda-compat-libs-from-container` is enabled.
	DisableCUDACompatLibHook *feature `toml:"disable-cuda-compat-lib-hook,omitempty"`
	// DisableImexChannelCreation ensures that the implicit creation of
	// requested IMEX channels is skipped when invoking the nvidia-container-cli.
	DisableImexChannelCreation *feature `toml:"disable-imex-channel-creation,omitempty"`
	// IgnoreImexChannelRequests configures the NVIDIA Container Toolkit to
	// ignore IMEX channel requests through the NVIDIA_IMEX_CHANNELS envvar or
	// volume mounts.
	// This ensures that the NVIDIA Container Toolkit cannot be used to provide
	// access to an IMEX channel by simply specifying an environment variable,
	// possibly bypassing other checks by an orchestration system such as
	// kubernetes.
	IgnoreImexChannelRequests *feature `toml:"ignore-imex-channel-requests,omitempty"`
}

type feature bool

// IsEnabled checks whether a feature is explicitly enabled.
func (f *feature) IsEnabled() bool {
	if f != nil {
		return bool(*f)
	}
	return false
}
