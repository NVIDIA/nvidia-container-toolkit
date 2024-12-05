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

package dgpu

import (
	"errors"
	"fmt"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
)

// NewDriverDiscoverer creates a discoverer for the libraries and binaries associated with a driver installation.
func NewDriverDiscoverer(opts ...Option) (discover.Discover, error) {
	o := new(opts...)

	if o.version == "" {
		return nil, fmt.Errorf("a version must be specified")
	}

	var discoverers []discover.Discover
	var errs error

	nvsandboxutilsDiscoverer, err := o.newNvsandboxutilsDriverDiscoverer()
	if err != nil {
		// TODO: Log a warning
		errs = errors.Join(errs, err)
	} else if nvsandboxutilsDiscoverer != nil {
		discoverers = append(discoverers, nvsandboxutilsDiscoverer)
	}

	nvmlDiscoverer, err := o.newNvmlDriverDiscoverer()
	if err != nil {
		// TODO: Log a warning
		errs = errors.Join(errs, err)
	} else if nvmlDiscoverer != nil {
		discoverers = append(discoverers, nvmlDiscoverer)
	}

	if len(discoverers) == 0 {
		return nil, errs
	}

	cached := discover.WithCache(
		discover.FirstValid(
			discoverers...,
		),
	)
	updateLDCache, _ := discover.NewLDCacheUpdateHook(o.logger, cached, o.nvidiaCDIHookPath, o.ldconfigPath)

	ipcs, err := discover.NewIPCDiscoverer(o.logger, o.driver.Root)
	if err != nil {
		return nil, fmt.Errorf("failed to create discoverer for IPC sockets: %v", err)
	}

	return discover.Merge(
		cached,
		updateLDCache,
		ipcs,
	), nil
}
