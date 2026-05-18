package nri

import (
	"context"
	"strings"

	"github.com/containerd/nri/pkg/api"
	nrilog "github.com/containerd/nri/pkg/log"
	"github.com/containerd/nri/pkg/plugin"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

type cdiInjectorPlugin struct {
	logger    nrilog.Logger
	namespace string
}

func NewCDIDeviceInjector(logger logger.Interface, namespace string) interface{} {
	return &cdiInjectorPlugin{
		logger: toNriLogger{
			logger,
		},
		namespace: namespace,
	}
}

// CreateContainer handles container creation requests.
func (c *cdiInjectorPlugin) CreateContainer(ctx context.Context, pod *api.PodSandbox, ctr *api.Container) (*api.ContainerAdjustment, []*api.ContainerUpdate, error) {
	adjust := &api.ContainerAdjustment{}

	if err := c.injectCDIDevices(ctx, pod, ctr, adjust); err != nil {
		return nil, nil, err
	}

	return adjust, nil, nil
}

func (c *cdiInjectorPlugin) injectCDIDevices(ctx context.Context, pod *api.PodSandbox, ctr *api.Container, a *api.ContainerAdjustment) error {

	devices := c.parseCDIDevices(ctx, pod, nriCDIAnnotationDomain, ctr.Name)
	if len(devices) == 0 {
		c.logger.Debugf(ctx, "%s: no CDI devices annotated...", containerName(pod, ctr))
		return nil
	}

	c.logger.Infof(ctx, "%s: injecting CDI devices %v...", containerName(pod, ctr), devices)
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
func (c *cdiInjectorPlugin) parseCDIDevices(ctx context.Context, pod *api.PodSandbox, key, container string) []string {
	if c.namespace != pod.Namespace {
		c.logger.Debugf(ctx, "pod %s/%s is not in the toolkit's namespace %s. Skipping CDI device injection...", pod.Namespace, pod.Name, c.namespace)
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
