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

package nvcdi

import (
	"sync"

	"github.com/NVIDIA/go-nvlib/pkg/nvlib/info"
)

type Mode string

const (
	// ModeAuto configures the CDI spec generator to automatically detect the system configuration
	ModeAuto = Mode("auto")
	// ModeNvml configures the CDI spec generator to use the NVML library.
	ModeNvml = Mode("nvml")
	// ModeWsl configures the CDI spec generator to generate a WSL spec.
	ModeWsl = Mode("wsl")
	// ModeManagement configures the CDI spec generator to generate a management spec.
	ModeManagement = Mode("management")
	// ModeGds configures the CDI spec generator to generate a GDS spec.
	ModeGds = Mode("gds")
	// ModeMofed configures the CDI spec generator to generate a MOFED spec.
	ModeMofed = Mode("mofed")
	// ModeCSV configures the CDI spec generator to generate a spec based on the contents of CSV
	// mountspec files.
	ModeCSV = Mode("csv")
	// ModeImex configures the CDI spec generated to generate a spec for the available IMEX channels.
	ModeImex = Mode("imex")
)

type modeConstraint interface {
	string | Mode
}

type modes struct {
	lookup map[Mode]bool
	all    []Mode
}

var validModes modes
var validModesOnce sync.Once

func getModes() modes {
	validModesOnce.Do(func() {
		all := []Mode{
			ModeAuto,
			ModeNvml,
			ModeWsl,
			ModeManagement,
			ModeGds,
			ModeMofed,
			ModeCSV,
		}
		lookup := make(map[Mode]bool)

		for _, m := range all {
			lookup[m] = true
		}

		validModes = modes{
			lookup: lookup,
			all:    all,
		}
	},
	)
	return validModes
}

// AllModes returns the set of valid modes.
func AllModes[T modeConstraint]() []T {
	var output []T
	for _, m := range getModes().all {
		output = append(output, T(m))
	}
	return output
}

// IsValidMode checks whether a specified mode is valid.
func IsValidMode[T modeConstraint](mode T) bool {
	return getModes().lookup[Mode(mode)]
}

// resolveMode resolves the mode for CDI spec generation based on the current system.
func (l *nvcdilib) resolveMode() (rmode Mode) {
	if l.mode != ModeAuto {
		return l.mode
	}
	defer func() {
		l.logger.Infof("Auto-detected mode as '%v'", rmode)
	}()

	platform := l.infolib.ResolvePlatform()
	switch platform {
	case info.PlatformNVML:
		return ModeNvml
	case info.PlatformTegra:
		return ModeCSV
	case info.PlatformWSL:
		return ModeWsl
	}
	l.logger.Warningf("Unsupported platform detected: %v; assuming %v", platform, ModeNvml)
	return ModeNvml
}
