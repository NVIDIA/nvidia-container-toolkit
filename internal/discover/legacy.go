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
	"path/filepath"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	"github.com/container-orchestrated-devices/container-device-interface/pkg/cdi"
	"github.com/sirupsen/logrus"
)

// NewLegacyDiscoverer creates a discoverer for the experimental runtime
func NewLegacyDiscoverer(logger *logrus.Logger, cfg *Config) (Discover, error) {
	d := legacy{
		logger: logger,
		lookup: lookup.NewExecutableLocator(logger, cfg.Root),
	}

	return &d, nil
}

type legacy struct {
	None
	logger *logrus.Logger
	lookup lookup.Locator
}

var _ Discover = (*legacy)(nil)

// Hooks returns the "legacy" NVIDIA Container Runtime hook. This hook calls out
// to the nvidia-container-cli to make modifications to the container as defined
// in libnvidia-container.
func (d legacy) Hooks() ([]Hook, error) {
	hookPath := filepath.Join(config.DefaultExecutableDir, config.NVIDIAContainerRuntimeHookExecutable)
	targets, err := d.lookup.Locate(config.NVIDIAContainerRuntimeHookExecutable)
	if err != nil {
		d.logger.Warnf("Failed to locate %v: %v", config.NVIDIAContainerRuntimeHookExecutable, err)
	} else if len(targets) == 0 {
		d.logger.Warnf("%v not found", config.NVIDIAContainerRuntimeHookExecutable)
	} else {
		d.logger.Debugf("Found %v candidates: %v", config.NVIDIAContainerRuntimeHookExecutable, targets)
		hookPath = targets[0]
	}
	d.logger.Debugf("Using NVIDIA Container Runtime Hook path %v", hookPath)

	args := []string{hookPath, "prestart"}
	h := Hook{
		Lifecycle: cdi.PrestartHook,
		Path:      hookPath,
		Args:      args,
	}

	return []Hook{h}, nil
}
