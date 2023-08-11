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

package ocihook

// podmanHook is the hook configuration structure.
// This is taken from `Hook` at https://github.com/containers/podman/blob/3c53200e9d61fdf95fe1da825bb2a89372551350/pkg/hooks/1.0.0/hook.go#L18
type podmanHook struct {
	Version string   `json:"version"`
	Hook    specHook `json:"hook"`
	When    When     `json:"when"`
	Stages  []string `json:"stages"`
}

// specHook specifies a command that is run at a particular event in the lifecycle of a container
// This is taken from `Hook` at https://github.com/opencontainers/runtime-spec/blob/9ee22abf867e374c5464c7bbe0d0db01482254ab/specs-go/config.go#L128
type specHook struct {
	Path    string   `json:"path"`
	Args    []string `json:"args,omitempty"`
	Env     []string `json:"env,omitempty"`
	Timeout *int     `json:"timeout,omitempty"`
}

// When holds hook-injection conditions.
// This is taken from `When` at https://github.com/containers/podman/blob/3c53200e9d61fdf95fe1da825bb2a89372551350/pkg/hooks/1.0.0/when.go#L11
type When struct {
	Always        *bool             `json:"always,omitempty"`
	Annotations   map[string]string `json:"annotations,omitempty"`
	Commands      []string          `json:"commands,omitempty"`
	HasBindMounts *bool             `json:"hasBindMounts,omitempty"`

	// Or enables any-of matching.
	//
	// Deprecated: this property is for is backwards-compatibility with
	// 0.1.0 hooks.  It will be removed when we drop support for them.
	Or bool `json:"-"`
}
