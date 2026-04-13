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
	"sync/atomic"
	"time"

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

	// nriReconnectBackoff is the backoff time between retries when attempting to connect the NRI Plugin to the ttrpc server
	nriReconnectBackoff = 2 * time.Second
)

type Plugin struct {
	ctx    context.Context
	logger logger.Interface

	namespace string
	stub      stub.Stub

	// stopped is set before Stop() so OnClose does not reconnect during shutdown.
	stopped atomic.Bool
	// reconnectInProgress ensures that only one NRI plugin reconnect operation runs at any given time.
	reconnectInProgress atomic.Bool
}

// NewPlugin creates a new NRI plugin for injecting CDI devices
func NewPlugin(ctx context.Context, logger logger.Interface, namespace string) *Plugin {
	return &Plugin{
		ctx:       ctx,
		logger:    logger,
		namespace: namespace,
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

	devices := p.parseCDIDevices(pod, nriCDIAnnotationDomain, ctr.Name)
	if len(devices) == 0 {
		pluginLogger.Debugf(ctx, "%s: no CDI devices annotated...", containerName(pod, ctr))
		return nil
	}

	pluginLogger.Infof(ctx, "%s: injecting CDI devices %v...", containerName(pod, ctr), devices)
	for _, name := range devices {
		a.AddCDIDevice(
			&api.CDIDevice{
				Name: name,
			},
		)
	}

	return nil
}

// parseCDIDevices processes the podSpec and determines which containers which need CDI devices injected to them
func (p *Plugin) parseCDIDevices(pod *api.PodSandbox, key, container string) []string {
	if p.namespace != pod.Namespace {
		p.logger.Debugf("pod %s/%s is not in the toolkit's namespace %s. Skipping CDI device injection...", pod.Namespace, pod.Name, p.namespace)
		return nil
	}

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
		stub.WithOnClose(func() {
			p.logger.Infof("NRI ttrpc connection to %s is down; attempting to reconnect...", nriSocketPath)
			p.scheduleReconnect(nriSocketPath)
		}),
	}
	if len(nriSocketPath) > 0 {
		_, err := os.Stat(nriSocketPath)
		if err != nil {
			return fmt.Errorf("failed to find valid nri socket at %s: %w", nriSocketPath, err)
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

// scheduleReconnect runs stub.Start in a loop until success, shutdown, or context cancellation.
func (p *Plugin) scheduleReconnect(nriSocketPath string) {
	if !p.reconnectInProgress.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer p.reconnectInProgress.Store(false)
		for i := 1; ; i++ {
			if p.stopped.Load() {
				p.logger.Infof("NRI plugin stopped. Stopping all reconnect attempts...")
				return
			}
			select {
			case <-p.ctx.Done():
				return
			case <-time.After(nriReconnectBackoff):
			}
			p.logger.Infof("NRI plugin reconnecting to %s (attempt %d)...", nriSocketPath, i)
			if err := p.stub.Start(p.ctx); err != nil {
				p.logger.Warningf("NRI plugin reconnect failed: %v", err)
				if p.stopped.Load() {
					p.logger.Infof("NRI plugin stopped. Stopping all reconnect attempts...")
					return
				}
				continue
			}
			p.logger.Infof("NRI plugin reconnected to %s", nriSocketPath)
			return
		}
	}()
}

// Stop stops the NRI plugin
func (p *Plugin) Stop() {
	if p == nil || p.stub == nil {
		return
	}
	p.stopped.Store(true)
	p.stub.Stop()
}
