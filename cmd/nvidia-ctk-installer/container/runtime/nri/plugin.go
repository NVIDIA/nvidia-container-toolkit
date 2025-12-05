package nri

import (
	"context"
	"fmt"
	"os"

	"github.com/containerd/nri/pkg/api"
	nriplugin "github.com/containerd/nri/pkg/stub"
	"sigs.k8s.io/yaml"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

// Compile-time interface checks
var (
	_ nriplugin.Plugin = (*Plugin)(nil)
)

const (
	// nodeResourceCDIDeviceKey is the prefix of the key used for CDI device annotations.
	nodeResourceCDIDeviceKey = "cdi-devices.noderesource.dev"
	// nriCDIDeviceKey is the prefix of the key used for CDI device annotations.
	nriCDIDeviceKey = "cdi-devices.nri.io"
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
	var (
		cdiDevices []string
	)

	annotation := getAnnotation(annotations, nodeResourceCDIDeviceKey, nriCDIDeviceKey, ctr)
	if len(annotation) == 0 {
		return nil, nil
	}

	if err := yaml.Unmarshal(annotation, &cdiDevices); err != nil {
		return nil, fmt.Errorf("invalid CDI device annotation %q: %w", string(annotation), err)
	}

	return cdiDevices, nil
}

func getAnnotation(annotations map[string]string, mainKey, oldKey, ctr string) []byte {
	for _, key := range []string{
		mainKey + "/container." + ctr,
		oldKey + "/container." + ctr,
		mainKey + "/pod",
		oldKey + "/pod",
		mainKey,
		oldKey,
	} {
		if value, ok := annotations[key]; ok {
			return []byte(value)
		}
	}

	return nil
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

	var pluginOpts []nriplugin.Option
	pluginOpts = append(pluginOpts, nriplugin.WithPluginIdx(nriPluginIdx))
	pluginOpts = append(pluginOpts, nriplugin.WithSocketPath(nriSocketPath))
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
