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
	"sync"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	"github.com/sirupsen/logrus"
)

// mounts is a generic discoverer for Mounts. It is customized by specifying the
// required entities as a list and a Locator that is used to find the target mounts
// based on the entry in the list.
type mounts struct {
	None
	logger   *logrus.Logger
	lookup   lookup.Locator
	required []string
	sync.Mutex
	cache []Mount
}

var _ Discover = (*mounts)(nil)

func (d *mounts) Mounts() ([]Mount, error) {
	if d.lookup == nil {
		return nil, fmt.Errorf("no lookup defined")
	}

	if d.cache != nil {
		d.logger.Debugf("returning cached mounts")
		return d.cache, nil
	}

	d.Lock()
	defer d.Unlock()

	paths := make(map[string]bool)

	for _, candidate := range d.required {
		d.logger.Debugf("Locating %v", candidate)
		located, err := d.lookup.Locate(candidate)
		if err != nil {
			d.logger.Warnf("Could not locate %v: %v", candidate, err)
			continue
		}
		if len(located) == 0 {
			d.logger.Warnf("Missing %v", candidate)
			continue
		}
		d.logger.Debugf("Located %v as %v", candidate, located)
		for _, p := range located {
			paths[p] = true
		}
	}

	var mounts []Mount
	for path := range paths {
		d.logger.Infof("Selecting %v", path)
		mount := Mount{
			Path: path,
		}
		mounts = append(mounts, mount)
	}

	d.cache = mounts

	return mounts, nil
}
