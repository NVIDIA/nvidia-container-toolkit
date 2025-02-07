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
	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/image"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/cuda"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/modifier/cdi"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/platform-support/tegra/csv"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/requirements"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi"
)

// NewCSVModifier creates a modifier that applies modications to an OCI spec if required by the runtime wrapper.
// The modifications are defined by CSV MountSpecs.
func NewCSVModifier(logger logger.Interface, cfg *config.Config, container image.CUDA) (oci.SpecModifier, error) {
	if devices := container.VisibleDevicesFromEnvVar(); len(devices) == 0 {
		logger.Infof("No modification required; no devices requested")
		return nil, nil
	}
	logger.Infof("Constructing modifier from config: %+v", *cfg)

	if err := checkRequirements(logger, container); err != nil {
		return nil, fmt.Errorf("requirements not met: %v", err)
	}

	csvFiles, err := csv.GetFileList(cfg.NVIDIAContainerRuntimeConfig.Modes.CSV.MountSpecPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get list of CSV files: %v", err)
	}

	if container.Getenv(image.EnvVarNvidiaRequireJetpack) != "csv-mounts=all" {
		csvFiles = csv.BaseFilesOnly(csvFiles)
	}

	cdilib, err := nvcdi.New(
		nvcdi.WithLogger(logger),
		nvcdi.WithDriverRoot(cfg.NVIDIAContainerCLIConfig.Root),
		nvcdi.WithNVIDIACDIHookPath(cfg.NVIDIACTKConfig.Path),
		nvcdi.WithMode(nvcdi.ModeCSV),
		nvcdi.WithCSVFiles(csvFiles),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to construct CDI library: %v", err)
	}

	spec, err := cdilib.GetSpec()
	if err != nil {
		return nil, fmt.Errorf("failed to get CDI spec: %v", err)
	}

	return cdi.New(
		cdi.WithLogger(logger),
		cdi.WithSpec(spec.Raw()),
	)
}

func checkRequirements(logger logger.Interface, image image.CUDA) error {
	if image.HasDisableRequire() {
		// TODO: We could print the real value here instead
		logger.Debugf("NVIDIA_DISABLE_REQUIRE=%v; skipping requirement checks", true)
		return nil
	}

	imageRequirements, err := image.GetRequirements()
	if err != nil {
		//  TODO: Should we treat this as a failure, or just issue a warning?
		return fmt.Errorf("failed to get image requirements: %v", err)
	}

	r := requirements.New(logger, imageRequirements)

	cudaVersion, err := cuda.Version()
	if err != nil {
		logger.Warningf("Failed to get CUDA version: %v", err)
	} else {
		r.AddVersionProperty(requirements.CUDA, cudaVersion)
	}

	compteCapability, err := cuda.ComputeCapability(0)
	if err != nil {
		logger.Warningf("Failed to get CUDA Compute Capability: %v", err)
	} else {
		r.AddVersionProperty(requirements.ARCH, compteCapability)
	}

	return r.Assert()
}
