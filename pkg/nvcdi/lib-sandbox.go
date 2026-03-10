/**
# SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
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

package nvcdi

import (
	"fmt"

	"tags.cncf.io/container-device-interface/pkg/cdi"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
)

type sandboxlib struct {
	*nvcdilib
	emptyDeviceSpecGenerator
}

var _ deviceSpecGeneratorFactory = (*sandboxlib)(nil)

func (l *sandboxlib) DeviceSpecGenerators(...string) (DeviceSpecGenerator, error) {
	return l, nil
}

func (l *sandboxlib) GetCommonEdits() (*cdi.ContainerEdits, error) {
	graphicsMounts, err := discover.NewGraphicsMountsDiscoverer(l.logger, l.driver, l.hookCreator)
	if err != nil {
		l.logger.Warningf("failed to create discoverer for graphics mounts: %v", err)
	}

	driver, err := l.newDriverVersionDiscoverer()
	if err != nil {
		return nil, fmt.Errorf("failed to create driver library discoverer: %v", err)
	}

	edits, err := l.editsFactory.FromDiscoverer(discover.Merge(graphicsMounts, driver))
	if err != nil {
		return nil, fmt.Errorf("failed to create edits from discoverer: %v", err)
	}

	return edits, nil
}
