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

package containerd

import (
	"fmt"
	"path/filepath"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine"
)

// A ConfigWithDropIn represents a pair of containerd configs.
// The first is the top-level config and the second is an in-memory drop-in config
// that only contains modifications made to the config.
type ConfigWithDropIn struct {
	logger         logger.Interface
	topLevelConfig *topLevelConfig
	engine.Interface
}

var _ engine.Interface = (*ConfigWithDropIn)(nil)

// A topLevelConfig stores the original on-disk top-level config.
// The path to the config is also stored to allow it to be modified if required.
type topLevelConfig struct {
	logger                 logger.Interface
	path                   string
	containerToHostPathMap map[string]string
	config                 *Config
}

func NewConfigWithDropIn(logger logger.Interface, topLevelConfigPath string, containerToHostPathMap map[string]string, tlConfig *Config, dropInConfig engine.Interface) *ConfigWithDropIn {
	return &ConfigWithDropIn{
		logger: logger,
		topLevelConfig: &topLevelConfig{
			logger:                 logger,
			path:                   topLevelConfigPath,
			containerToHostPathMap: containerToHostPathMap,
			config:                 tlConfig,
		},
		Interface: dropInConfig,
	}
}

// Save the drop-in config to the specified path.
// The top-level config is optionally updated to include the required imports
// to allow the drop-in-file to be created.
func (c *ConfigWithDropIn) Save(dropInPath string) (int64, error) {
	if dropInPath == engine.SaveToSTDOUT {
		c.logger.Infof("Drop-in config:")
	}
	bytesWritten, err := c.Interface.Save(dropInPath)
	if err != nil {
		return 0, err
	}

	switch {
	case bytesWritten == 0:
		// If the drop-in config is empty, we try to simplify the config.
		c.topLevelConfig.simplify(dropInPath)
	case bytesWritten > 0:
		// If the drop-in config has contents, we need to ensure that the
		// drop-in path is included in the imports.
		c.topLevelConfig.ensureImports(dropInPath)
	}

	// TODO: Only do this if we've actually modified the config.
	if _, err := c.topLevelConfig.Save(dropInPath); err != nil {
		return 0, fmt.Errorf("failed to save top-level config: %w", err)
	}

	return bytesWritten, nil
}

// RemoveRuntime removes the runtime from both configs.
func (c *ConfigWithDropIn) RemoveRuntime(name string) error {
	if err := c.topLevelConfig.RemoveRuntime(name); err != nil {
		return err
	}
	return c.Interface.RemoveRuntime(name)
}

// UpdateDefaultRuntime updates the default runtime setting in the drop-in config.
// When action is 'set' the provided runtime name is set as the default.
// When action is 'unset' we make sure the provided runtime name is not
// the default.
func (c *ConfigWithDropIn) UpdateDefaultRuntime(name string, action string) error {
	return c.Interface.UpdateDefaultRuntime(name, action)
}

// flush saves the top-level config to its path.
// If the config is empty, the file will be deleted.
func (c *topLevelConfig) Save(dropInPath string) (int64, error) {
	saveToPath := c.path
	if dropInPath == engine.SaveToSTDOUT {
		saveToPath = engine.SaveToSTDOUT
		c.logger.Infof("Top-level config:")
	}
	return c.config.Save(saveToPath)
}

func (c *topLevelConfig) simplify(dropInFilename string) {
	c.removeImports(dropInFilename)
	c.removeVersion()
}

// removeImports removes the imports specified in the file if the only entry
// corresponds to the path for the drop-in-file and the only other field in the
// file is the version field.
func (c *topLevelConfig) removeImports(dropInFilename string) {
	if len(c.config.Keys()) != 2 {
		return
	}
	if c.config.Get("version") == nil || c.config.Get("imports") == nil {
		return
	}

	currentImports, _ := c.config.getStringArrayValue([]string{"imports"})
	if len(currentImports) != 1 {
		return
	}

	requiredImport := c.importPattern(dropInFilename)
	if currentImports[0] != requiredImport {
		return
	}
	c.config.Delete("imports")
}

func (c *topLevelConfig) importPattern(dropInFilename string) string {
	// TODO: If we make output to STDOUT a property of the config itself, then
	// we can actually generate the correct import statement.
	if dropInFilename == engine.SaveToSTDOUT {
		return "/etc/containerd/config.d/*.toml"
	}
	return c.asHostPath(filepath.Dir(dropInFilename)) + "/*.toml"
}

func (c *topLevelConfig) asHostPath(path string) string {
	if c.containerToHostPathMap == nil {
		return path
	}
	if hostPath, ok := c.containerToHostPathMap[path]; ok {
		return hostPath
	}
	return path
}

// removeVersion removes the version if it is the ONLY field in the file.
func (c *topLevelConfig) removeVersion() {
	if len(c.config.Keys()) > 1 {
		return
	}
	if c.config.Get("version") == nil {
		return
	}
	c.config.Delete("version")
}

func (c *topLevelConfig) getCurrentImports() []string {
	rawImports := c.config.Get("imports")
	if rawImports == nil {
		return nil
	}

	if importsStringSlice, ok := rawImports.([]string); ok {
		return importsStringSlice
	}

	importsAnySlice, ok := rawImports.([]any)
	if !ok {
		return nil
	}
	var importsStringSlice []string
	for _, importAny := range importsAnySlice {
		importString, ok := importAny.(string)
		if !ok {
			continue
		}
		importsStringSlice = append(importsStringSlice, importString)
	}

	return importsStringSlice
}

func (c *topLevelConfig) ensureImports(dropInFilename string) {
	config := c.config.Tree
	currentImports := c.getCurrentImports()

	requiredImport := c.importPattern(dropInFilename)
	for _, currentImport := range currentImports {
		// If the requiredImport is already present, then we need not update the config.
		if currentImport == requiredImport {
			return
		}
	}

	currentImports = append(currentImports, requiredImport)

	// If the config is empty we need to set the version too.
	if len(config.Keys()) == 0 {
		config.Set("version", c.config.Version)
	}
	config.Set("imports", currentImports)
}

// RemoveRuntime removes the specified runtime from the top-level config.
func (c *topLevelConfig) RemoveRuntime(name string) error {
	return c.config.RemoveRuntime(name)
}
