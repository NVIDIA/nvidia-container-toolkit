/*
# Copyright (c) 2022, NVIDIA CORPORATION.  All rights reserved.
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
*/

package nvidia

const (
	// RuntimeName is the default name to use in configs for the NVIDIA Container Runtime
	RuntimeName = "nvidia"
	// RuntimeExecutable is the default NVIDIA Container Runtime executable file name
	RuntimeExecutable = "nvidia-container-runtime"
)

// Options specifies the options for the NVIDIA Container Runtime w.r.t a container engine such as docker.
type Options struct {
	SetAsDefault bool
	RuntimeName  string
	RuntimePath  string
}

// Runtime defines an NVIDIA runtime with a name and a executable
type Runtime struct {
	Name string
	Path string
}

// DefaultRuntime returns the default runtime for the configured options.
// If the configuration is invalid or the default runtimes should not be set
// the empty string is returned.
func (o Options) DefaultRuntime() string {
	if !o.SetAsDefault {
		return ""
	}

	return o.RuntimeName
}

// Runtime creates a runtime struct based on the options.
func (o Options) Runtime() Runtime {
	path := o.RuntimePath

	if o.RuntimePath == "" {
		path = RuntimeExecutable
	}

	r := Runtime{
		Name: o.RuntimeName,
		Path: path,
	}

	return r
}

// DockerRuntimesConfig generatest the expected docker config for the specified runtime
func (r Runtime) DockerRuntimesConfig() map[string]interface{} {
	runtimes := make(map[string]interface{})
	runtimes[r.Name] = map[string]interface{}{
		"path": r.Path,
		"args": []string{},
	}

	return runtimes
}
