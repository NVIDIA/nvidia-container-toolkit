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
	"fmt"
	"path/filepath"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
)

// nvidiaContainerRuntimeHookRemover is a spec modifer that detects and removes inserted nvidia-container-runtime hooks
type nvidiaContainerRuntimeHookRemover struct {
	logger *logrus.Logger
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

	var updateRequired bool
	newPrestart := make([]specs.Hook, 0, len(spec.Hooks.Prestart))

	for _, hook := range spec.Hooks.Prestart {
		if isNVIDIAContainerRuntimeHook(&hook) {
			m.logger.Warnf("Found existing NVIDIA Container Runtime Hook: %v", hook)
			updateRequired = true
			continue
		}
		newPrestart = append(newPrestart, hook)
	}

	if updateRequired {
		// TODO: Once we have updated the hook implementation to give an error if invoked incorrectly, we will update the spec hooks here instead of just logging.
		// We can then also use a boolean to track whether this is required instead of storing the removed hooks
		// spec.Hooks.Prestart = newPrestart
		m.logger.Debugf("Updating 'prestart' hooks to %v", newPrestart)
		return fmt.Errorf("spec already contains required 'prestart' hook")
	}

	return nil
}

// isNVIDIAContainerRuntimeHook checks if the provided hook is an nvidia-container-runtime-hook
// or nvidia-container-toolkit hook. These are included, for example, by the non-experimental
// nvidia-container-runtime or docker when specifying the --gpus flag.
func isNVIDIAContainerRuntimeHook(hook *specs.Hook) bool {
	lookFor := map[string]bool{
		nvidiaContainerRuntimeHookExecutable: true,
		nvidiaContainerToolkitExecutable:     true,
	}
	base := filepath.Base(hook.Path)

	return lookFor[base]
}
