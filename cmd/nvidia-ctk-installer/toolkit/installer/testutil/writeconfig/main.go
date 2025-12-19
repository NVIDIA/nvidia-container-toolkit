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

package main

import (
	"debug/elf"
	"encoding/json"
	"fmt"
	"os"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/toolkit/installer/wrappercore"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <wrapper program> <wrapper config>\n", os.Args[0])
		os.Exit(1)
	}

	var config wrappercore.WrapperConfig
	if err := json.Unmarshal([]byte(os.Args[2]), &config); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse wrapper config: %v\n", err)
		os.Exit(1)
	}

	file, err := os.OpenFile(os.Args[1], os.O_RDWR, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open wrapper program: %v\n", err)
		os.Exit(1)
	}
	elfFile, err := elf.NewFile(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse ELF file: %v\n", err)
		os.Exit(1)
	}
	configSection := elfFile.Section(wrappercore.InstallerWrapperConfigELFSectionName)
	if configSection == nil {
		fmt.Fprintf(os.Stderr, "Wrapper config section %q not found\n", wrappercore.InstallerWrapperConfigELFSectionName)
		os.Exit(1)
	}
	sectionData := make([]byte, configSection.Size)
	if err := wrappercore.WriteWrapperConfigSection(sectionData, &config); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write wrapper config: %v\n", err)
		os.Exit(1)
	}
	if _, err := file.WriteAt(sectionData, int64(configSection.Offset)); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write section data to file: %v\n", err)
		os.Exit(1)
	}
}
