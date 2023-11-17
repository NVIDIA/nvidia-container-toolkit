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

package ldcache

import "github.com/NVIDIA/nvidia-container-toolkit/internal/logger"

type empty struct {
	logger logger.Interface
	path   string
}

var _ LDCache = (*empty)(nil)

// List always returns nil for an empty ldcache
func (e *empty) List() ([]string, []string) {
	return nil, nil
}

// Lookup logs a debug message and returns nil for an empty ldcache
func (e *empty) Lookup(prefixes ...string) ([]string, []string) {
	e.logger.Debugf("Calling Lookup(%v) on empty ldcache: %v", prefixes, e.path)
	return nil, nil
}
