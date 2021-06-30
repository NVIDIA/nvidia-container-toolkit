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

package discover

import (
	"fmt"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	log "github.com/sirupsen/logrus"
)

const (
	capabilityLabel = "capability"
	versionLabel    = "version"
)

// mounts is a generic discoverer for Mounts. It is customized by specifying the
// required entities as a key-value pair as well as a Locator that is used to
// identify the mounts that are to be included.
type mounts struct {
	None
	logger   *log.Logger
	lookup   lookup.Locator
	required map[string][]string
}

var _ Discover = (*mounts)(nil)

func (d mounts) Mounts() ([]Mount, error) {
	mounts, err := d.uniqueMounts()
	if err != nil {
		return nil, fmt.Errorf("error discovering mounts: %v", err)
	}

	return mounts.Slice(), nil
}

func (d mounts) uniqueMounts() (mountsByPath, error) {
	if d.lookup == nil {
		return nil, fmt.Errorf("no lookup defined")
	}

	mounts := make(mountsByPath)

	for id, keys := range d.required {
		for _, key := range keys {
			d.logger.Debugf("Locating %v [%v]", key, id)
			located, err := d.lookup.Locate(key)
			if err != nil {
				d.logger.Warnf("Could not locate %v [%v]: %v", key, id, err)
				continue
			}
			d.logger.Infof("Located %v [%v]: %v", key, id, located)
			for _, p := range located {
				// TODO: We need to add labels
				mount := newMount(p)
				mounts.Put(mount)
			}
		}
	}

	return mounts, nil
}

type mountsByPath map[string]Mount

func (m mountsByPath) Slice() []Mount {
	var mounts []Mount
	for _, mount := range m {
		mounts = append(mounts, mount)
	}

	return mounts
}

func (m *mountsByPath) Put(value Mount) {
	key := value.Path
	mount, exists := (*m)[key]
	if !exists {
		(*m)[key] = value
		return
	}

	for k, v := range value.Labels {
		mount.Labels[k] = v
	}
	(*m)[key] = mount
}

// NewMountForCapability creates a mount with the specified capability label
func NewMountForCapability(path string, capability string) Mount {
	return newMount(path, capabilityLabel, capability)
}

// NewMountForVersion creates a mount with the specified version label
func NewMountForVersion(path string, version string) Mount {
	return newMount(path, versionLabel, version)
}

func newMount(path string, labels ...string) Mount {
	l := make(map[string]string)

	for i := 0; i < len(labels)-1; i += 2 {
		l[labels[i]] = labels[i+1]
	}

	return Mount{
		Path:   path,
		Labels: l,
	}
}
