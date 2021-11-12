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
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
	log "github.com/sirupsen/logrus"
)

// nvidiaContainerRuntime encapsulates the NVIDIA Container Runtime. It wraps the specified runtime, conditionally
// modifying the specified OCI specification before invoking the runtime.
type nvidiaContainerRuntime struct {
	logger  *log.Logger
	runtime oci.Runtime
	ociSpec oci.Spec
}

var _ oci.Runtime = (*nvidiaContainerRuntime)(nil)

// newNvidiaContainerRuntime is a constructor for a standard runtime shim.
func newNvidiaContainerRuntimeWithLogger(logger *log.Logger, runtime oci.Runtime, ociSpec oci.Spec) (oci.Runtime, error) {
	r := nvidiaContainerRuntime{
		logger:  logger,
		runtime: runtime,
		ociSpec: ociSpec,
	}

	return &r, nil
}

// Exec defines the entrypoint for the NVIDIA Container Runtime. A check is performed to see whether modifications
// to the OCI spec are required -- and applicable modifcations applied. The supplied arguments are then
// forwarded to the underlying runtime's Exec method.
func (r nvidiaContainerRuntime) Exec(args []string) error {
	if r.modificationRequired(args) {
		err := r.modifyOCISpec()
		if err != nil {
			return fmt.Errorf("error modifying OCI spec: %v", err)
		}
	}

	r.logger.Println("Forwarding command to runtime")
	return r.runtime.Exec(args)
}

// modificationRequired checks the intput arguments to determine whether a modification
// to the OCI spec is required.
func (r nvidiaContainerRuntime) modificationRequired(args []string) bool {
	if oci.HasCreateSubcommand(args) {
		r.logger.Infof("'create' command detected; modification required")
		return true
	}

	r.logger.Infof("No modification required")
	return false
}

// modifyOCISpec loads and modifies the OCI spec specified in the nvidiaContainerRuntime
// struct. The spec is modified in-place and written to the same file as the input after
// modifcationas are applied.
func (r nvidiaContainerRuntime) modifyOCISpec() error {
	err := r.ociSpec.Load()
	if err != nil {
		return fmt.Errorf("error loading OCI specification for modification: %v", err)
	}

	err = r.ociSpec.Modify(r.addNVIDIAHook)
	if err != nil {
		return fmt.Errorf("error injecting NVIDIA Container Runtime hook: %v", err)
	}

	err = r.ociSpec.Flush()
	if err != nil {
		return fmt.Errorf("error writing modified OCI specification: %v", err)
	}
	return nil
}

// addNVIDIAHook modifies the specified OCI specification in-place, inserting a
// prestart hook.
func (r nvidiaContainerRuntime) addNVIDIAHook(spec *specs.Spec) error {
	path, err := exec.LookPath("nvidia-container-runtime-hook")
	if err != nil {
		path = hookDefaultFilePath
		_, err = os.Stat(path)
		if err != nil {
			return err
		}
	}

	r.logger.Printf("prestart hook path: %s\n", path)

	args := []string{path}
	if spec.Hooks == nil {
		spec.Hooks = &specs.Hooks{}
	} else if len(spec.Hooks.Prestart) != 0 {
		for _, hook := range spec.Hooks.Prestart {
			if !strings.Contains(hook.Path, "nvidia-container-runtime-hook") {
				continue
			}
			r.logger.Println("existing nvidia prestart hook in OCI spec file")
			return nil
		}
	}

	spec.Hooks.Prestart = append(spec.Hooks.Prestart, specs.Hook{
		Path: path,
		Args: append(args, "prestart"),
	})

	return nil
}
