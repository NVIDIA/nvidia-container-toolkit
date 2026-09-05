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

package modifier

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

// newRlimitModifier constructs a modifier that applies the POSIX rlimits
// configured for the NVIDIA Container Runtime to the OCI runtime
// specification. This allows resource limits such as RLIMIT_MEMLOCK -- which
// RDMA workloads require far in excess of typical daemon defaults -- to be
// set for the containers handled by this runtime without changing the limits
// of every container on the host.
//
// The modifier is intentionally not gated on requested devices: workloads
// that need the configured limits (e.g. RDMA-only containers) do not
// necessarily request GPUs.
func (f *Factory) newRlimitModifier() (oci.SpecModifier, error) {
	rlimits, err := parseRlimits(f.cfg.NVIDIAContainerRuntimeConfig.Rlimits)
	if err != nil {
		return nil, fmt.Errorf("failed to parse rlimits: %w", err)
	}
	if len(rlimits) == 0 {
		return nil, nil
	}
	return &rlimitModifier{
		logger:  f.logger,
		rlimits: rlimits,
	}, nil
}

type rlimitModifier struct {
	logger  logger.Interface
	rlimits []specs.POSIXRlimit
}

var _ oci.SpecModifier = (*rlimitModifier)(nil)

// Modify applies the configured rlimits to the OCI spec. An entry replaces an
// existing rlimit of the same type; other entries in the spec are preserved.
func (m *rlimitModifier) Modify(spec *specs.Spec) error {
	if spec == nil {
		return fmt.Errorf("cannot modify nil spec")
	}
	if spec.Process == nil {
		spec.Process = &specs.Process{}
	}
	for _, rlimit := range m.rlimits {
		m.logger.Debugf("Setting rlimit %v to %v:%v", rlimit.Type, rlimit.Soft, rlimit.Hard)
		spec.Process.Rlimits = upsertRlimit(spec.Process.Rlimits, rlimit)
	}
	return nil
}

// upsertRlimit replaces the rlimit of the same type in the supplied list, or
// appends it if no such entry exists.
func upsertRlimit(rlimits []specs.POSIXRlimit, rlimit specs.POSIXRlimit) []specs.POSIXRlimit {
	for i, existing := range rlimits {
		if existing.Type == rlimit.Type {
			rlimits[i] = rlimit
			return rlimits
		}
	}
	return append(rlimits, rlimit)
}

// parseRlimits converts configured NAME=SOFT[:HARD] entries into OCI POSIX
// rlimits. Duplicate types are rejected to surface configuration mistakes
// instead of silently applying one of the values.
func parseRlimits(entries []string) ([]specs.POSIXRlimit, error) {
	var rlimits []specs.POSIXRlimit
	seen := make(map[string]bool)
	for _, entry := range entries {
		rlimit, err := parseRlimit(entry)
		if err != nil {
			return nil, err
		}
		if seen[rlimit.Type] {
			return nil, fmt.Errorf("duplicate rlimit type %v", rlimit.Type)
		}
		seen[rlimit.Type] = true
		rlimits = append(rlimits, rlimit)
	}
	return rlimits, nil
}

// parseRlimit parses a single NAME=SOFT[:HARD] entry. The name is
// case-insensitive and the RLIMIT_ prefix is optional; values are
// non-negative integers or "unlimited"/"infinity". If HARD is omitted, it is
// set to SOFT.
func parseRlimit(entry string) (specs.POSIXRlimit, error) {
	name, values, ok := strings.Cut(entry, "=")
	name = strings.TrimSpace(name)
	if !ok || name == "" {
		return specs.POSIXRlimit{}, fmt.Errorf("invalid rlimit %q: expected NAME=SOFT[:HARD]", entry)
	}

	rlimitType := strings.ToUpper(name)
	if !strings.HasPrefix(rlimitType, "RLIMIT_") {
		rlimitType = "RLIMIT_" + rlimitType
	}

	softValue, hardValue, ok := strings.Cut(values, ":")
	if !ok {
		hardValue = softValue
	}
	soft, err := parseRlimitValue(softValue)
	if err != nil {
		return specs.POSIXRlimit{}, fmt.Errorf("invalid rlimit %q: %w", entry, err)
	}
	hard, err := parseRlimitValue(hardValue)
	if err != nil {
		return specs.POSIXRlimit{}, fmt.Errorf("invalid rlimit %q: %w", entry, err)
	}
	if soft > hard {
		return specs.POSIXRlimit{}, fmt.Errorf("invalid rlimit %q: soft limit %v exceeds hard limit %v", entry, softValue, hardValue)
	}

	return specs.POSIXRlimit{
		Type: rlimitType,
		Soft: soft,
		Hard: hard,
	}, nil
}

// parseRlimitValue parses a single rlimit value: a non-negative integer or
// "unlimited"/"infinity", which map to RLIM_INFINITY.
func parseRlimitValue(value string) (uint64, error) {
	value = strings.TrimSpace(value)
	switch strings.ToLower(value) {
	case "unlimited", "infinity":
		return math.MaxUint64, nil
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid value %q: expected a non-negative integer, \"unlimited\", or \"infinity\"", value)
	}
	return parsed, nil
}
