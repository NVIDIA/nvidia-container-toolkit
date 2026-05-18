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
	"context"
	"fmt"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

// Manager owns a set of running NRI plugin instances.
type Manager struct {
	logger  logger.Interface
	plugins []Runner
}

// NewManager creates a manager for starting and stopping NRI plugins.
func NewManager(log logger.Interface) *Manager {
	return &Manager{
		logger: log,
	}
}

// Start initializes each entry. Already-started plugins are stopped if a later
// plugin fails to start.
func (m *Manager) Start(ctx context.Context, entries []Entry) error {
	if err := ValidateEntries(entries); err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}

	m.logger.Infof("Starting %d NRI plugin(s)...", len(entries))
	for _, entry := range entries {
		m.logger.Infof("Starting NRI plugin %q (index=%02d, socket=%s)...",
			entry.Name, entry.Config.Index, entry.Config.Socket)

		if err := entry.PluginRunner.Start(ctx, entry.Config); err != nil {
			m.Stop()
			return fmt.Errorf("nri plugin %q (index=%02d): %w", entry.Name, entry.Config.Index, err)
		}
		m.plugins = append(m.plugins, entry.PluginRunner)
	}
	return nil
}

// Stop stops all running NRI plugins.
func (m *Manager) Stop() {
	for _, plugin := range m.plugins {
		plugin.Stop()
	}
	m.plugins = nil
}
