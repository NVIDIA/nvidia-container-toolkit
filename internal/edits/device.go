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

package edits

import (
	"os"

	"tags.cncf.io/container-device-interface/pkg/cdi"
	"tags.cncf.io/container-device-interface/specs-go"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/devices"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
)

type device discover.Device

// toEdits converts a discovered device to CDI Container Edits.
func (d device) toEdits() (*cdi.ContainerEdits, error) {
	deviceNode, err := d.toSpec()
	if err != nil {
		return nil, err
	}

	e := cdi.ContainerEdits{
		ContainerEdits: &specs.ContainerEdits{
			DeviceNodes: []*specs.DeviceNode{deviceNode},
		},
	}
	return &e, nil
}

// toSpec converts a discovered Device to a CDI Spec Device. Note
// that missing info is filled in when edits are applied by querying the Device node.
func (d device) toSpec() (*specs.DeviceNode, error) {
	s := d.fromPathOrDefault()
	// The HostPath field was added in the v0.5.0 CDI specification.
	// The cdi package uses strict unmarshalling when loading specs from file causing failures for
	// unexpected fields.
	// Since the behaviour for HostPath == "" and HostPath == Path are equivalent, we clear HostPath
	// if it is equal to Path to ensure compatibility with the widest range of specs.
	if s.HostPath == d.Path {
		s.HostPath = ""
	}

	return s, nil
}

// fromPathOrDefault attempts to return the returns the information about the
// CDI device from the specified host path.
// If this fails a minimal device is returned so that this information can be
// queried by the container runtime such as containerd.
func (d device) fromPathOrDefault() *specs.DeviceNode {
	path := d.HostPath
	if path == "" {
		path = d.Path
	}
	dn, err := devices.DeviceFromPath(path, "rwm")
	if err != nil {
		return &specs.DeviceNode{
			HostPath: d.HostPath,
			Path:     d.Path,
		}
	}

	// We construct a CDI spec DeviceNode with the information retrieved.
	// Note that in addition to the fields that we specify here the following
	// are not taken from the extracted information:
	//
	// * dn.Rule.Allow: This has no equivalent in the CDI spec and is used for
	//					specifying cgroup rules in a container.
	// * dn.Rule.Type:  This could be translated to the DeviceNode.Type, but is
	//					not done. In the toolkit we only consider char devices
	//					(Type = 'c') and these are the default for device nodes
	//					in OCI compliant runtimes.
	// * dn.UID:		This is ignored so as to allow the UID of the container
	//					user to be applied when making modifications to the OCI
	//					runtime specification. Note that for most NVIDIA devices
	//					this would be 0 and as such the target UID pointer will
	//					remain `nil`.
	//					See: https://github.com/cncf-tags/container-device-interface/blob/e2632194760242fc74a30c3803107f9c1ba5718b/pkg/cdi/container-edits.go#L96-L100
	return &specs.DeviceNode{
		HostPath:    d.HostPath,
		Path:        d.Path,
		Major:       dn.Major,
		Minor:       dn.Minor,
		FileMode:    ptrIfNonZero(dn.FileMode),
		Permissions: string(dn.Permissions),
		GID:         ptrIfNonZero(dn.Gid),
	}
}

func ptrIfNonZero[T uint32 | os.FileMode](id T) *T {
	var zero T
	if id == zero {
		return nil
	}
	return &id
}
