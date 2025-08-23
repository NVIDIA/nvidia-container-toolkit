/**
# Copyright (c) NVIDIA CORPORATION.  All rights reserved.
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
	"os"
	"path/filepath"

	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/toml"
)

// AddRuntime adds a runtime using drop-in configuration if enabled
func (c *Config) AddRuntime(name string, path string, setAsDefault bool) error {
	dropInConfig := c.NVConfig

	dropInConfig.Set("version", c.Version)

	runtimeNamesForConfig := engine.GetLowLevelRuntimes(c)
	for _, r := range runtimeNamesForConfig {
		options := c.GetSubtreeByPath([]string{"plugins", c.CRIRuntimePluginName, "containerd", "runtimes", r})
		if options == nil {
			continue
		}
		c.Logger.Debugf("using options from runtime %v: %v", r, options)
		dropInConfig.SetPath([]string{"plugins", c.CRIRuntimePluginName, "containerd", "runtimes", name}, options.Copy())
		break
	}

	dropInConfig.SetPath([]string{"plugins", c.CRIRuntimePluginName, "containerd", "runtimes", name, "runtime_type"}, c.RuntimeType)
	dropInConfig.SetPath([]string{"plugins", c.CRIRuntimePluginName, "containerd", "runtimes", name, "runtime_root"}, "")
	dropInConfig.SetPath([]string{"plugins", c.CRIRuntimePluginName, "containerd", "runtimes", name, "runtime_engine"}, "")
	dropInConfig.SetPath([]string{"plugins", c.CRIRuntimePluginName, "containerd", "runtimes", name, "privileged_without_host_devices"}, false)

	if len(c.ContainerAnnotations) > 0 {
		annotations, err := c.getRuntimeAnnotations([]string{"plugins", c.CRIRuntimePluginName, "containerd", "runtimes", name, "container_annotations"})
		if err != nil {
			return err
		}
		annotations = append(c.ContainerAnnotations, annotations...)
		dropInConfig.SetPath([]string{"plugins", c.CRIRuntimePluginName, "containerd", "runtimes", name, "container_annotations"}, annotations)
	}

	dropInConfig.SetPath([]string{"plugins", c.CRIRuntimePluginName, "containerd", "runtimes", name, "options", "BinaryName"}, path)

	if setAsDefault {
		dropInConfig.SetPath([]string{"plugins", c.CRIRuntimePluginName, "containerd", "default_runtime_name"}, name)
	}

	c.NVConfig = dropInConfig

	return nil
}

func (c *Config) getRuntimeAnnotations(path []string) ([]string, error) {
	if c == nil || c.Tree == nil {
		return nil, nil
	}

	config := *c.Tree
	if !config.HasPath(path) {
		return nil, nil
	}
	annotationsI, ok := config.GetPath(path).([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid annotations: %v", annotationsI)
	}

	var annotations []string
	for _, annotation := range annotationsI {
		a, ok := annotation.(string)
		if !ok {
			return nil, fmt.Errorf("invalid annotation: %v", annotation)
		}
		annotations = append(annotations, a)
	}

	return annotations, nil
}

// DefaultRuntime returns the default runtime for the containerd config
func (c Config) DefaultRuntime() string {
	if runtime, ok := c.GetPath([]string{"plugins", c.CRIRuntimePluginName, "containerd", "default_runtime_name"}).(string); ok {
		return runtime
	}
	return ""
}

// EnableCDI enables CDI in the drop-in configuration
func (c *Config) EnableCDI() {
	config := *c.NVConfig
	config.SetPath([]string{"plugins", c.CRIRuntimePluginName, "enable_cdi"}, true)
	*c.NVConfig = config
}

// RemoveRuntime removes the drop-in configuration
func (d *Config) RemoveRuntime(name string) error {
	// The drop-in file is expected to be named 99.nvidia.toml in the drop-in directory
	dropInFile := filepath.Join(d.dropInConfigPath, "99.nvidia.toml")

	// Check if the file exists
	if _, err := os.Stat(dropInFile); os.IsNotExist(err) {
		return nil
	}

	// Remove the drop-in file
	if err := os.Remove(dropInFile); err != nil {
		return fmt.Errorf("failed to remove drop-in file: %w", err)
	}

	return nil
}

func (c *Config) Save(path string) (int64, error) {
	if err := c.ensureMainConfigWithImports(); err != nil {
		return 0, fmt.Errorf("failed to ensure main config with imports: %w", err)
	}

	// Create drop-in directory if it doesn't exist
	if err := os.MkdirAll(c.dropInConfigPath, 0755); err != nil {
		return 0, fmt.Errorf("failed to create drop-in directory %s: %w", c.dropInConfigPath, err)
	}

	return c.NVConfig.Save(path)
}

// ensureMainConfigWithImports ensures the main config exists and has the imports directive
func (d *Config) ensureMainConfigWithImports() error {
	// Check if main config exists
	_, err := os.Stat(d.baseConfigPath)
	if os.IsNotExist(err) {
		// Create minimal config with imports
		return d.createMinimalMainConfig()
	} else if err != nil {
		return fmt.Errorf("failed to check main config: %w", err)
	}

	// Main config exists, ensure it has imports directive
	return d.ensureImportsDirective()
}

// createMinimalMainConfig creates a minimal containerd config with imports
func (d *Config) createMinimalMainConfig() error {
	// Create a new TOML tree with proper initialization
	// Include both the main config and drop-in directory in imports
	config, err := toml.TreeFromMap(map[string]interface{}{
		"version": int64(defaultConfigVersion),
		"imports": []string{
			d.baseConfigPath,
			filepath.Join(d.dropInConfigPath, "*.toml"),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create config tree: %w", err)
	}

	if _, err := config.Save(d.baseConfigPath); err != nil {
		return fmt.Errorf("failed to save minimal config: %w", err)
	}

	return nil
}

// ensureImportsDirective ensures the main config has the imports directive
func (c *Config) ensureImportsDirective() error {
	if c == nil || c.Tree == nil {
		return nil
	}
	baseConfig := *c.Tree

	// Check and update imports
	// We need both the main config path and the drop-in directory
	requiredImports := []string{
		c.baseConfigPath,
		filepath.Join(c.dropInConfigPath, "*.toml"),
	}

	importsI, ok := baseConfig.GetPath([]string{"imports"}).([]interface{})
	if !ok {
		// No imports, set to empty array
		importsI = []interface{}{}
	}

	imports := make([]string, 0, len(importsI))
	for _, imp := range importsI {
		imports = append(imports, imp.(string))
	}

	// Check which imports are missing
	needsUpdate := false
	for _, required := range requiredImports {
		found := false
		for _, imp := range imports {
			if imp == required {
				found = true
				break
			}
		}
		if !found {
			imports = append(imports, required)
			needsUpdate = true
		}
	}

	if needsUpdate {
		// Update imports
		baseConfig.SetPath([]string{"imports"}, imports)

		// Save the updated config
		if _, err := baseConfig.Save(c.baseConfigPath); err != nil {
			return fmt.Errorf("failed to save config with imports: %w", err)
		}
	}

	return nil
}

func (d *Config) String() string {
	return d.NVConfig.String()
}
