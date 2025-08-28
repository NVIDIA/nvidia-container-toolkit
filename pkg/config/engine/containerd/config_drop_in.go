package containerd

import (
	"path/filepath"

	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/engine"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/toml"
)

type WithTopLevel struct {
	engine.Interface
	topLevelConfig *topLevelConfig
}

type topLevelConfig struct {
	filename string
	version  int64
}

func (c *WithTopLevel) Save(path string) (int64, error) {
	if err := c.topLevelConfig.ensureImports(path); err != nil {
		return 0, err
	}

	return c.Interface.Save(path)
}

func (c *WithTopLevel) RemoveRuntime(name string) error {
	if err := c.topLevelConfig.RemoveRuntime(name); err != nil {
		return err
	}
	return c.Interface.RemoveRuntime(name)
}

func (c *topLevelConfig) RemoveRuntime(name string) error {
	// TODO: Implement the same logic that we currently have.
	return nil
}

func (c *topLevelConfig) ensureImports(dropInFilename string) error {
	// TODO: Load the config from c.filename
	config, err := toml.FromFile(c.filename).Load()
	if err != nil {
		return err
	}
	// TODO: Load the current imports from the config file.
	var currentImports []string

	requiredImport := filepath.Dir(dropInFilename) + "/*.toml"
	for _, currentImport := range currentImports {
		// If the requiredImport is already present, then we need not update the config.
		if currentImport == requiredImport {
			return nil
		}
	}

	currentImports = append(currentImports, requiredImport)

	// If the config is empty we need to set the version too.
	if len(config.Keys()) == 0 {
		config.Set("version", c.version)
	}
	config.Set("imports", currentImports)

	if _, err := config.Save(c.filename); err != nil {
		return err
	}
	return nil
}
