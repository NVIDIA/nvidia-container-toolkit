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

type legacy struct {
	None
	logger *logrus.Logger
	lookup lookup.Locator
}

const (
	nvidiaContainerRuntimeHookExecutable = "nvidia-container-runtime-hook"
	hookDefaultFilePath                  = "/usr/bin/nvidia-container-runtime-hook"
)

var _ Discover = (*legacy)(nil)

// NewLegacyDiscoverer creates a discoverer for the legacy runtime
func NewLegacyDiscoverer(logger *logrus.Logger, root string) (Discover, error) {
	d := legacy{
		logger: logger,
		lookup: lookup.NewExecutableLocator(logger, root),
	}

	return &d, nil
}

// Hooks returns the "legacy" NVIDIA Container Runtime hook. This hook calls out
// to the nvidia-container-cli to make modifications to the container as defined
// in libnvidia-container.
func (d legacy) Hooks() ([]Hook, error) {
	var hooks []Hook

	hookPath := hookDefaultFilePath
	targets, err := d.lookup.Locate(nvidiaContainerRuntimeHookExecutable)
	if err != nil {
		d.logger.Warnf("Failed to locate %v: %v", nvidiaContainerRuntimeHookExecutable, err)
	} else if len(targets) == 0 {
		d.logger.Warnf("%v not found", nvidiaContainerRuntimeHookExecutable)
	} else {
		d.logger.Debugf("Found %v candidates: %v", nvidiaContainerRuntimeHookExecutable, targets)
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
