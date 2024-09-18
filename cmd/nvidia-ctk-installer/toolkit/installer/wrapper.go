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

package installer

import (
	"bytes"
	"debug/elf"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/toolkit/installer/wrappercore"
)

type wrapper struct {
	Source             string
	WrapperProgramPath string
	Config             wrappercore.WrapperConfig
}

func (w *wrapper) Install(destDir string) error {
	sourceFilename := filepath.Base(w.Source)
	sourceFilenameDotReal := sourceFilename + ".real"

	// Copy the source/original executable to the destination with a .real extension.
	mode, err := installFile(w.Source, filepath.Join(destDir, sourceFilenameDotReal))
	if err != nil {
		return err
	}

	// Create a wrapper program with the original's filename.
	content, err := w.render()
	if err != nil {
		return fmt.Errorf("failed to render wrapper: %w", err)
	}
	wrapperFile := filepath.Join(destDir, sourceFilename)
	return installContent(content, wrapperFile, mode|0111)
}

func (r *wrapper) render() (*bytes.Buffer, error) {
	wrapperProgram, err := os.Open(r.WrapperProgramPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer wrapperProgram.Close()
	elfBytes, err := io.ReadAll(wrapperProgram)
	if err != nil {
		return nil, fmt.Errorf("failed to read wrapper program: %v", err)
	}
	elfBuf := bytes.NewBuffer(elfBytes)
	elfFile, err := elf.NewFile(bytes.NewReader(elfBuf.Bytes()))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ELF file: %v", err)
	}
	configSection := elfFile.Section(wrappercore.InstallerWrapperConfigELFSectionName)
	if configSection == nil {
		return nil, fmt.Errorf("wrapper config section not found")
	}
	if err := wrappercore.WriteWrapperConfigSection(elfBuf.Bytes()[configSection.Offset:configSection.Offset+configSection.Size], &r.Config); err != nil {
		return nil, fmt.Errorf("failed to write wrapper config section: %v", err)
	}
	return elfBuf, nil
}
