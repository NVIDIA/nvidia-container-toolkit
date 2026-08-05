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
	"fmt"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/root"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/nvcaps"
)

// NewMigConfigDiscoverer creates a discoverer for the MIG config capability device node.
func NewMigConfigDiscoverer(logger logger.Interface, driver *root.Driver) (Discover, error) {
	migCaps, err := nvcaps.NewMigCaps()
	if err != nil {
		return nil, fmt.Errorf("error getting MIG capability device paths: %w", err)
	}
	return newMigCapDiscoverer(logger, driver, migCaps, nvcaps.MigCap("config"))
}

// NewMigMonitorDiscoverer creates a discoverer for the MIG monitor capability device node.
func NewMigMonitorDiscoverer(logger logger.Interface, driver *root.Driver) (Discover, error) {
	migCaps, err := nvcaps.NewMigCaps()
	if err != nil {
		return nil, fmt.Errorf("error getting MIG capability device paths: %w", err)
	}
	return newMigCapDiscoverer(logger, driver, migCaps, nvcaps.MigCap("monitor"))
}

func newMigCapDiscoverer(logger logger.Interface, driver *root.Driver, migCaps nvcaps.MigCaps, cap nvcaps.MigCap) (Discover, error) {
	if migCaps == nil {
		return None{}, nil
	}

	capDevicePath, err := migCaps.GetCapDevicePath(cap)
	if err != nil {
		return nil, fmt.Errorf("failed to get MIG cap device path for %q: %w", cap, err)
	}

	return NewCharDeviceDiscoverer(
		logger,
		driver.DevRoot,
		[]string{capDevicePath},
	), nil
}
