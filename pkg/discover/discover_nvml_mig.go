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

	"github.com/NVIDIA/nvidia-container-toolkit/internal/nvcaps"
)

// getMigCaps returns a mapping of MIG capability path to device nodes
func getMigCaps(capDeviceMajor int) (map[ProcPath]DeviceNode, error) {
	migCaps, err := nvcaps.LoadMigMinors()
	if err != nil {
		return nil, fmt.Errorf("error loading MIG minors: %v", err)
	}
	return getMigCapsFromMigMinors(migCaps, capDeviceMajor), nil
}

func getMigCapsFromMigMinors(migCaps map[nvcaps.MigCap]nvcaps.MigMinor, capDeviceMajor int) map[ProcPath]DeviceNode {
	capsDevicePaths := make(map[ProcPath]DeviceNode)
	for cap, minor := range migCaps {
		capsDevicePaths[ProcPath(cap.ProcPath())] = DeviceNode{
			Path:  DevicePath(minor.DevicePath()),
			Major: capDeviceMajor,
			Minor: int(minor),
		}
	}
	return capsDevicePaths
}
