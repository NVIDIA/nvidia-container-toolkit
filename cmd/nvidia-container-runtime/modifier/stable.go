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
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
)

const (
	nvidiaContainerRuntimeHookExecutable = "nvidia-container-runtime-hook"
	nvidiaContainerToolkitExecutable     = "nvidia-container-toolkit"
)

// NewStableRuntimeModifier creates an OCI spec modifier that inserts the NVIDIA Container Runtime Hook into an OCI
// spec. The specified logger is used to capture log output.
func NewStableRuntimeModifier(logger *logrus.Logger) oci.SpecModifier {
	m := stableRuntimeModifier{logger: logger}

	return &m
}

// stableRuntimeModifier modifies an OCI spec inplace, inserting the nvidia-container-runtime-hook as a
// prestart hook. If the hook is already present, no modification is made.
type stableRuntimeModifier struct {
	logger *logrus.Logger
}

// Modify applies the required modification to the incoming OCI spec, inserting the nvidia-container-runtime-hook
// as a prestart hook.
func (m stableRuntimeModifier) Modify(spec *specs.Spec) error {
	path, err := exec.LookPath(nvidiaContainerRuntimeHookExecutable)
	if err != nil {
		path = filepath.Join(config.DefaultExecutableDir, nvidiaContainerRuntimeHookExecutable)
		_, err = os.Stat(path)
		if err != nil {
			return err
		}
	}

	m.logger.Infof("Using prestart hook path: %s", path)

	args := []string{path}
	if spec.Hooks == nil {
		spec.Hooks = &specs.Hooks{}
	} else if len(spec.Hooks.Prestart) != 0 {
		for _, hook := range spec.Hooks.Prestart {
			if strings.Contains(hook.Path, nvidiaContainerRuntimeHookExecutable) {
				m.logger.Infof("existing nvidia prestart hook found in OCI spec")
				return nil
			}
		}
	}

	spec.Hooks.Prestart = append(spec.Hooks.Prestart, specs.Hook{
		Path: path,
		Args: append(args, "prestart"),
	})

	return nil
}
