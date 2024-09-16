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
	FeatureGDS                        = featureName("gds")
	FeatureMOFED                      = featureName("mofed")
	FeatureNVSWITCH                   = featureName("nvswitch")
	FeatureGDRCopy                    = featureName("gdrcopy")
	FeatureAllowLDConfigFromContainer = featureName("allow-ldconfig-from-container")
	FeatureIncludePersistencedSocket  = featureName("include-persistenced-socket")
)

// features specifies a set of named features.
type features struct {
	GDS      *feature `toml:"gds,omitempty"`
	MOFED    *feature `toml:"mofed,omitempty"`
	NVSWITCH *feature `toml:"nvswitch,omitempty"`
	GDRCopy  *feature `toml:"gdrcopy,omitempty"`
	// AllowLDConfigFromContainer allows non-host ldconfig paths to be used.
	// If this feature flag is not set to 'true' only host-rooted config paths
	// (i.e. paths starting with an '@' are considered valid)
	AllowLDConfigFromContainer *feature `toml:"allow-ldconfig-from-container,omitempty"`
	// IncludePersistencedSocket enables the injection of the nvidia-persistenced
	// socket into containers.
	IncludePersistencedSocket *feature `toml:"include-persistenced-socket,omitempty"`
}

type feature bool

// IsEnabledInEnvironment checks whether a specified named feature is enabled.
// An optional list of environments to check for feature-specific environment
// variables can also be supplied.
func (fs features) IsEnabledInEnvironment(n featureName, in ...getenver) bool {
	switch n {
	// Features with envvar overrides
	case FeatureGDS:
		return fs.GDS.isEnabledWithEnvvarOverride("NVIDIA_GDS", in...)
	case FeatureMOFED:
		return fs.MOFED.isEnabledWithEnvvarOverride("NVIDIA_MOFED", in...)
	case FeatureNVSWITCH:
		return fs.NVSWITCH.isEnabledWithEnvvarOverride("NVIDIA_NVSWITCH", in...)
	case FeatureGDRCopy:
		return fs.GDRCopy.isEnabledWithEnvvarOverride("NVIDIA_GDRCOPY", in...)
	// Features without envvar overrides
	case FeatureAllowLDConfigFromContainer:
		return fs.AllowLDConfigFromContainer.IsEnabled()
	case FeatureIncludePersistencedSocket:
		return fs.IncludePersistencedSocket.IsEnabled()
	default:
		return false
	}
}

// IsEnabled checks whether a feature is enabled.
func (f *feature) IsEnabled() bool {
	if f != nil {
		return bool(*f)
	}
	return false
}

// isEnabledWithEnvvarOverride checks whether a feature is enabled and allows an envvar to overide the feature.
// If the enabled value is explicitly set, this is returned, otherwise the
// associated envvar is checked in the specified getenver for the string "enabled"
// A CUDA container / image can be passed here.
func (f *feature) isEnabledWithEnvvarOverride(envvar string, ins ...getenver) bool {
	if envvar != "" {
		for _, in := range ins {
			if in.Getenv(envvar) == "enabled" {
				return true
			}
		}
	}

	return f.IsEnabled()
}

type getenver interface {
	Getenv(string) string
}
