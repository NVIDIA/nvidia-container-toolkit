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

package runtime

import (
	"fmt"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/modify"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
	log "github.com/sirupsen/logrus"
)

type modifyingRuntimeWrapper struct {
	logger   *log.Logger
	runtime  oci.Runtime
	ociSpec  oci.Spec
	modifier modify.Modifier
}

var _ oci.Runtime = (*modifyingRuntimeWrapper)(nil)

// NewModifyingRuntimeWrapperWithLogger creates a runtime wrapper that applies the specified modifier to the OCI specification
// before invoking the wrapped runtime.
func NewModifyingRuntimeWrapperWithLogger(logger *log.Logger, runtime oci.Runtime, spec oci.Spec, modifier modify.Modifier) oci.Runtime {
	rt := modifyingRuntimeWrapper{
		logger:   logger,
		runtime:  runtime,
		ociSpec:  spec,
		modifier: modifier,
	}
	return &rt
}

// Exec checks whether a modification of the OCI specification is required and modifies it accordingly before exec-ing
// into the wrapped runtime.
func (r *modifyingRuntimeWrapper) Exec(args []string) error {
	if oci.HasCreateSubcommand(args) {
		err := r.modify()
		if err != nil {
			return fmt.Errorf("could not apply required modification to OCI specification: %v", err)
		}
		r.logger.Infof("Applied required modification to OCI specification")
	} else {
		r.logger.Infof("No modification of OCI specification required")
	}

	r.logger.Infof("Forwarding command to runtime")
	return r.runtime.Exec(args)
}

// modify loads, modifies, and flushes the OCI specification using the defined Modifier
func (r *modifyingRuntimeWrapper) modify() error {
	err := r.ociSpec.Load()
	if err != nil {
		return fmt.Errorf("error loading OCI specification for modification: %v", err)
	}

	err = r.modifier.Modify(r.ociSpec)
	if err != nil {
		return fmt.Errorf("error modifying OCI spec: %v", err)
	}

	err = r.ociSpec.Flush()
	if err != nil {
		return fmt.Errorf("error writing modified OCI specification: %v", err)
	}
	return nil
}
