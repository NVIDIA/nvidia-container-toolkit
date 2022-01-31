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

package main

import (
	"os"
	"os/exec"
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/runtime"
	"github.com/opencontainers/runtime-spec/specs-go"
	log "github.com/sirupsen/logrus"
)

// newNvidiaContainerRuntime is a constructor for a standard runtime shim. This uses
// a ModifyingRuntimeWrapper to apply the required modifications before execing to the
// specified low-level runtime
func newNvidiaContainerRuntime(logger *log.Logger, lowlevelRuntime oci.Runtime, ociSpec oci.Spec) (oci.Runtime, error) {
	modifier := addHookModifier{logger: logger}

	r := runtime.NewModifyingRuntimeWrapper(
		logger,
		lowlevelRuntime,
		ociSpec,
		modifier,
	)

	return r, nil
}

// addHookModifier modifies an OCI spec inplace, inserting the nvidia-container-runtime-hook as a
// prestart hook. If the hook is already present, no modification is made.
type addHookModifier struct {
	logger *log.Logger
}

// Modify applies the required modification to the incoming OCI spec, inserting the nvidia-container-runtime-hook
// as a prestart hook.
func (m addHookModifier) Modify(spec *specs.Spec) error {
	path, err := exec.LookPath("nvidia-container-runtime-hook")
	if err != nil {
		path = hookDefaultFilePath
		_, err = os.Stat(path)
		if err != nil {
			return err
		}
	}

	m.logger.Printf("prestart hook path: %s\n", path)

	args := []string{path}
	if spec.Hooks == nil {
		spec.Hooks = &specs.Hooks{}
	} else if len(spec.Hooks.Prestart) != 0 {
		for _, hook := range spec.Hooks.Prestart {
			if !strings.Contains(hook.Path, "nvidia-container-runtime-hook") {
				continue
			}
			m.logger.Println("existing nvidia prestart hook in OCI spec file")
			return nil
		}
	}

	spec.Hooks.Prestart = append(spec.Hooks.Prestart, specs.Hook{
		Path: path,
		Args: append(args, "prestart"),
	})

	return nil
}
