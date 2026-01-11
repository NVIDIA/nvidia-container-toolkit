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
	nriplugin "github.com/containerd/nri/pkg/stub"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

// Compile-time interface checks
var (
	_ nriplugin.Plugin = (*Plugin)(nil)
)

const (
	// nriCDIDeviceDomain is the domain name used to denote an nvidia cdi device
	nriCDIDeviceDomain = "nvidia.cdi.k8s.io"
	// nriCDIDeviceKey is the prefix of the key used for CDI device annotations
	nriCDIDeviceKey = nriCDIDeviceDomain + "/container"
	// defaultNRISocket represents the default path of the NRI socket
	defaultNRISocket = "/var/run/nri/nri.sock"
)

type Plugin struct {
	logger logger.Interface

	stub nriplugin.Stub
}

// NewPlugin creates a new NRI plugin for injecting CDI devices
func NewPlugin(logger logger.Interface) *Plugin {
	return &Plugin{
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
	devices, err := parseCDIDevices(ctr.Name, pod.Annotations)
	if err != nil {
		return err
	}

	if len(devices) == 0 {
		p.logger.Debugf("%s: no CDI devices annotated...", containerName(pod, ctr))
		return nil
	}

	for _, name := range devices {
		a.AddCDIDevice(
			&api.CDIDevice{
				Name: name,
			},
		)
		p.logger.Infof("%s: injected CDI device %q...", containerName(pod, ctr), name)
	}

	return nil
}

func parseCDIDevices(ctr string, annotations map[string]string) ([]string, error) {
	annotation := getCDIDeviceFromAnnotation(annotations, ctr)
	if len(annotation) == 0 {
		return nil, nil
	}

	cdiDevices := strings.Split(annotation, ",")
	return cdiDevices, nil
}

func getCDIDeviceFromAnnotation(annotations map[string]string, ctr string) string {
	nriPluginAnnotationKey := fmt.Sprintf("%s.%s", nriCDIDeviceKey, ctr)
	if value, ok := annotations[nriPluginAnnotationKey]; ok {
		return value
	}

	// If there isn't an exact match, we look for a wildcard character match
	for annotationKey := range annotations {
		if after, found := strings.CutPrefix(annotationKey, fmt.Sprintf("%s.", nriCDIDeviceKey)); found {
			if after == "*" {
				return annotations[annotationKey]
			}
		}
	}

	return ""
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
	if len(nriSocketPath) == 0 {
		nriSocketPath = defaultNRISocket
	}
	_, err := os.Stat(nriSocketPath)
	if err != nil {
		return fmt.Errorf("failed to find valid nri socket in %s: %w", nriSocketPath, err)
	}

	pluginOpts := []nriplugin.Option{
		nriplugin.WithPluginIdx(nriPluginIdx),
		nriplugin.WithSocketPath(nriSocketPath),
	}
	if p.stub, err = nriplugin.New(p, pluginOpts...); err != nil {
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
	if p != nil && p.stub != nil {
		p.stub.Stop()
	}
}
