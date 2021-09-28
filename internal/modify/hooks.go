/*
# Copyright (c) 2021, NVIDIA CORPORATION.  All rights reserved.
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
*/

package modify

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
	log "github.com/sirupsen/logrus"
)

// Hook is an alias to discover.Hook that allows for addition of a Modify method
type Hook struct {
	logger *log.Logger
	discover.Hook
	bundleDir string
}

var _ Modifier = (*Hook)(nil)

// Modify applies the modifications required by a Hook to the specified OCI specification
func (h Hook) Modify(spec oci.Spec) error {
	return spec.Modify(h.specModifier)
}

func (h Hook) specModifier(spec *specs.Spec) error {
	if spec.Hooks == nil {
		h.logger.Debugf("Initializing spec.Hooks")
		spec.Hooks = &specs.Hooks{}
	}

	// TODO: This is duplicated in the hook specification
	const rootPattern = "@Root.Path@"

	rootPath := spec.Root.Path
	if !filepath.IsAbs(rootPath) {
		rootPath = filepath.Join(h.bundleDir, rootPath)
	}

	var args []string
	for _, a := range h.Args {
		if strings.Contains(a, rootPattern) {
			args = append(args, strings.ReplaceAll(a, rootPattern, rootPath))
			continue
		}
		args = append(args, a)
	}

	specHook := specs.Hook{
		Path: h.Path,
		Args: args,
	}

	h.logger.Infof("Adding %v hook %+v", h.HookName, specHook)
	switch h.HookName {
	case "create-container":
		spec.Hooks.CreateContainer = append(spec.Hooks.CreateContainer, specHook)
	default:
		return fmt.Errorf("unexpected hook name: %v", h.HookName)
	}

	return nil
}
