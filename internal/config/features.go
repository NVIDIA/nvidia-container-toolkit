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
type feature bool

const (
	FeatureAllowAdditionalGIDs = featureName("allow-additional-gids")
	FeatureGDRCopy             = featureName("gdrcopy")
	FeatureGDS                 = featureName("gds")
	FeatureMOFED               = featureName("mofed")
	FeatureNVSWITCH            = featureName("nvswitch")

	featureEnabled  feature = true
	featureDisabled feature = false
)

// features specifies a set of named features.
type features struct {
	GDS      *feature `toml:"gds,omitempty"`
	MOFED    *feature `toml:"mofed,omitempty"`
	NVSWITCH *feature `toml:"nvswitch,omitempty"`
	GDRCopy  *feature `toml:"gdrcopy,omitempty"`
	// AllowAdditionalGIDs triggers the additionalGIDs field in internal CDI
	// specifications to be populated if required. This can be useful when
	// running the container as a user id that may not have access to device
	// nodes.
	AllowAdditionalGIDs *feature `toml:"allow-additional-gids,omitempty"`
}

// IsEnabled checks whether a specified named feature is enabled.
// An optional list of environments to check for feature-specific environment
// variables can also be supplied.
func (fs features) IsEnabled(n featureName, in ...getenver) bool {
	featureEnvvars := map[featureName]string{
		FeatureGDS:                 "NVIDIA_GDS",
		FeatureMOFED:               "NVIDIA_MOFED",
		FeatureNVSWITCH:            "NVIDIA_NVSWITCH",
		FeatureGDRCopy:             "NVIDIA_GDRCOPY",
		FeatureAllowAdditionalGIDs: "NVIDIA_ALLOW_ADDITIONAL_GIDS",
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
	case FeatureAllowAdditionalGIDs:
		return fs.AllowAdditionalGIDs.isEnabled(envvar, in...)
	default:
		return false
	}
}

// isEnabled checks whether a feature is enabled.
// If the enabled value is explicitly set, this is returned, otherwise the
// associated envvar is checked in the specified getenver for the string "enabled"
// A CUDA container / image can be passed here.
func (f *feature) isEnabled(envvar string, ins ...getenver) bool {
	if envvar != "" {
		for _, in := range ins {
			switch in.Getenv(envvar) {
			case "enabled", "true":
				return true
			case "disabled", "false":
				return false
			}
		}
	}
	if f != nil {
		return bool(*f)
	}
	return false
}

type getenver interface {
	Getenv(string) string
}
