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

type noop struct{}

func (n noop) Resolve(mode string) string {
	return ""
}

type notEqualsResolver struct {
	logger basicLogger
	mode   string
}

func (n *notEqualsResolver) Resolve(mode string) string {
	if mode != string(n.mode) {
		n.logger.Infof("Using requested mode '%s'", mode)
		return mode
	}
	return ""
}

type firstOf []Resolver

func (resolvers firstOf) Resolve(mode string) string {
	for _, resolver := range resolvers {
		rmode := resolver.Resolve(mode)
		if rmode != "" {
			return rmode
		}
	}
	return ""
}

type systemMode struct {
	logger basicLogger
	Properties
}

func (s *systemMode) Resolve(string) string {
	isWSL, reason := s.HasDXCore()
	s.logger.Debugf("Is WSL-based system? %v: %v", isWSL, reason)

	isTegra, reason := s.IsTegraSystem()
	s.logger.Debugf("Is Tegra-based system? %v: %v", isTegra, reason)

	hasNVML, reason := s.HasNvml()
	s.logger.Debugf("Is NVML-based system? %v: %v", hasNVML, reason)

	usesNVGPUModule, reason := s.UsesNVGPUModule()
	s.logger.Debugf("Uses nvgpu kernel module? %v: %v", usesNVGPUModule, reason)

	if isWSL {
		return "wsl"
	}

	if (isTegra && !hasNVML) || usesNVGPUModule {
		return "csv"
	}
	return "nvml"
}
