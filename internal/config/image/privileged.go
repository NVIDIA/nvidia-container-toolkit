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

package image

import (
	"github.com/opencontainers/runtime-spec/specs-go"
)

const (
	capSysAdmin = "CAP_SYS_ADMIN"
)

type CapabilitiesGetter interface {
	GetCapabilities() []string
}

type OCISpec specs.Spec

type OCISpecCapabilities specs.LinuxCapabilities

// IsPrivileged returns true if the container is a privileged container.
func IsPrivileged(s CapabilitiesGetter) bool {
	if s == nil {
		return false
	}
	for _, c := range s.GetCapabilities() {
		if c == capSysAdmin {
			return true
		}
	}

	return false
}

func (s OCISpec) GetCapabilities() []string {
	if s.Process == nil || s.Process.Capabilities == nil {
		return nil
	}
	return (*OCISpecCapabilities)(s.Process.Capabilities).GetCapabilities()
}

func (c OCISpecCapabilities) GetCapabilities() []string {
	// We only make sure that the bounding capability set has
	// CAP_SYS_ADMIN. This allows us to make sure that the container was
	// actually started as '--privileged', but also allow non-root users to
	// access the privileged NVIDIA capabilities.
	return c.Bounding
}
