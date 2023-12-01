/*
# Copyright (c) 2021, NVIDIA CORPORATION.  All rights reserved.
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
*/

package modifier

import (
	"path/filepath"

	"github.com/opencontainers/runtime-spec/specs-go"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

// NewStableRuntimeModifier creates an OCI spec modifier that inserts the NVIDIA Container Runtime Hook into an OCI
// spec. The specified logger is used to capture log output.
func NewStableRuntimeModifier(logger logger.Interface, nvidiaContainerRuntimeHookPath string) oci.SpecModifier {
	m := stableRuntimeModifier{
		logger:                         logger,
		nvidiaContainerRuntimeHookPath: nvidiaContainerRuntimeHookPath,
	}

	return &m
}

// stableRuntimeModifier modifies an OCI spec inplace, inserting the nvidia-container-runtime-hook as a
// prestart hook. If the hook is already present, no modification is made.
type stableRuntimeModifier struct {
	logger                         logger.Interface
	nvidiaContainerRuntimeHookPath string
}

// Modify applies the required modification to the incoming OCI spec, inserting the nvidia-container-runtime-hook
// as a prestart hook.
func (m stableRuntimeModifier) Modify(spec *specs.Spec) error {
	// If an NVIDIA Container Runtime Hook already exists, we don't make any modifications to the spec.
	if spec.Hooks != nil {
		for _, hook := range spec.Hooks.Prestart {
			hook := hook
			if isNVIDIAContainerRuntimeHook(&hook) {
				m.logger.Infof("Existing nvidia prestart hook (%v) found in OCI spec", hook.Path)
				return nil
			}
		}
	}

	path := m.nvidiaContainerRuntimeHookPath
	m.logger.Infof("Using prestart hook path: %v", path)
	args := []string{filepath.Base(path)}
	if spec.Hooks == nil {
		spec.Hooks = &specs.Hooks{}
	}
	spec.Hooks.Prestart = append(spec.Hooks.Prestart, specs.Hook{
		Path: path,
		Args: append(args, "prestart"),
	})

	return nil
}
