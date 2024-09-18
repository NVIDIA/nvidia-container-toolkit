/**
# SPDX-FileCopyrightText: Copyright (c) 2025 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
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

// Package wrappercore provides program wrapper functionality shared between the installer package
// and the wrapper program. The installer package imports many additional Go and CGO modules that
// should be excluded from the wrapper program binary to minimize its size.
package wrappercore

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
)

const InstallerWrapperConfigELFSectionName = ".nvctkiwc"
const InstallerWrapperFilename = "nvidia-ctk-installer-wrapper"

// WrapperConfig is the configuration for the installer wrapper program.
//
// The configuration for a wrapper is embedded in a dedicated ELF section with a short binary prefix
// using the `WriteWrapperConfigSection` function.
type WrapperConfig struct {
	Argv []string `json:"argv,omitempty"`
	// Envm is the environment variable map for the wrapped executable.
	//
	// The wrapper supports prepending and appending to path list variables (like PATH).
	//
	//   - '<': prepend (e.g. '<PATH')
	//   - '>': append (e.g. '>PATH')
	Envm                         map[string]string `json:"envm,omitempty"`
	RequiresKernelModule         bool              `json:"requiresKernelModule,omitempty"`
	DefaultRuntimeExecutablePath string            `json:"defaultRuntimeExecutablePath,omitempty"`
}

// WriteWrapperConfigSection writes a wrapper config into the provided section buffer.
func WriteWrapperConfigSection(sectionData []byte, config *WrapperConfig) error {
	configData, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal wrapper config: %v", err)
	}

	if int64(len(configData))+4 > int64(len(sectionData)) {
		return fmt.Errorf("data size %d (+ size header) exceeds section size %d", len(configData), len(sectionData))
	}
	var sizeBuf [4]byte
	binary.NativeEndian.PutUint32(sizeBuf[:], uint32(len(configData)))
	copy(sectionData, sizeBuf[:])
	copy(sectionData[4:], configData)
	return nil
}

// ReadWrapperConfigSection unmarshals the wrapper config from the provided section buffer.
func ReadWrapperConfigSection(sectionData []byte) (*WrapperConfig, error) {
	if len(sectionData) < 4 {
		return nil, fmt.Errorf("wrapper config section too small: %d bytes", len(sectionData))
	}
	dataSize := int64(binary.NativeEndian.Uint32(sectionData[:4]))
	if int64(len(sectionData)) < 4+dataSize {
		return nil, fmt.Errorf("wrapper config section data size mismatch: header says %d but only %d bytes available", dataSize, len(sectionData)-4)
	}
	configData := sectionData[4 : 4+dataSize]
	var config WrapperConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal wrapper config: %v", err)
	}
	return &config, nil
}
