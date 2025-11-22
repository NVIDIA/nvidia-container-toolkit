package nri

import (
	"context"
	"fmt"

	"github.com/containerd/nri/pkg/api"
	"github.com/containerd/nri/pkg/stub"
	"sigs.k8s.io/yaml"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

const (
	// nodeResourceCDIDeviceKey is the prefix of the key used for CDI device annotations.
	nodeResourceCDIDeviceKey = "cdi-devices.noderesource.dev"
	// Prefix of the key used for CDI device annotations.
	nriCDIDeviceKey = "cdi-devices.nri.io"
)

type Plugin struct {
	logger logger.Interface

	Stub stub.Stub
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
