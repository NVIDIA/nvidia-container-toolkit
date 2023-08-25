//go:build !linux

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

package devchar

import "golang.org/x/sys/unix"

func newDeviceNode(d string, stat unix.Stat_t) deviceNode {
	deviceNode := deviceNode{
		path:  d,
		major: unix.Major(uint64(stat.Rdev)),
		minor: unix.Minor(uint64(stat.Rdev)),
	}
	return deviceNode
}
