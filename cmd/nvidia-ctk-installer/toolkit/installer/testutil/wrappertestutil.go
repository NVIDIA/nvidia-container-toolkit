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

package testutil

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// WriteTestELF returns the path to a minimal ELF file with a "config section" for testing.
func WriteTestELF(t *testing.T, configSectionName string) string {
	t.Helper()
	elfBuf := CreateTestELF(t, configSectionName)
	elfPath := filepath.Join(t.TempDir(), "test-wrapper")
	f, err := os.Create(elfPath)
	require.NoError(t, err)
	defer f.Close()
	_, err = f.Write(elfBuf.Bytes())
	require.NoError(t, err)
	return elfPath
}

// CreateTestELF returns an in-memory minimal ELF file with a "config section" for testing.
func CreateTestELF(t *testing.T, configSectionName string) *bytes.Buffer {
	t.Helper()

	var buf bytes.Buffer

	// ELF header with 4 sections (null, .shstrtab, wrapper config section, alignment)
	ehdr := elf.Header64{
		Ident:     [16]byte{0x7f, 'E', 'L', 'F', 2, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		Type:      uint16(elf.ET_EXEC),
		Machine:   uint16(elf.EM_X86_64),
		Version:   1,
		Entry:     0,
		Phoff:     0,
		Shoff:     64, // Section headers start after ELF header
		Flags:     0,
		Ehsize:    64,
		Phentsize: 0,
		Phnum:     0,
		Shentsize: 64,
		Shnum:     4,
		Shstrndx:  1, // .shstrtab is section 1
	}
	require.NoError(t, binary.Write(&buf, binary.LittleEndian, &ehdr))

	shdrTableOffset := uint64(64)
	shdrTableSize := uint64(4 * 64)
	shstrtabOffset := shdrTableOffset + shdrTableSize
	configSectionOffset := shstrtabOffset + 64 // some space for string table

	// Build section string table
	shstrtab := []byte{0}
	shstrtab = append(shstrtab, ".shstrtab"...)
	shstrtab = append(shstrtab, 0)
	configSectionNameOffset := len(shstrtab)
	shstrtab = append(shstrtab, configSectionName...)
	shstrtab = append(shstrtab, 0)

	// Pad to section headers offset
	for buf.Len() < int(shdrTableOffset) {
		buf.WriteByte(0)
	}

	// Section 0: null
	nullShdr := elf.Section64{}
	require.NoError(t, binary.Write(&buf, binary.LittleEndian, &nullShdr))

	// Section 1: .shstrtab
	shstrtabShdr := elf.Section64{
		Name:      1, // offset in shstrtab
		Type:      uint32(elf.SHT_STRTAB),
		Flags:     0,
		Addr:      0,
		Off:       shstrtabOffset,
		Size:      uint64(len(shstrtab)),
		Link:      0,
		Info:      0,
		Addralign: 1,
		Entsize:   0,
	}
	require.NoError(t, binary.Write(&buf, binary.LittleEndian, &shstrtabShdr))

	// Section 2: wrapper config
	configShdr := elf.Section64{
		Name:      uint32(configSectionNameOffset),
		Type:      uint32(elf.SHT_PROGBITS),
		Flags:     uint64(elf.SHF_ALLOC),
		Addr:      0,
		Off:       configSectionOffset,
		Size:      4096,
		Link:      0,
		Info:      0,
		Addralign: 8,
		Entsize:   0,
	}
	require.NoError(t, binary.Write(&buf, binary.LittleEndian, &configShdr))

	// Section 3: alignment
	dummyShdr := elf.Section64{}
	require.NoError(t, binary.Write(&buf, binary.LittleEndian, &dummyShdr))

	// Pad to shstrtab offset and write string table
	for buf.Len() < int(shstrtabOffset) {
		buf.WriteByte(0)
	}
	buf.Write(shstrtab)

	// Pad to wrapper config section and zero fill
	for buf.Len() < int(configSectionOffset) {
		buf.WriteByte(0)
	}
	buf.Write(make([]byte, 4096))

	return &buf
}
