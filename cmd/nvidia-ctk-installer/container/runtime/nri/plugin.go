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

package nri

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/containerd/nri/pkg/api"
	"github.com/containerd/nri/pkg/plugin"
	"github.com/containerd/nri/pkg/stub"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

// Compile-time interface checks
var (
	_ stub.Plugin = (*Plugin)(nil)
)

const (
	// nriCDIAnnotationDomain is the domain name used for CDI device annotations
	nriCDIAnnotationDomain = "nvidia.cdi.k8s.io"
)

type Plugin struct {
	ctx    context.Context
	logger logger.Interface

	stub stub.Stub
}

// NewPlugin creates a new NRI plugin for injecting CDI devices
func NewPlugin(ctx context.Context, logger logger.Interface) *Plugin {
	return &Plugin{
		ctx:    ctx,
		logger: logger,
	}
}

// CreateContainer handles container creation requests.
func (p *Plugin) CreateContainer(_ context.Context, pod *api.PodSandbox, ctr *api.Container) (*api.ContainerAdjustment, []*api.ContainerUpdate, error) {
	adjust := &api.ContainerAdjustment{}

	if err := p.injectCDIDevices(pod, ctr, adjust); err != nil {
		return nil, nil, err
	}

	return adjust, nil, nil
}

func (p *Plugin) injectCDIDevices(pod *api.PodSandbox, ctr *api.Container, a *api.ContainerAdjustment) error {
	ctx := p.ctx
	pluginLogger := p.stub.Logger()

	devices := parseCDIDevices(pod, nriCDIAnnotationDomain, ctr.Name)
	if len(devices) == 0 {
		pluginLogger.Debugf(ctx, "%s: no CDI devices annotated...", containerName(pod, ctr))
		return nil
	}

	for _, name := range devices {
		a.AddCDIDevice(
			&api.CDIDevice{
				Name: name,
			},
		)
		pluginLogger.Infof(ctx, "%s: injected CDI device %q...", containerName(pod, ctr), name)
	}

	return nil
}

func parseCDIDevices(pod *api.PodSandbox, key, container string) []string {
	cdiDeviceNames, ok := plugin.GetEffectiveAnnotation(pod, key, container)
	if !ok {
		return nil
	}

	cdiDevices := strings.Split(cdiDeviceNames, ",")
	return cdiDevices
}

// Construct a container name for log messages.
func containerName(pod *api.PodSandbox, container *api.Container) string {
	if pod != nil {
		return pod.Name + "/" + container.Name
	}
	return container.Name
}

// Start starts the NRI plugin
func (p *Plugin) Start(ctx context.Context, nriSocketPath, nriPluginIdx string) error {
	pluginOpts := []stub.Option{
		stub.WithPluginIdx(nriPluginIdx),
		stub.WithLogger(toNriLogger{p.logger}),
	}
	if len(nriSocketPath) > 0 {
		_, err := os.Stat(nriSocketPath)
		if err != nil {
			return fmt.Errorf("failed to find valid nri socket in %s: %w", nriSocketPath, err)
		}
		pluginOpts = append(pluginOpts, stub.WithSocketPath(nriSocketPath))
	}

	var err error
	if p.stub, err = stub.New(p, pluginOpts...); err != nil {
		return fmt.Errorf("failed to initialise plugin at %s: %w", nriSocketPath, err)
	}
	err = p.stub.Start(ctx)
	if err != nil {
		return fmt.Errorf("plugin exited with error: %w", err)
	}
	return nil
}

// Stop stops the NRI plugin
func (p *Plugin) Stop() {
	if p == nil || p.stub == nil {
		return
	}
	p.stub.Stop()
}
