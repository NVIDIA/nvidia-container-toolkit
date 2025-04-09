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

package dgpu

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/NVIDIA/go-nvml/pkg/nvml"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/nvsandboxutils"
)

type nvsandboxutilsDGPU struct {
	lib         nvsandboxutils.Interface
	uuid        string
	devRoot     string
	isMig       bool
	hookCreator discover.HookCreator
	deviceLinks []string
}

var _ discover.Discover = (*nvsandboxutilsDGPU)(nil)

type UUIDer interface {
	GetUUID() (string, nvml.Return)
}

func (o *options) newNvsandboxutilsDGPUDiscoverer(d UUIDer) (discover.Discover, error) {
	if o.nvsandboxutilslib == nil {
		return nil, nil
	}

	uuid, nvmlRet := d.GetUUID()
	if nvmlRet != nvml.SUCCESS {
		return nil, fmt.Errorf("failed to get device UUID: %w", nvmlRet)
	}

	nvd := nvsandboxutilsDGPU{
		lib:         o.nvsandboxutilslib,
		uuid:        uuid,
		devRoot:     strings.TrimSuffix(filepath.Clean(o.devRoot), "/dev"),
		isMig:       o.isMigDevice,
		hookCreator: o.hookCreator,
	}

	return &nvd, nil
}

func (d *nvsandboxutilsDGPU) Devices() ([]discover.Device, error) {
	gpuFileInfos, ret := d.lib.GetGpuResource(d.uuid)
	if ret != nvsandboxutils.SUCCESS {
		return nil, fmt.Errorf("failed to get GPU resource: %w", ret)
	}

	var devices []discover.Device
	for _, info := range gpuFileInfos {
		switch info.SubType {
		case nvsandboxutils.NV_DEV_DRI_CARD, nvsandboxutils.NV_DEV_DRI_RENDERD:
			if d.isMig {
				continue
			}
			fallthrough
		case nvsandboxutils.NV_DEV_NVIDIA, nvsandboxutils.NV_DEV_NVIDIA_CAPS_NVIDIA_CAP:
			containerPath := info.Path
			if d.devRoot != "/" {
				containerPath = strings.TrimPrefix(containerPath, d.devRoot)
			}

			// TODO: Extend discover.Device with additional information.
			device := discover.Device{
				HostPath: info.Path,
				Path:     containerPath,
			}
			devices = append(devices, device)
		case nvsandboxutils.NV_DEV_DRI_CARD_SYMLINK, nvsandboxutils.NV_DEV_DRI_RENDERD_SYMLINK:
			if d.isMig {
				continue
			}
			if info.Flags == nvsandboxutils.NV_FILE_FLAG_CONTENT {
				targetPath, ret := d.lib.GetFileContent(info.Path)
				if ret != nvsandboxutils.SUCCESS {
					return nil, fmt.Errorf("failed to get symlink: %w", ret)
				}
				d.deviceLinks = append(d.deviceLinks, fmt.Sprintf("%v::%v", targetPath, info.Path))
			}
		}
	}

	return devices, nil
}

func (d *nvsandboxutilsDGPU) EnvVars() ([]discover.EnvVar, error) {
	return nil, nil
}

// Hooks returns a hook to create the by-path symlinks for the discovered devices.
func (d *nvsandboxutilsDGPU) Hooks() ([]discover.Hook, error) {
	if len(d.deviceLinks) == 0 {
		return nil, nil
	}

	hook := d.hookCreator.Create("create-symlinks", d.deviceLinks...)

	return hook.Hooks()
}

func (d *nvsandboxutilsDGPU) Mounts() ([]discover.Mount, error) {
	return nil, nil
}
