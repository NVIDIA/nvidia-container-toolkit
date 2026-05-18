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

package nri

import (
	"fmt"
)

const (
	maxPluginIndex = 99
)

// RegistrationConfig holds NRI registration settings shared by all plugins.
type RegistrationConfig struct {
	// Index is the plugin index registered with NRI (0-99).
	Index uint
	// Socket is the path to the NRI socket. When empty, the NRI default is used.
	Socket string
}

// ValidateEntries checks that each entry is usable and that plugin indices are unique.
func ValidateEntries(entries []Entry) error {
	if len(entries) == 0 {
		return nil
	}

	seenIndex := make(map[uint]string, len(entries))
	seenName := make(map[string]struct{}, len(entries))
	for i, entry := range entries {
		if len(entry.Name) == 0 {
			return fmt.Errorf("nri plugin %d: name must be specified", i)
		}
		if entry.PluginRunner == nil {
			return fmt.Errorf("nri plugin %q: implementation must be specified", entry.Name)
		}
		if entry.Config.Index > maxPluginIndex {
			return fmt.Errorf("nri plugin %q: index must be in the range [0,%d]", entry.Name, maxPluginIndex)
		}
		if other, ok := seenIndex[entry.Config.Index]; ok {
			return fmt.Errorf("nri plugin %q: duplicate plugin index %d (already used by %q)", entry.Name, entry.Config.Index, other)
		}
		seenIndex[entry.Config.Index] = entry.Name
		if _, ok := seenName[entry.Name]; ok {
			return fmt.Errorf("nri plugin %q: duplicate plugin name", entry.Name)
		}
		seenName[entry.Name] = struct{}{}
	}
	return nil
}

func (c RegistrationConfig) pluginIndex() string {
	return fmt.Sprintf("%02d", c.Index)
}
