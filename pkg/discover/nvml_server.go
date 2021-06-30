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

	"github.com/NVIDIA/nvidia-container-toolkit/internal/nvml"
	log "github.com/sirupsen/logrus"
)

type nvmlServer struct {
	logger *log.Logger
	composite
}

var _ Discover = (*nvmlServer)(nil)

// NewNVMLServer constructs a discoverer for server systems using NVML to discover devices
func NewNVMLServer(root string) (Discover, error) {
	return NewNVMLServerWithLogger(log.StandardLogger(), root)
}

// NewNVMLServerWithLogger constructs a discoverer for server systems using NVML to discover devices with
// the specified logger
func NewNVMLServerWithLogger(logger *log.Logger, root string) (Discover, error) {
	return createNVMLServer(logger, nvml.New(), root)
}

func createNVMLServer(logger *log.Logger, nvml nvml.Interface, root string) (Discover, error) {
	d := nvmlServer{
		logger: logger,
	}

	devices, err := NewNVMLDiscoverWithLogger(logger, nvml)
	if err != nil {
		return nil, fmt.Errorf("error constructing NVML device discoverer: %v", err)
	}

	libraries, err := NewLibrariesWithLogger(logger, root)
	if err != nil {
		return nil, fmt.Errorf("error constructing library discoverer: %v", err)
	}

	d.add(
		// Device discovery
		devices,
		// Mounts discovery
		libraries,
		NewBinaryMountsWithLogger(logger, root),
		NewIPCMountsWithLogger(logger, root),
		// Hook discovery
		NewHooksWithLogger(logger),
	)

	return &d, nil
}
