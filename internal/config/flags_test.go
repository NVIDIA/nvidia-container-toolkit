/*
*
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
*
*/
package config

import (
	"flag"
	"reflect"
	"testing"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/platform-support/tegra/csv"

	"github.com/stretchr/testify/require"
	cli "github.com/urfave/cli/v2"
)

type mockConfig struct {
	SpecDirs     []string
	Mode         string
	HookPath     string
	LdconfigPath string
	CSVSpecPath  string
}

func (m *mockConfig) toConfig() *Config {
	return &Config{
		NVIDIAContainerRuntimeConfig: RuntimeConfig{
			Mode: m.Mode,
			Modes: modesConfig{
				CDI: cdiModeConfig{
					SpecDirs: m.SpecDirs,
				},
				CSV: csvModeConfig{
					MountSpecPath: m.CSVSpecPath,
				},
			},
		},
		NVIDIAContainerRuntimeHookConfig: RuntimeHookConfig{
			Path: m.HookPath,
		},
		NVIDIAContainerCLIConfig: ContainerCLIConfig{
			Ldconfig: ldconfigPath(m.LdconfigPath),
		},
	}
}

func TestResolveCDIListConfig(t *testing.T) {
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		&cli.StringSliceFlag{
			Name: "spec-dir",
		},
	}
	set := func(args ...string) *cli.Context {
		set := flagSet(app, args...)
		return cli.NewContext(app, set, nil)
	}
	t.Run("CLI takes precedence", func(t *testing.T) {
		ctx := set("--spec-dir", "/cli/dir1", "--spec-dir", "/cli/dir2")
		cfg := (&mockConfig{SpecDirs: []string{"/config/dir"}}).toConfig()
		var target struct{ cdiSpecDirs cli.StringSlice }
		ResolveCDIListConfig(ctx, cfg, &target)
		require.Equal(t, []string{"/cli/dir1", "/cli/dir2"}, getStringSliceFieldValue(reflect.ValueOf(target.cdiSpecDirs)))
	})
	t.Run("Config used if CLI not set", func(t *testing.T) {
		ctx := set()
		cfg := (&mockConfig{SpecDirs: []string{"/config/dir1", "/config/dir2"}}).toConfig()
		var target struct{ cdiSpecDirs cli.StringSlice }
		ResolveCDIListConfig(ctx, cfg, &target)
		require.Equal(t, []string{"/config/dir1", "/config/dir2"}, getStringSliceFieldValue(reflect.ValueOf(target.cdiSpecDirs)))
	})
	t.Run("Default used if neither set", func(t *testing.T) {
		ctx := set()
		cfg := (&mockConfig{}).toConfig()
		var target struct{ cdiSpecDirs cli.StringSlice }
		ResolveCDIListConfig(ctx, cfg, &target)
		require.Equal(t, []string{"/etc/cdi", "/var/run/cdi"}, getStringSliceFieldValue(reflect.ValueOf(target.cdiSpecDirs)))
	})
}

// Helper for safely extracting []string from a reflect.Value of cli.StringSlice or *cli.StringSlice
func getStringSliceFieldValue(v reflect.Value) []string {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	ss, ok := v.Interface().(cli.StringSlice)
	if ok {
		return ss.Value()
	}
	return nil
}

// Helper for safely extracting string from a reflect.Value of string or *string
func getStringFieldValue(v reflect.Value) string {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	if v.Kind() == reflect.String {
		return v.String()
	}
	return ""
}

// optsWithSetters is used to test setter-based normalization
type optsWithSetters struct {
	csvFiles          []string
	csvIgnorePatterns []string
}

// Implement the setter methods
func (o *optsWithSetters) SetCSVFiles(files []string)             { o.csvFiles = files }
func (o *optsWithSetters) SetCSVIgnorePatterns(patterns []string) { o.csvIgnorePatterns = patterns }

func TestResolveCDIGenerateOptions(t *testing.T) {
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		&cli.StringSliceFlag{Name: "config-search-path"},
		&cli.StringFlag{Name: "format"},
		&cli.StringFlag{Name: "mode"},
		&cli.StringSliceFlag{Name: "device-name-strategy"},
		&cli.StringFlag{Name: "nvidia-cdi-hook-path"},
		&cli.StringFlag{Name: "ldconfig-path"},
		&cli.StringFlag{Name: "vendor"},
		&cli.StringFlag{Name: "class"},
		&cli.StringSliceFlag{Name: "library-search-path"},
		&cli.StringSliceFlag{Name: "csv.file"},
		&cli.StringSliceFlag{Name: "csv.ignore-pattern"},
		&cli.StringSliceFlag{Name: "disable-hook"},
		&cli.StringFlag{Name: "output"},
		&cli.StringFlag{Name: "driver-root"},
		&cli.StringFlag{Name: "dev-root"},
	}
	set := func(args ...string) *cli.Context {
		set := flagSet(app, args...)
		return cli.NewContext(app, set, nil)
	}
	cfg := (&mockConfig{
		SpecDirs:     []string{"/config/dir"},
		Mode:         "configmode",
		HookPath:     "/config/hook",
		LdconfigPath: "/config/ldconfig",
		CSVSpecPath:  "/config/csv",
	}).toConfig()

	t.Run("All CLI flags", func(t *testing.T) {
		ctx := set(
			"--config-search-path", "/cli/cfg1", "--config-search-path", "/cli/cfg2",
			"--format", "json",
			"--mode", "climode",
			"--device-name-strategy", "uuid",
			"--nvidia-cdi-hook-path", "/cli/hook",
			"--ldconfig-path", "/cli/ldconfig",
			"--vendor", "cli-vendor",
			"--class", "cli-class",
			"--library-search-path", "/cli/lib1",
			"--csv.file", "/cli/csv1",
			"--csv.ignore-pattern", "pat1",
			"--disable-hook", "hook1",
			"--output", "/cli/output",
			"--driver-root", "/cli/driver",
			"--dev-root", "/cli/dev",
		)
		var opts struct {
			Output               string
			Format               string
			DeviceNameStrategies cli.StringSlice
			DriverRoot           string
			DevRoot              string
			NvidiaCDIHookPath    string
			LdconfigPath         string
			Mode                 string
			Vendor               string
			Class                string
			ConfigSearchPaths    cli.StringSlice
			LibrarySearchPaths   cli.StringSlice
			DisabledHooks        cli.StringSlice
			Csv                  struct {
				Files          cli.StringSlice
				IgnorePatterns cli.StringSlice
			}
		}
		ResolveCDIGenerateOptions(ctx, cfg, &opts)
		// Use reflection to check values
		v := reflect.ValueOf(&opts).Elem()
		field := v.FieldByName("Format")
		require.Equal(t, "json", getStringFieldValue(field))
		field = v.FieldByName("Mode")
		require.Equal(t, "climode", getStringFieldValue(field))
		field = v.FieldByName("NvidiaCDIHookPath")
		require.Equal(t, "/cli/hook", getStringFieldValue(field))
		field = v.FieldByName("LdconfigPath")
		require.Equal(t, "/cli/ldconfig", getStringFieldValue(field))
		field = v.FieldByName("Vendor")
		require.Equal(t, "cli-vendor", getStringFieldValue(field))
		field = v.FieldByName("Class")
		require.Equal(t, "cli-class", getStringFieldValue(field))
		field = v.FieldByName("Output")
		require.Equal(t, "/cli/output", getStringFieldValue(field))
		field = v.FieldByName("DriverRoot")
		require.Equal(t, "/cli/driver", getStringFieldValue(field))
		field = v.FieldByName("DevRoot")
		require.Equal(t, "/cli/dev", getStringFieldValue(field))
		require.Equal(t, []string{"uuid"}, getStringSliceFieldValue(v.FieldByName("DeviceNameStrategies")))
		require.Equal(t, []string{"/cli/cfg1", "/cli/cfg2"}, getStringSliceFieldValue(v.FieldByName("ConfigSearchPaths")))
		require.Equal(t, []string{"/cli/lib1"}, getStringSliceFieldValue(v.FieldByName("LibrarySearchPaths")))
		require.Equal(t, []string{"hook1"}, getStringSliceFieldValue(v.FieldByName("DisabledHooks")))
		csvField := v.FieldByName("Csv")
		requireStringSliceEqual(t, []string{"/cli/csv1"}, getStringSliceFieldValue(csvField.FieldByName("Files")))
		requireStringSliceEqual(t, []string{"pat1"}, getStringSliceFieldValue(csvField.FieldByName("IgnorePatterns")))
	})

	t.Run("Config fallback", func(t *testing.T) {
		ctx := set()
		var opts struct {
			Output               string
			Format               string
			DeviceNameStrategies cli.StringSlice
			DriverRoot           string
			DevRoot              string
			NvidiaCDIHookPath    string
			LdconfigPath         string
			Mode                 string
			Vendor               string
			Class                string
			ConfigSearchPaths    cli.StringSlice
			LibrarySearchPaths   cli.StringSlice
			DisabledHooks        cli.StringSlice
			Csv                  struct {
				Files          cli.StringSlice
				IgnorePatterns cli.StringSlice
			}
		}
		ResolveCDIGenerateOptions(ctx, cfg, &opts)
		v := reflect.ValueOf(&opts).Elem()
		require.Equal(t, "configmode", getStringFieldValue(v.FieldByName("Mode")))
		require.Equal(t, "/config/hook", getStringFieldValue(v.FieldByName("NvidiaCDIHookPath")))
		require.Equal(t, "/config/ldconfig", getStringFieldValue(v.FieldByName("LdconfigPath")))
		csvField := v.FieldByName("Csv")
		requireStringSliceEqual(t, []string{"/config/csv"}, getStringSliceFieldValue(csvField.FieldByName("Files")))
		requireStringSliceEqual(t, []string{}, getStringSliceFieldValue(csvField.FieldByName("IgnorePatterns")))
	})

	t.Run("Default fallback", func(t *testing.T) {
		ctx := set()
		cfg := (&mockConfig{}).toConfig()
		var opts struct {
			Output               string
			Format               string
			DeviceNameStrategies cli.StringSlice
			DriverRoot           string
			DevRoot              string
			NvidiaCDIHookPath    string
			LdconfigPath         string
			Mode                 string
			Vendor               string
			Class                string
			ConfigSearchPaths    cli.StringSlice
			LibrarySearchPaths   cli.StringSlice
			DisabledHooks        cli.StringSlice
			Csv                  struct {
				Files          cli.StringSlice
				IgnorePatterns cli.StringSlice
			}
		}
		ResolveCDIGenerateOptions(ctx, cfg, &opts)
		v := reflect.ValueOf(&opts).Elem()
		require.Equal(t, "auto", getStringFieldValue(v.FieldByName("Mode")))
		require.Equal(t, "yaml", getStringFieldValue(v.FieldByName("Format")))
		require.Equal(t, []string{"index", "uuid"}, getStringSliceFieldValue(v.FieldByName("DeviceNameStrategies")))
		require.Equal(t, "nvidia.com", getStringFieldValue(v.FieldByName("Vendor")))
		require.Equal(t, "gpu", getStringFieldValue(v.FieldByName("Class")))
		csvField := v.FieldByName("Csv")
		requireStringSliceEqual(t, csv.DefaultFileList(), getStringSliceFieldValue(csvField.FieldByName("Files")))
		requireStringSliceEqual(t, []string{}, getStringSliceFieldValue(csvField.FieldByName("IgnorePatterns")))
	})
}

func TestResolveCDIGenerateOptions_SetterMethods(t *testing.T) {
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		&cli.StringSliceFlag{Name: "csv.file"},
		&cli.StringSliceFlag{Name: "csv.ignore-pattern"},
	}
	set := func(args ...string) *cli.Context {
		set := flagSet(app, args...)
		return cli.NewContext(app, set, nil)
	}
	cfg := (&mockConfig{
		CSVSpecPath: "/config/csv",
	}).toConfig()

	t.Run("CLI takes precedence", func(t *testing.T) {
		ctx := set("--csv.file", "/cli/csv1", "--csv.file", "/cli/csv2", "--csv.ignore-pattern", "pat1")
		opts := &optsWithSetters{}
		ResolveCDIGenerateOptions(ctx, cfg, opts)
		requireStringSliceEqual(t, []string{"/cli/csv1", "/cli/csv2"}, opts.csvFiles)
		requireStringSliceEqual(t, []string{"pat1"}, opts.csvIgnorePatterns)
	})

	t.Run("Config fallback", func(t *testing.T) {
		ctx := set()
		opts := &optsWithSetters{}
		ResolveCDIGenerateOptions(ctx, cfg, opts)
		requireStringSliceEqual(t, []string{"/config/csv"}, opts.csvFiles)
		requireStringSliceEqual(t, []string{}, opts.csvIgnorePatterns)
	})

	t.Run("Default fallback", func(t *testing.T) {
		ctx := set()
		cfg := (&mockConfig{}).toConfig()
		opts := &optsWithSetters{}
		ResolveCDIGenerateOptions(ctx, cfg, opts)
		requireStringSliceEqual(t, csv.DefaultFileList(), opts.csvFiles)
		requireStringSliceEqual(t, []string{}, opts.csvIgnorePatterns)
	})
}

// Helper to create a cli.FlagSet for testing
func flagSet(app *cli.App, args ...string) *flag.FlagSet {
	set := flag.NewFlagSet(app.Name, flag.ContinueOnError)
	for _, f := range app.Flags {
		_ = f.Apply(set)
	}
	_ = set.Parse(args)
	return set
}

// Helper to compare two string slices, treating nil and empty as equal
func requireStringSliceEqual(t *testing.T, expected, actual []string, msgAndArgs ...interface{}) {
	if expected == nil {
		expected = []string{}
	}
	if actual == nil {
		actual = []string{}
	}
	require.Equal(t, expected, actual, msgAndArgs...)
}
