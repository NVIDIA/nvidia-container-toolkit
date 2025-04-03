//go:build linux
// +build linux

/**
# SPDX-FileCopyrightText: Copyright (c) 2025 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
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

package disabledevicenodemodification

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func createTmpFs(target string, size int) error {
	return unix.Mount("tmpfs", target, "tmpfs", 0, fmt.Sprintf("size=%d", size))
}

func bindMountReadonly(source string, target string) error {
	return unix.Mount(source, target, "", unix.MS_BIND|unix.MS_RDONLY|unix.MS_NOSYMFOLLOW, "")
}
