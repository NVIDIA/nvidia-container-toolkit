/**
# SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
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

package cudacompat

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"golang.org/x/mod/semver"
)

type compatElfHeader struct {
	Format      int
	CUDAVersion cudaVersion `json:"CUDA Version"`
	Driver      []int
	Device      []int
}

// Elf32_Nhdr defines the header information for an ELF note.
// See https://man7.org/linux/man-pages/man5/elf.5.html#:~:text=by%20the%20linker.-,Notes,-(Nhdr)%0A%20%20%20%20%20%20%20ELF
// for the definition of an elf note.
// TODO: When should a 64-bit header be used?
type elf32_Nhdr struct {
	NameSize uint32
	DescSize uint32
	DescType uint32
}

func (h elf32_Nhdr) sizeof() int {
	return 12
}

func GetCUDACompatElfHeader(libraryPath string) (*compatElfHeader, error) {
	lib, err := elf.Open(libraryPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load elf info for %q: %w", libraryPath, err)
	}
	defer func() {
		_ = lib.Close()
	}()

	s := getCUDAFwdCompatibilitySection(lib)
	if s == nil {
		return nil, nil
	}
	data, err := s.Data()
	if err != nil {
		return nil, err
	}

	note := elf32_Nhdr{}
	r := bytes.NewReader(data)
	if err := binary.Read(r, lib.ByteOrder, &note); err != nil {
		return nil, fmt.Errorf("failed to read data header: %w", err)
	}

	if note.NameSize == 0 || note.DescSize == 0 {
		return nil, nil
	}

	name := string(trim(data, note.sizeof(), alignUp(note.NameSize, s.Addralign)))
	if name != "NVIDIA Corporation" {
		return nil, nil
	}

	description := trim(data, note.sizeof()+alignUp(note.NameSize, s.Addralign), int(note.DescSize))
	h := &compatElfHeader{}
	if err := json.Unmarshal(description, h); err != nil {
		return nil, fmt.Errorf("could not unmarshal JSON data: %w", err)
	}

	return h, err
}

func alignUp[T uint32 | uint64, S uint64](size T, to S) int {
	return int((size + T(to) - 1) &^ (T(to) - 1))
}

func trim(data []byte, from int, len int) []byte {
	return bytes.Trim(data[from:from+len], "\x00")
}

func getCUDAFwdCompatibilitySection(lib *elf.File) *elf.Section {
	for _, s := range lib.Sections {
		if s.Type != elf.SHT_NOTE {
			continue
		}
		if s.Name != ".note.cuda.fwd_compatibility" {
			continue
		}
		return s
	}
	return nil
}

// UseCompat checks whether the CUDA compat libraries with the specified elf
// header should be used given the specified host versions.
// If the hostDriverVersion is specified and the ELF header includes a list of
// driver verions, this is checked, otherwise the CUDA version specified in the
// ELF section is checked.
func (h *compatElfHeader) UseCompat(hostDriverMajor int, hostCUDAVersion string) bool {
	if h == nil {
		return false
	}

	if hostDriverMajor != 0 && len(h.Driver) > 0 {
		return slices.Contains(h.Driver, hostDriverMajor)
	}

	return h.CUDAVersion.UseCompat(hostCUDAVersion)
}

type cudaVersion string

// UseCompat is true if the container CUDA version is strictly greater than the
// host CUDA version.
func (containerVersion cudaVersion) UseCompat(hostVersion string) bool {
	if containerVersion == "" || hostVersion == "" {
		return false
	}

	return semver.Compare(normalizeVersion(containerVersion), normalizeVersion(hostVersion)) > 0
}

func normalizeVersion[T string | cudaVersion](v T) string {
	return "v" + strings.TrimPrefix(string(v), "v")
}
