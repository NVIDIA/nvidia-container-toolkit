/**
# Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
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

package utils

import "github.com/NVIDIA/nvidia-container-toolkit/internal/logger"

// A SafeExecer is used to Exec an application from a memfd to prevent possible
// tampering.
type SafeExecer struct {
	logger logger.Interface
}

// NewSafeExecer creates a SafeExecer with the specified logger.
func NewSafeExecer(logger logger.Interface) SafeExecer {
	return SafeExecer{
		logger: logger,
	}
}
