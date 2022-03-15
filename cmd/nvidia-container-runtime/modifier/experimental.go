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

package modifier

import (
	"fmt"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover/csv"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/edits"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
)

// experiemental represents the modifications required by the experimental runtime
type experimental struct {
	logger     *logrus.Logger
	discoverer discover.Discover
}

const (
	visibleDevicesEnvvar = "NVIDIA_VISIBLE_DEVICES"
	visibleDevicesVoid   = "void"

	nvidiaRequireJetpackEnvvar = "NVIDIA_REQUIRE_JETPACK"
)

// NewExperimentalModifier creates a modifier that applies the experimental
// modications to an OCI spec if required by the runtime wrapper.
func NewExperimentalModifier(logger *logrus.Logger, cfg *config.Config, ociSpec oci.Spec) (oci.SpecModifier, error) {
	if err := ociSpec.Load(); err != nil {
		return nil, fmt.Errorf("failed to load OCI spec: %v", err)
	}

	// In experimental mode, we check whether a modification is required at all and return the lowlevelRuntime directly
	// if no modification is required.
	visibleDevices, exists := ociSpec.LookupEnv(visibleDevicesEnvvar)
	if !exists || visibleDevices == "" || visibleDevices == visibleDevicesVoid {
		logger.Infof("No modification required: %v=%v (exists=%v)", visibleDevicesEnvvar, visibleDevices, exists)
		return nil, nil
	}
	logger.Infof("Constructing modifier from config: %+v", cfg)

	root := cfg.NVIDIAContainerCLIConfig.Root

	var d discover.Discover
	switch cfg.NVIDIAContainerRuntimeConfig.DiscoverMode {
	case "legacy":
		legacyDiscoverer, err := discover.NewLegacyDiscoverer(logger, root)
		if err != nil {
			return nil, fmt.Errorf("failed to create legacy discoverer: %v", err)
		}
		d = legacyDiscoverer
	case "csv":
		csvFiles, err := csv.GetFileList(csv.DefaultRoot)
		if err != nil {
			return nil, fmt.Errorf("failed to get list of CSV files: %v", err)
		}

		nvidiaRequireJetpack, _ := ociSpec.LookupEnv(nvidiaRequireJetpackEnvvar)
		if nvidiaRequireJetpack != "csv-mounts=all" {
			csvFiles = csv.BaseFilesOnly(csvFiles)
		}

		csvDiscoverer, err := discover.NewFromCSVFiles(logger, csvFiles, root)
		if err != nil {
			return nil, fmt.Errorf("failed to create CSV discoverer: %v", err)
		}
		d = csvDiscoverer
	default:
		return nil, fmt.Errorf("invalid discover mode: %v", cfg.NVIDIAContainerRuntimeConfig.DiscoverMode)
	}

	return newExperimentalModifierFromDiscoverer(logger, d)
}

// newExperimentalModifierFromDiscoverer created a modifier that aplies the discovered
// modifications to an OCI spec if require by the runtime wrapper.
func newExperimentalModifierFromDiscoverer(logger *logrus.Logger, d discover.Discover) (oci.SpecModifier, error) {
	m := experimental{
		logger:     logger,
		discoverer: d,
	}
	return &m, nil
}

// Modify applies the required modifications to the incomming OCI spec. These modifications
// are applied in-place.
func (m experimental) Modify(spec *specs.Spec) error {
	err := nvidiaContainerRuntimeHookRemover{m.logger}.Modify(spec)
	if err != nil {
		return fmt.Errorf("failed to remove existing hooks: %v", err)
	}

	specEdits, err := edits.NewSpecEdits(m.logger, m.discoverer)
	if err != nil {
		return fmt.Errorf("failed to get required container edits: %v", err)
	}

	return specEdits.Modify(spec)
}
