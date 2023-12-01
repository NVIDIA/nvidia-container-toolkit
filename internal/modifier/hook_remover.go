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

package modifier

import (
	"path/filepath"

	"github.com/opencontainers/runtime-spec/specs-go"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

// nvidiaContainerRuntimeHookRemover is a spec modifer that detects and removes inserted nvidia-container-runtime hooks
type nvidiaContainerRuntimeHookRemover struct {
	logger logger.Interface
}

var _ oci.SpecModifier = (*nvidiaContainerRuntimeHookRemover)(nil)

// Modify removes any NVIDIA Container Runtime hooks from the provided spec
func (m nvidiaContainerRuntimeHookRemover) Modify(spec *specs.Spec) error {
	if spec == nil {
		return nil
	}

	if spec.Hooks == nil {
		return nil
	}

	if len(spec.Hooks.Prestart) == 0 {
		return nil
	}

	var newPrestart []specs.Hook

	for _, hook := range spec.Hooks.Prestart {
		hook := hook
		if isNVIDIAContainerRuntimeHook(&hook) {
			m.logger.Debugf("Removing hook %v", hook)
			continue
		}
		newPrestart = append(newPrestart, hook)
	}

	if len(newPrestart) != len(spec.Hooks.Prestart) {
		m.logger.Debugf("Updating 'prestart' hooks to %v", newPrestart)
		spec.Hooks.Prestart = newPrestart
	}

	return nil
}

// isNVIDIAContainerRuntimeHook checks if the provided hook is an nvidia-container-runtime-hook
// or nvidia-container-toolkit hook. These are included, for example, by the non-experimental
// nvidia-container-runtime or docker when specifying the --gpus flag.
func isNVIDIAContainerRuntimeHook(hook *specs.Hook) bool {
	bins := map[string]struct{}{
		config.NVIDIAContainerRuntimeHookExecutable: {},
		config.NVIDIAContainerToolkitExecutable:     {},
	}

	_, exists := bins[filepath.Base(hook.Path)]

	return exists
}
