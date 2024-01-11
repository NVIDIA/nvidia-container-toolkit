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

package config

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/pelletier/go-toml"
)

// Toml is a type for the TOML representation of a config.
type Toml toml.Tree

type options struct {
	configFile string
	required   bool
}

// Option is a functional option for loading TOML config files.
type Option func(*options)

// WithConfigFile sets the config file option.
func WithConfigFile(configFile string) Option {
	return func(o *options) {
		o.configFile = configFile
	}
}

// WithRequired sets the required option.
// If this is set to true, a failure to open the specified file is treated as an error
func WithRequired(required bool) Option {
	return func(o *options) {
		o.required = required
	}
}

// New creates a new toml tree based on the provided options
func New(opts ...Option) (*Toml, error) {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	return o.loadConfigToml()
}

func (o options) loadConfigToml() (*Toml, error) {
	filename := o.configFile
	if filename == "" {
		return defaultToml()
	}

	_, err := os.Stat(filename)
	if os.IsNotExist(err) && o.required {
		return nil, os.ErrNotExist
	}

	tomlFile, err := os.Open(filename)
	if os.IsNotExist(err) {
		return defaultToml()
	} else if err != nil {
		return nil, fmt.Errorf("failed to load specified config file: %w", err)
	}
	defer tomlFile.Close()

	return loadConfigTomlFrom(tomlFile)

}

func defaultToml() (*Toml, error) {
	cfg, err := GetDefault()
	if err != nil {
		return nil, err
	}
	contents, err := toml.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	return loadConfigTomlFrom(bytes.NewReader(contents))
}

func loadConfigTomlFrom(reader io.Reader) (*Toml, error) {
	tree, err := toml.LoadReader(reader)
	if err != nil {
		return nil, err
	}
	return (*Toml)(tree), nil
}

// Config returns the typed config associated with the toml tree.
func (t *Toml) Config() (*Config, error) {
	cfg, err := GetDefault()
	if err != nil {
		return nil, err
	}
	if t == nil {
		return cfg, nil
	}
	if err := t.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}
	return cfg, nil
}

// Unmarshal wraps the toml.Tree Unmarshal function.
func (t *Toml) Unmarshal(v interface{}) error {
	return (*toml.Tree)(t).Unmarshal(v)
}

// Save saves the config to the specified Writer.
func (t *Toml) Save(w io.Writer) (int64, error) {
	contents, err := t.contents()
	if err != nil {
		return 0, err
	}

	n, err := w.Write(contents)
	return int64(n), err
}

// contents returns the config TOML as a byte slice.
// Any required formatting is applied.
func (t Toml) contents() ([]byte, error) {
	commented := t.commentDefaults()

	buffer := bytes.NewBuffer(nil)

	enc := toml.NewEncoder(buffer).Indentation("")
	if err := enc.Encode((*toml.Tree)(commented)); err != nil {
		return nil, fmt.Errorf("invalid config: %v", err)
	}
	return t.format(buffer.Bytes())
}

// format fixes the comments for the config to ensure that they start in column
// 1 and are not followed by a space.
func (t Toml) format(contents []byte) ([]byte, error) {
	r := regexp.MustCompile(`(\n*)\s*?#\s*(\S.*)`)
	replaced := r.ReplaceAll(contents, []byte("$1#$2"))

	return replaced, nil
}

// Delete deletes the specified key from the TOML config.
func (t *Toml) Delete(key string) error {
	return (*toml.Tree)(t).Delete(key)
}

// Get returns the value for the specified key.
func (t *Toml) Get(key string) interface{} {
	return (*toml.Tree)(t).Get(key)
}

// Set sets the specified key to the specified value in the TOML config.
func (t *Toml) Set(key string, value interface{}) {
	(*toml.Tree)(t).Set(key, value)
}

// commentDefaults applies the required comments for default values to the Toml.
func (t *Toml) commentDefaults() *Toml {
	asToml := (*toml.Tree)(t)
	commentedDefaults := map[string]interface{}{
		"swarm-resource": "DOCKER_RESOURCE_GPU",
		"accept-nvidia-visible-devices-envvar-when-unprivileged": true,
		"accept-nvidia-visible-devices-as-volume-mounts":         false,
		"nvidia-container-cli.root":                              "/run/nvidia/driver",
		"nvidia-container-cli.path":                              "/usr/bin/nvidia-container-cli",
		"nvidia-container-cli.debug":                             "/var/log/nvidia-container-toolkit.log",
		"nvidia-container-cli.ldcache":                           "/etc/ld.so.cache",
		"nvidia-container-cli.no-cgroups":                        false,
		"nvidia-container-cli.user":                              "root:video",
		"nvidia-container-runtime.debug":                         "/var/log/nvidia-container-runtime.log",
	}
	for k, v := range commentedDefaults {
		set := asToml.Get(k)
		if !shouldComment(k, v, set) {
			continue
		}
		asToml.SetWithComment(k, "", true, v)
	}
	return (*Toml)(asToml)
}

func shouldComment(key string, defaultValue interface{}, setTo interface{}) bool {
	if key == "nvidia-container-cli.user" && defaultValue == setTo && isSuse() {
		return false
	}
	if key == "nvidia-container-runtime.debug" && setTo == "/dev/null" {
		return true
	}
	if setTo == nil || defaultValue == setTo || setTo == "" {
		return true
	}

	return false
}
