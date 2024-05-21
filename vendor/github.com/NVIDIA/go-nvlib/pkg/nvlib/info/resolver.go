/**
# Copyright 2024 NVIDIA CORPORATION
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

package info

// Platform represents a supported plaform.
type Platform string

const (
	PlatformAuto    = Platform("auto")
	PlatformNVML    = Platform("nvml")
	PlatformTegra   = Platform("tegra")
	PlatformWSL     = Platform("wsl")
	PlatformUnknown = Platform("unknown")
)

type platformResolver struct {
	logger            basicLogger
	platform          Platform
	propertyExtractor PropertyExtractor
}

func (p platformResolver) ResolvePlatform() Platform {
	if p.platform != PlatformAuto {
		p.logger.Infof("Using requested platform '%s'", p.platform)
		return p.platform
	}

	hasDXCore, reason := p.propertyExtractor.HasDXCore()
	p.logger.Debugf("Is WSL-based system? %v: %v", hasDXCore, reason)

	hasTegraFiles, reason := p.propertyExtractor.HasTegraFiles()
	p.logger.Debugf("Is Tegra-based system? %v: %v", hasTegraFiles, reason)

	hasNVML, reason := p.propertyExtractor.HasNvml()
	p.logger.Debugf("Is NVML-based system? %v: %v", hasNVML, reason)

	usesOnlyNVGPUModule, reason := p.propertyExtractor.UsesOnlyNVGPUModule()
	p.logger.Debugf("Uses nvgpu kernel module? %v: %v", usesOnlyNVGPUModule, reason)

	switch {
	case hasDXCore:
		return PlatformWSL
	case (hasTegraFiles && !hasNVML), usesOnlyNVGPUModule:
		return PlatformTegra
	case hasNVML:
		return PlatformNVML
	default:
		return PlatformUnknown
	}
}
