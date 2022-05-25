/**
# Copyright (c) 2020-2021, NVIDIA CORPORATION.  All rights reserved.
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

package main

import (
	"github.com/pelletier/go-toml"
)

// configV2 represents a V2 containerd config
type configV2 struct {
	config
}

func newConfigV2(cfg *toml.Tree) UpdateReverter {
	c := configV2{
		config: config{
			Tree:    cfg,
			version: 2,
			cri:     "io.containerd.grpc.v1.cri",
		},
	}

	return &c
}

// Update performs an update specific to v2 of the containerd config
func (config *configV2) Update(o *options) error {
	defaultRuntime := o.getDefaultRuntime()
	for runtimeClass, runtimeBinary := range o.getRuntimeBinaries() {
		setAsDefault := defaultRuntime == runtimeClass
		config.update(runtimeClass, o.runtimeType, runtimeBinary, setAsDefault)
	}

	return nil
}

// Revert performs a revert specific to v2 of the containerd config
func (config *configV2) Revert(o *options) error {
	for runtimeClass := range o.getRuntimeBinaries() {
		config.revert(runtimeClass)
	}

	return nil
}
