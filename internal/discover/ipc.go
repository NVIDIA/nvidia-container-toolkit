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

package discover

import (
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	"github.com/sirupsen/logrus"
)

type ipcMounts mounts

// NewIPCDiscoverer creats a discoverer for NVIDIA IPC sockets.
func NewIPCDiscoverer(logger *logrus.Logger, driverRoot string) (Discover, error) {
	d := newMounts(
		logger,
		lookup.NewFileLocator(
			lookup.WithLogger(logger),
			lookup.WithRoot(driverRoot),
		),
		driverRoot,
		[]string{
			"/var/run/nvidia-persistenced/socket",
			"/var/run/nvidia-fabricmanager/socket",
			"/tmp/nvidia-mps",
		},
	)

	return (*ipcMounts)(d), nil
}

func (d *ipcMounts) Mounts() ([]Mount, error) {
	mounts, err := (*mounts)(d).Mounts()
	if err != nil {
		return nil, err
	}

	return mounts, nil
}
