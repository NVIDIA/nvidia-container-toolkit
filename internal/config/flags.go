/**
# Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
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
	"reflect"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/platform-support/tegra/csv"

	cli "github.com/urfave/cli/v2"
)

// prt returns a reference to whatever type is passed into it
func ptr[T any](x T) *T {
	return &x
}

// ResolveCDIListConfig resolves the config struct for the CDI list subcommand in-place.
// It sets cfg.cdiSpecDirs using CLI > config > default priority.
// Accepts *list.config as the third argument.
func ResolveCDIListConfig(ctx *cli.Context, config *Config, cfg interface{}) {
	// Use switch statement for type assertion
	switch v := cfg.(type) {
	case *struct{ cdiSpecDirs cli.StringSlice }:
		var dirs []string
		switch {
		case ctx.IsSet("spec-dir"):
			dirs = ctx.StringSlice("spec-dir")
		case config != nil && len(config.NVIDIAContainerRuntimeConfig.Modes.CDI.SpecDirs) > 0:
			dirs = config.NVIDIAContainerRuntimeConfig.Modes.CDI.SpecDirs
		default:
			dirs = []string{"/etc/cdi", "/var/run/cdi"}
		}
		v.cdiSpecDirs = *cli.NewStringSlice(dirs...)
	case interface{ SetCDISpecDirs([]string) }:
		var dirs []string
		switch {
		case ctx.IsSet("spec-dir"):
			dirs = ctx.StringSlice("spec-dir")
		case config != nil && len(config.NVIDIAContainerRuntimeConfig.Modes.CDI.SpecDirs) > 0:
			dirs = config.NVIDIAContainerRuntimeConfig.Modes.CDI.SpecDirs
		default:
			dirs = []string{"/etc/cdi", "/var/run/cdi"}
		}
		v.SetCDISpecDirs(dirs)
	default:
		panic("ResolveCDIListConfig: unsupported config struct type")
	}
}

// ResolveCDIGenerateOptions resolves the options struct for the CDI generate subcommand in-place.
// It sets all fields using CLI > config > default priority.
// Uses reflection to support unexported fields from another package.
func ResolveCDIGenerateOptions(ctx *cli.Context, config *Config, opts interface{}) {
	// Define resolveStringSlice before use
	resolveStringSlice := func(flagName string, configVal []string, defaultVal []string) []string {
		if ctx != nil && ctx.IsSet(flagName) {
			return ctx.StringSlice(flagName)
		}
		if len(configVal) > 0 {
			return configVal
		}
		return defaultVal
	}

	// Always use csv.DefaultFileList() as the default for csv.file
	csvFileDefault := csv.DefaultFileList()
	csvFileConfig := []string{config.NVIDIAContainerRuntimeConfig.Modes.CSV.MountSpecPath}
	if config.NVIDIAContainerRuntimeConfig.Modes.CSV.MountSpecPath == "" {
		csvFileConfig = nil
	}

	// Use type assertion for setter methods first (like list.go)
	if setter, ok := opts.(interface{ SetCSVFiles([]string) }); ok {
		setter.SetCSVFiles(resolveStringSlice("csv.file", csvFileConfig, csvFileDefault))
	}
	if setter, ok := opts.(interface{ SetCSVIgnorePatterns([]string) }); ok {
		setter.SetCSVIgnorePatterns(resolveStringSlice("csv.ignore-pattern", nil, nil))
	}
	// ... existing reflection-based logic for other fields ...
	v := reflect.ValueOf(opts)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		panic("ResolveCDIGenerateOptions: opts must be a non-nil pointer to struct")
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		panic("ResolveCDIGenerateOptions: opts must be a pointer to struct")
	}

	setString := func(field, value string) {
		f := v.FieldByName(field)
		if f.IsValid() && f.CanSet() {
			f.SetString(value)
		}
	}
	setStringSlice := func(field string, value []string) {
		f := v.FieldByName(field)
		if f.IsValid() && f.CanSet() {
			f.Set(reflect.ValueOf(*cli.NewStringSlice(value...)))
		}
	}

	resolveString := func(flagName, configVal, defaultVal string) string {
		if ctx.IsSet(flagName) {
			return ctx.String(flagName)
		}
		if configVal != "" {
			return configVal
		}
		return defaultVal
	}

	setString("Format", resolveString("format", "", "yaml"))
	setString("Mode", resolveString("mode", config.NVIDIAContainerRuntimeConfig.Mode, "auto"))
	setString("NvidiaCDIHookPath", resolveString("nvidia-cdi-hook-path", config.NVIDIAContainerRuntimeHookConfig.Path, ""))
	setString("LdconfigPath", resolveString("ldconfig-path", string(config.NVIDIAContainerCLIConfig.Ldconfig), ""))
	setString("Vendor", resolveString("vendor", "nvidia.com", "nvidia.com"))
	setString("Class", resolveString("class", "gpu", "gpu"))
	setString("Output", resolveString("output", "", ""))
	setString("DriverRoot", resolveString("driver-root", "", ""))
	setString("DevRoot", resolveString("dev-root", "", ""))

	setStringSlice("DeviceNameStrategies", resolveStringSlice("device-name-strategy", nil, []string{"index", "uuid"}))
	setStringSlice("ConfigSearchPaths", resolveStringSlice("config-search-path", nil, nil))
	setStringSlice("LibrarySearchPaths", resolveStringSlice("library-search-path", nil, nil))
	setStringSlice("DisabledHooks", resolveStringSlice("disable-hook", nil, nil))

	// For reflection-based path, set csv.Files and csv.IgnorePatterns if present
	csvField := v.FieldByName("Csv")
	if csvField.IsValid() && csvField.Kind() == reflect.Struct {
		filesField := csvField.FieldByName("Files")
		if filesField.IsValid() && filesField.CanSet() {
			filesField.Set(reflect.ValueOf(*cli.NewStringSlice(resolveStringSlice("csv.file", csvFileConfig, csvFileDefault)...)))
		}
		ignorePatternsField := csvField.FieldByName("IgnorePatterns")
		if ignorePatternsField.IsValid() && ignorePatternsField.CanSet() {
			ignorePatternsField.Set(reflect.ValueOf(*cli.NewStringSlice(resolveStringSlice("csv.ignore-pattern", nil, nil)...)))
		}
	}
}
