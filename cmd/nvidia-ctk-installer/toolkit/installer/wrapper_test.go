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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/toolkit/installer/testutil"
	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/toolkit/installer/wrappercore"
)

func TestWrapperRender(t *testing.T) {
	elfPath := testutil.WriteTestELF(t, wrappercore.InstallerWrapperConfigELFSectionName)

	testCases := []struct {
		description string
		w           *wrapper
	}{
		{
			description: "executable is added",
			w: &wrapper{
				Config: wrappercore.WrapperConfig{
					DefaultRuntimeExecutablePath: "runc",
				},
			},
		},
		{
			description: "module check is added",
			w: &wrapper{
				Config: wrappercore.WrapperConfig{
					RequiresKernelModule:         true,
					DefaultRuntimeExecutablePath: "runc",
				},
			},
		},
		{
			description: "environment is added",
			w: &wrapper{
				Config: wrappercore.WrapperConfig{
					Envm: map[string]string{
						"PATH": "/foo/bar/baz",
					},
					DefaultRuntimeExecutablePath: "runc",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			tc.w.WrapperProgramPath = elfPath
			buffer, err := tc.w.render()
			require.NoError(t, err)
			elfFile, err := elf.NewFile(bytes.NewReader(buffer.Bytes()))
			require.NoError(t, err)
			section := elfFile.Section(wrappercore.InstallerWrapperConfigELFSectionName)
			require.NotNil(t, section)
			sectionData, err := section.Data()
			require.NoError(t, err)
			config, err := wrappercore.ReadWrapperConfigSection(sectionData)
			require.NoError(t, err)
			require.Equal(t, tc.w.Config, *config)
		})
	}
}

func TestWrapperRender_Errors(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("section not found", func(t *testing.T) {
		elfBuf := testutil.CreateTestELF(t, ".foobar")
		elfPath := filepath.Join(tmpDir, "test-wrapper")
		f, err := os.Create(elfPath)
		require.NoError(t, err)
		defer f.Close()
		_, err = f.Write(elfBuf.Bytes())
		require.NoError(t, err)
		w := &wrapper{
			WrapperProgramPath: elfPath,
		}
		_, err = w.render()
		require.Contains(t, err.Error(), "section not found")
	})
}

func TestWriteWrapperConfigSection(t *testing.T) {
	testCases := []struct {
		description string
		config      wrappercore.WrapperConfig
	}{
		{
			description: "empty config",
			config:      wrappercore.WrapperConfig{},
		},
		{
			description: "basic config",
			config: wrappercore.WrapperConfig{
				DefaultRuntimeExecutablePath: "runc",
			},
		},
		{
			description: "full config",
			config: wrappercore.WrapperConfig{
				Argv: []string{"--flag1", "--flag2"},
				Envm: map[string]string{
					"PATH":             "/usr/local/bin",
					"<LD_LIBRARY_PATH": "/opt/lib",
					">CUSTOM_VAR":      "/append/path",
				},
				RequiresKernelModule:         true,
				DefaultRuntimeExecutablePath: "/usr/bin/crun",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			elfBuf := bytes.NewBuffer(make([]byte, 4096))
			err := wrappercore.WriteWrapperConfigSection(elfBuf.Bytes(), &tc.config)
			require.NoError(t, err)
			readConfig, err := wrappercore.ReadWrapperConfigSection(elfBuf.Bytes())
			require.NoError(t, err)
			require.Equal(t, tc.config, *readConfig)
		})
	}
}

func TestWriteWrapperConfigSection_Errors(t *testing.T) {
	t.Run("data too large", func(t *testing.T) {
		largeConfig := wrappercore.WrapperConfig{
			Envm: make(map[string]string),
		}
		for i := 0; i < 500; i++ {
			largeConfig.Envm[fmt.Sprintf("VAR_%d", i)] = strings.Repeat("x", 100)
		}
		elfBuf := bytes.NewBuffer(make([]byte, 4096))
		err := wrappercore.WriteWrapperConfigSection(elfBuf.Bytes(), &largeConfig)
		require.Error(t, err)
		require.Contains(t, err.Error(), "exceeds section size")
	})
}

func TestReadWrapperConfigSection_Errors(t *testing.T) {
	testCases := []struct {
		description string
		sectionData []byte
		expectError string
	}{
		{
			description: "too small for size header",
			sectionData: []byte{0x01, 0x02},
			expectError: "too small",
		},
		{
			description: "size mismatch",
			sectionData: []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x01, 0x02},
			expectError: "size mismatch",
		},
		{
			description: "invalid JSON",
			sectionData: append([]byte{0x05, 0x00, 0x00, 0x00}, []byte("{not json")...),
			expectError: "unmarshal",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			_, err := wrappercore.ReadWrapperConfigSection(tc.sectionData)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.expectError)
		})
	}
}
