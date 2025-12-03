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

package tegra

import (
	"github.com/NVIDIA/nvidia-container-toolkit/internal/platform-support/tegra/csv"
)

// MountSpecPathsByType define the per-type paths that define the entities
// (e.g. device nodes, directories, libraries, symlinks) that are required for
// gpu use on Tegra-based systems.
// These are typically populated from CSV files defined by the platform owner.
type MountSpecPathsByType map[csv.MountSpecType][]string
