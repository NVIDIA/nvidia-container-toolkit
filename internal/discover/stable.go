/**
# Copyright (c) 2022, NVIDIA CORPORATION.  All rights reserved.
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

package discover

import (
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	"github.com/container-orchestrated-devices/container-device-interface/pkg/cdi"
	"github.com/sirupsen/logrus"
)

type stable struct {
	logger *logrus.Logger
	lookup lookup.Locator
}

const (
	nvidiaContainerRuntimeHookExecuable = "nvidia-container-runtime-hook"
	hookDefaultFilePath                 = "/usr/bin/nvidia-container-runtime-hook"
)

var _ Discover = (*stable)(nil)

// NewStableDiscoverer creates a discoverer for the stable runtime
func NewStableDiscoverer(logger *logrus.Logger, root string) (Discover, error) {
	d := stable{
		logger: logger,
		lookup: lookup.NewPathLocator(logger, root),
	}

	return &d, nil
}

// Hooks returns the "stable" NVIDIA Container Runtime hook
func (d stable) Hooks() ([]Hook, error) {
	var hooks []Hook

	hookPath := hookDefaultFilePath
	targets, err := d.lookup.Locate(nvidiaContainerRuntimeHookExecuable)
	if err != nil {
		d.logger.Warnf("Failed to locate %v: %v", nvidiaContainerRuntimeHookExecuable, err)
	} else if len(targets) == 0 {
		d.logger.Warnf("%v not found", nvidiaContainerRuntimeHookExecuable)
	} else {
		d.logger.Debugf("Found %v candidates: %v", nvidiaContainerRuntimeHookExecuable, targets)
		hookPath = targets[0]
	}
	d.logger.Debugf("Using NVIDIA Container Runtime Hook path %v", hookPath)

	args := []string{hookPath, "prestart"}
	legacyHook := Hook{
		Lifecycle: cdi.PrestartHook,
		Path:      hookPath,
		Args:      args,
	}
	hooks = append(hooks, legacyHook)
	return hooks, nil
}
