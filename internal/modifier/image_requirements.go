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
	"strconv"
	"strings"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"golang.org/x/mod/semver"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/config/image"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/cuda"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/root"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/requirements"
)

// checkRequirements evaluates NVIDIA_REQUIRE_* constraints using the host
// CUDA driver API version from libcuda, the NVIDIA display driver version from
// the driver root (libcuda / libnvidia-ml soname), the compute capability of
// CUDA device 0, and (when requirements reference brand) the GPU product brand
// from NVML. It is used for CSV and CDI / JIT-CDI modes.
func checkRequirements(logger logger.Interface, image *image.CUDA, driver *root.Driver) error {
	if image == nil || image.HasDisableRequire() {
		logger.Debugf("NVIDIA_DISABLE_REQUIRE=%v; skipping requirement checks", true)
		return nil
	}

	imageRequirements, err := image.GetRequirements()
	if err != nil {
		return fmt.Errorf("failed to get image requirements: %v", err)
	}
	if len(imageRequirements) == 0 {
		return nil
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

	driverVersion, err := driver.Version()
	if err != nil {
		logger.Warningf("Failed to get NVIDIA driver version: %v", err)
	} else {
		normalized, normErr := normalizeDriverVersionForSemver(driverVersion)
		if normErr != nil {
			logger.Warningf("NVIDIA driver version %q is not semver-normalizable: %v", driverVersion, normErr)
		} else {
			r.AddVersionProperty(requirements.DRIVER, normalized)
		}
	}

	brand, err := getBrandFromNVML(driver)
	if err != nil {
		logger.Warningf("Failed to get GPU brand from NVML: %v", err)
	} else {
		r.AddStringProperty(requirements.BRAND, brand)
	}

	return r.Assert()
}

// normalizeDriverVersionForSemver converts a driver version taken from a
// libcuda / libnvidia-ml soname suffix into a form accepted by
// golang.org/x/mod/semver (no leading zeros in numeric segments)
func normalizeDriverVersionForSemver(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("empty driver version")
	}
	parts := strings.Split(raw, ".")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			return "", fmt.Errorf("empty version segment in %q", raw)
		}
		if strings.TrimLeft(p, "0123456789") != "" {
			return "", fmt.Errorf("non-numeric version segment %q in %q", p, raw)
		}
		n, err := strconv.ParseUint(p, 10, 64)
		if err != nil {
			return "", fmt.Errorf("invalid version segment %q in %q: %w", p, raw, err)
		}
		out = append(out, strconv.FormatUint(n, 10))
	}
	normalized := strings.Join(out, ".")
	if !semver.IsValid("v" + normalized) {
		return "", fmt.Errorf("normalized driver version %q is not valid semver", normalized)
	}
	return normalized, nil
}

// getBrandFromNVML returns a lowercase brand token for the first visible GPU
// (index 0), using NVML. When driver is non-nil, NVML is loaded from the
// versioned libnvidia-ml under the driver root when possible.
func getBrandFromNVML(driver *root.Driver) (string, error) {
	var lib nvml.Interface
	var opts []nvml.LibraryOption
	v, err := driver.Version()
	if err == nil && v != "" && v != "*.*" {
		paths, err := driver.Libraries().Locate("libnvidia-ml.so." + v)
		if err == nil && len(paths) > 0 {
			opts = append(opts, nvml.WithLibraryPath(paths[0]))
		}
	}

	lib = nvml.New(opts...)
	if ret := lib.Init(); ret != nvml.SUCCESS {
		return "", fmt.Errorf("nvml.Init: %s", lib.ErrorString(ret))
	}
	defer func() {
		_ = lib.Shutdown()
	}()

	device, ret := lib.DeviceGetHandleByIndex(0)
	if ret != nvml.SUCCESS {
		return "", fmt.Errorf("nvml.DeviceGetHandleByIndex(0): %s", lib.ErrorString(ret))
	}

	brandType, ret := lib.DeviceGetBrand(device)
	if ret != nvml.SUCCESS {
		return "", fmt.Errorf("nvml.DeviceGetBrand: %s", lib.ErrorString(ret))
	}
	brand, ok := brandTypeToRequirementString(brandType)
	if !ok {
		return "", fmt.Errorf("unknown NVML brand type %v", brandType)
	}
	return brand, nil
}

// brandTypeToRequirementString maps NVML brand enums to lowercase tokens
// consistent with typical NVIDIA_REQUIRE_* image constraints.
func brandTypeToRequirementString(b nvml.BrandType) (string, bool) {
	switch b {
	case nvml.BRAND_UNKNOWN:
		return "", false
	case nvml.BRAND_QUADRO:
		return "quadro", true
	case nvml.BRAND_TESLA:
		return "tesla", true
	case nvml.BRAND_NVS:
		return "nvs", true
	case nvml.BRAND_GRID:
		return "grid", true
	case nvml.BRAND_GEFORCE:
		return "geforce", true
	case nvml.BRAND_TITAN:
		return "titan", true
	case nvml.BRAND_NVIDIA_VAPPS:
		return "nvidiavapps", true
	case nvml.BRAND_NVIDIA_VPC:
		return "nvidiavpc", true
	case nvml.BRAND_NVIDIA_VCS:
		return "nvidiavcs", true
	case nvml.BRAND_NVIDIA_VWS:
		return "nvidiavws", true
	case nvml.BRAND_NVIDIA_CLOUD_GAMING:
		return "nvidiacloudgaming", true
	case nvml.BRAND_QUADRO_RTX:
		return "quadrortx", true
	case nvml.BRAND_NVIDIA_RTX:
		return "nvidiartx", true
	case nvml.BRAND_NVIDIA:
		return "nvidia", true
	case nvml.BRAND_GEFORCE_RTX:
		return "geforcertx", true
	case nvml.BRAND_TITAN_RTX:
		return "titanrtx", true
	default:
		return "", false
	}
}
