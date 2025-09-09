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

// AddRuntime adds a runtime to the containerd config
func (c *Config) AddRuntime(name string, path string, setAsDefault bool) error {
	if c == nil || c.Tree == nil {
		return fmt.Errorf("config is nil")
	}
	defaultRuntimeOptions := c.GetDefaultRuntimeOptions()
	return c.AddRuntimeWithOptions(name, path, setAsDefault, defaultRuntimeOptions)
}

func (c *Config) GetDefaultRuntimeOptions() interface{} {
	runtimeNamesForConfig := engine.GetLowLevelRuntimes(c)
	for _, r := range runtimeNamesForConfig {
		options := c.GetSubtreeByPath([]string{"plugins", c.CRIRuntimePluginName, "containerd", "runtimes", r})
		if options != nil {
			c.Logger.Debugf("Using options from runtime %v: %v", r, options)
			return options.Copy()
		}
	}
	c.Logger.Warningf("Could not infer options from runtimes %v", runtimeNamesForConfig)
	options, _ := toml.TreeFromMap(map[string]interface{}{
		"runtime_type":                    c.RuntimeType,
		"runtime_root":                    "",
		"runtime_engine":                  "",
		"privileged_without_host_devices": false,
	})
	return options
}

func (c *Config) AddRuntimeWithOptions(name string, path string, setAsDefault bool, options interface{}) error {
	config := *c.Tree

	config.Set("version", c.Version)

	if options != nil {
		config.SetPath([]string{"plugins", c.CRIRuntimePluginName, "containerd", "runtimes", name}, options)
	}
	if len(c.ContainerAnnotations) > 0 {
		annotations, err := c.getRuntimeAnnotations([]string{"plugins", c.CRIRuntimePluginName, "containerd", "runtimes", name, "container_annotations"})
		if err != nil {
			return err
		}
		annotations = append(c.ContainerAnnotations, annotations...)
		config.SetPath([]string{"plugins", c.CRIRuntimePluginName, "containerd", "runtimes", name, "container_annotations"}, annotations)
	}

	config.SetPath([]string{"plugins", c.CRIRuntimePluginName, "containerd", "runtimes", name, "options", "BinaryName"}, path)

	if setAsDefault {
		config.SetPath([]string{"plugins", c.CRIRuntimePluginName, "containerd", "default_runtime_name"}, name)
	} else {
		defaultRuntime, ok := config.GetPath([]string{"plugins", c.CRIRuntimePluginName, "containerd", "default_runtime_name"}).(string)
		if ok && defaultRuntime == name {
			config.DeletePath([]string{"plugins", c.CRIRuntimePluginName, "containerd", "default_runtime_name"})
		}
	}
	*c.Tree = config
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

// DefaultRuntime returns the default runtime for the cri-o config
func (c Config) DefaultRuntime() string {
	if runtime, ok := c.GetPath([]string{"plugins", c.CRIRuntimePluginName, "containerd", "default_runtime_name"}).(string); ok {
		return runtime
	}
	return ""
}

// EnableCDI sets the enable_cdi field in the Containerd config to true.
func (c *Config) EnableCDI() {
	config := *c.Tree
	config.SetPath([]string{"plugins", c.CRIRuntimePluginName, "enable_cdi"}, true)
	*c.Tree = config
}

// RemoveRuntime removes a runtime from the containerd config
func (c *Config) RemoveRuntime(name string) error {
	if c == nil || c.Tree == nil {
		return nil
	}

	// If using NVIDIA-specific configuration, handle file cleanup
	if c.nvidiaConfig != "" {
		// Check if all NVIDIA runtimes are being removed
		remainingNvidiaRuntimes := 0
		if runtimes := c.GetPath([]string{"plugins", c.CRIRuntimePluginName, "containerd", "runtimes"}); runtimes != nil {
			if runtimesTree, ok := runtimes.(*toml.Tree); ok {
				for _, runtimeName := range runtimesTree.Keys() {
					if c.isNvidiaRuntime(runtimeName) && runtimeName != name {
						remainingNvidiaRuntimes++
					}
				}
			}
		}

		// If this is the last NVIDIA runtime, remove the NVIDIA config file
		if remainingNvidiaRuntimes == 0 {
			if err := os.Remove(c.nvidiaConfig); err != nil && !os.IsNotExist(err) {
				c.Logger.Warningf("Failed to remove NVIDIA config file %s: %v", c.nvidiaConfig, err)
			} else {
				c.Logger.Infof("Removed NVIDIA config file: %s", c.nvidiaConfig)
			}
			// Don't modify the in-memory tree when using NVIDIA-specific configuration
			return nil
		}
	}

	config := *c.Tree

	config.DeletePath([]string{"plugins", c.CRIRuntimePluginName, "containerd", "runtimes", name})
	if runtime, ok := config.GetPath([]string{"plugins", c.CRIRuntimePluginName, "containerd", "default_runtime_name"}).(string); ok {
		if runtime == name {
			config.DeletePath([]string{"plugins", c.CRIRuntimePluginName, "containerd", "default_runtime_name"})
		}
	}

	runtimePath := []string{"plugins", c.CRIRuntimePluginName, "containerd", "runtimes", name}
	for i := 0; i < len(runtimePath); i++ {
		if runtimes, ok := config.GetPath(runtimePath[:len(runtimePath)-i]).(*toml.Tree); ok {
			if len(runtimes.Keys()) == 0 {
				config.DeletePath(runtimePath[:len(runtimePath)-i])
			}
		}
	}

	if len(config.Keys()) == 1 && config.Keys()[0] == "version" {
		config.Delete("version")
	}

	*c.Tree = config
	return nil
}

// Save writes the config to the specified path or NVIDIA-specific config file
func (c *Config) Save(path string) (int64, error) {
	if c.nvidiaConfig == "" {
		// Backward compatibility: save to main config
		return c.Tree.Save(path)
	}

	// Ensure directory for NVIDIA config file exists
	dir := filepath.Dir(c.nvidiaConfig)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create directory for NVIDIA config: %w", err)
	}

	// Save runtime configs to NVIDIA config file
	nvidiaConfig := c.extractRuntimeConfig()
	n, err := nvidiaConfig.Save(c.nvidiaConfig)
	if err != nil {
		return n, fmt.Errorf("failed to save NVIDIA config: %w", err)
	}

	// Update main config with imports directive
	if err := c.updateMainConfigImports(path); err != nil {
		// Try to clean up the NVIDIA config file on error
		os.Remove(c.nvidiaConfig)
		return n, fmt.Errorf("failed to update main config imports: %w", err)
	}

	c.Logger.Infof("Wrote NVIDIA runtime configuration to: %s", c.nvidiaConfig)
	return n, nil
}

// extractRuntimeConfig creates a new config tree with only runtime configurations
func (c *Config) extractRuntimeConfig() *toml.Tree {
	config, _ := toml.TreeFromMap(map[string]interface{}{
		"version": c.Version,
	})

	// Extract runtime configurations for NVIDIA runtimes
	if runtimes := c.GetPath([]string{"plugins", c.CRIRuntimePluginName, "containerd", "runtimes"}); runtimes != nil {
		if runtimesTree, ok := runtimes.(*toml.Tree); ok {
			nvidiaRuntimes, _ := toml.TreeFromMap(map[string]interface{}{})
			for _, name := range runtimesTree.Keys() {
				if c.isNvidiaRuntime(name) {
					if runtime := runtimesTree.Get(name); runtime != nil {
						nvidiaRuntimes.Set(name, runtime)
					}
				}
			}
			if len(nvidiaRuntimes.Keys()) > 0 {
				config.SetPath([]string{"plugins", c.CRIRuntimePluginName, "containerd", "runtimes"}, nvidiaRuntimes)
			}
		}
	}

	// Extract default runtime name if it's one of ours
	if defaultRuntime, ok := c.GetPath([]string{"plugins", c.CRIRuntimePluginName, "containerd", "default_runtime_name"}).(string); ok {
		if c.isNvidiaRuntime(defaultRuntime) {
			config.SetPath([]string{"plugins", c.CRIRuntimePluginName, "containerd", "default_runtime_name"}, defaultRuntime)
		}
	}

	// Extract CDI enablement
	if cdiEnabled, ok := c.GetPath([]string{"plugins", c.CRIRuntimePluginName, "enable_cdi"}).(bool); ok && cdiEnabled {
		config.SetPath([]string{"plugins", c.CRIRuntimePluginName, "enable_cdi"}, true)
	}

	return config
}

// updateMainConfigImports ensures the main config includes an imports directive
func (c *Config) updateMainConfigImports(path string) error {
	// Load the main config file
	mainConfig, err := toml.FromFile(path).Load()
	if err != nil {
		// If the file doesn't exist, create a minimal config with imports
		if os.IsNotExist(err) {
			mainConfig, _ = toml.TreeFromMap(map[string]interface{}{
				"version": c.Version,
			})
		} else {
			return fmt.Errorf("failed to load main config: %w", err)
		}
	}

	// Add imports directive if not present
	importPattern := c.nvidiaConfig
	imports := mainConfig.Get("imports")
	if imports == nil {
		mainConfig.Set("imports", []string{importPattern})
	} else if importsList, ok := imports.([]interface{}); ok {
		// Check if the import pattern already exists
		found := false
		for _, imp := range importsList {
			if impStr, ok := imp.(string); ok && impStr == importPattern {
				found = true
				break
			}
		}
		if !found {
			// Add our import pattern
			importsList = append(importsList, importPattern)
			mainConfig.Set("imports", importsList)
		}
	} else if importsStrList, ok := imports.([]string); ok {
		// Check if the import pattern already exists
		found := false
		for _, imp := range importsStrList {
			if imp == importPattern {
				found = true
				break
			}
		}
		if !found {
			// Add our import pattern
			importsStrList = append(importsStrList, importPattern)
			mainConfig.Set("imports", importsStrList)
		}
	} else {
		return fmt.Errorf("unexpected imports type: %T", imports)
	}

	// Save the updated main config
	_, err = mainConfig.Save(path)
	return err
}

// isNvidiaRuntime checks if the runtime name is an NVIDIA runtime
func (c *Config) isNvidiaRuntime(name string) bool {
	return name == "nvidia" || name == "nvidia-cdi" || name == "nvidia-legacy"
}
