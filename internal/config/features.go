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

package config

type featureName string

const (
	FeatureGDS      = featureName("gds")
	FeatureMOFED    = featureName("mofed")
	FeatureNVSWITCH = featureName("nvswitch")
	FeatureGDRCopy  = featureName("gdrcopy")
)

// features specifies a set of named features.
type features struct {
	GDS      *feature `toml:"gds,omitempty"`
	MOFED    *feature `toml:"mofed,omitempty"`
	NVSWITCH *feature `toml:"nvswitch,omitempty"`
	GDRCopy  *feature `toml:"gdrcopy,omitempty"`
}

type feature bool

// IsEnabled checks whether a specified named feature is enabled.
// An optional list of environments to check for feature-specific environment
// variables can also be supplied.
func (fs features) IsEnabled(n featureName, in ...getenver) bool {
	featureEnvvars := map[featureName]string{
		FeatureGDS:      "NVIDIA_GDS",
		FeatureMOFED:    "NVIDIA_MOFED",
		FeatureNVSWITCH: "NVIDIA_NVSWITCH",
		FeatureGDRCopy:  "NVIDIA_GDRCOPY",
	}

	envvar := featureEnvvars[n]
	switch n {
	case FeatureGDS:
		return fs.GDS.isEnabled(envvar, in...)
	case FeatureMOFED:
		return fs.MOFED.isEnabled(envvar, in...)
	case FeatureNVSWITCH:
		return fs.NVSWITCH.isEnabled(envvar, in...)
	case FeatureGDRCopy:
		return fs.GDRCopy.isEnabled(envvar, in...)
	default:
		return false
	}
}

// isEnabled checks whether a feature is enabled.
// If the enabled value is explicitly set, this is returned, otherwise the
// associated envvar is checked in the specified getenver for the string "enabled"
// A CUDA container / image can be passed here.
func (f *feature) isEnabled(envvar string, ins ...getenver) bool {
	if f != nil {
		return bool(*f)
	}
	if envvar == "" {
		return false
	}
	for _, in := range ins {
		if in.Getenv(envvar) == "enabled" {
			return true
		}
	}
	return false
}

type getenver interface {
	Getenv(string) string
}
