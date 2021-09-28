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
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	log "github.com/sirupsen/logrus"
)

// NewBinaryMounts creates a discoverer for binaries using the specified root
func NewBinaryMounts(root string) Discover {
	return NewBinaryMountsWithLogger(log.StandardLogger(), root)
}

// NewBinaryMountsWithLogger creates a Mounts discoverer as with NewBinaryMounts
// with the specified logger
func NewBinaryMountsWithLogger(logger *log.Logger, root string) Discover {
	d := mounts{
		logger:   logger,
		lookup:   lookup.NewPathLocatorWithLogger(logger, root),
		required: requiredBinaries,
	}
	return &d
}

// requiredBinaries defines a set of binaries and their labels
var requiredBinaries = map[string][]string{
	"utility": {
		"nvidia-smi",          /* System management interface */
		"nvidia-debugdump",    /* GPU coredump utility */
		"nvidia-persistenced", /* Persistence mode utility */
	},
	"compute": {
		"nvidia-cuda-mps-control", /* Multi process service CLI */
		"nvidia-cuda-mps-server",  /* Multi process service server */
	},
}
