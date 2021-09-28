/*
# Copyright (c) 2021, NVIDIA CORPORATION.  All rights reserved.
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
	"encoding/json"
	"os"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/filter"
	log "github.com/sirupsen/logrus"
)

func main() {
	d, err := discover.NewNVMLServer("")
	if err != nil {
		log.Errorf("Error discovering devices: %v", err)
	}

	selected := filter.NewSelectDevicesFrom(d, "all", nil)

	devices, err := selected.Devices()
	if err != nil {
		log.Errorf("Error discovering devices: %v", err)
		return
	}

	mounts, err := selected.Mounts()
	if err != nil {
		log.Errorf("Error discovering mounts: %v", err)
		return
	}

	hooks, err := selected.Hooks()
	if err != nil {
		log.Errorf("Error discovering hooks: %v", err)
		return
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	log.Infof("Discovered devices:")
	enc.Encode(devices)

	log.Infof("Discovered libraries:")
	enc.Encode(mounts)

	log.Infof("Discovered hook:")
	enc.Encode(hooks)
}
