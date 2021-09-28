package modify

import (
	"fmt"
	"os"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
	log "github.com/sirupsen/logrus"
)

// Device is an alias to discover.Device that allows for addition of a Modify method
type Device struct {
	logger *log.Logger
	discover.Device
}

// ProcMount is an alias to discover.Mount that allows for the addition of a Modify method for
// proc paths associated with devices
type ProcMount struct {
	logger *log.Logger
	discover.ProcPath
}

var _ Modifier = (*Device)(nil)
var _ Modifier = (*ProcMount)(nil)

// Modify applies the modifications required by a Device to the specified OCI specification
func (d Device) Modify(spec oci.Spec) error {
	for _, dn := range d.DeviceNodes {
		mi := deviceNode{
			logger:     d.logger,
			DeviceNode: dn,
		}
		err := mi.Modify(spec)
		if err != nil {
			return fmt.Errorf("could not inject device node %v: %v", dn, err)
		}
	}

	for _, p := range d.ProcPaths {
		mi := ProcMount{
			logger:   d.logger,
			ProcPath: p,
		}
		err := mi.Modify(spec)
		if err != nil {
			return fmt.Errorf("could not inject proc path %v: %v", p, err)
		}
	}
	return nil
}

type deviceNode struct {
	logger *log.Logger
	discover.DeviceNode
}

func (d deviceNode) Modify(spec oci.Spec) error {
	return spec.Modify(d.specModifier)
}

func (d deviceNode) specModifier(spec *specs.Spec) error {
	if spec.Linux == nil {
		d.logger.Debugf("Initializing spec.Linux")
		spec.Linux = &specs.Linux{}
	}
	if spec.Linux.Resources == nil {
		d.logger.Debugf("Initializing spec.LinuxResources")
		spec.Linux.Resources = &specs.LinuxResources{}
	}

	// TODO: These need to be configurable
	deviceFileMode := os.FileMode(8630)
	deviceUID := uint32(0)
	deviceGID := uint32(0)

	deviceMajor := int64(d.Major)
	deviceMinor := int64(d.Minor)

	d.logger.Infof("Adding device %v", d.Path)
	ociDevice := specs.LinuxDevice{
		Path:     string(d.Path),
		Type:     "c",
		Major:    deviceMajor,
		Minor:    deviceMinor,
		FileMode: &deviceFileMode,
		UID:      &deviceUID,
		GID:      &deviceGID,
	}
	spec.Linux.Devices = append(spec.Linux.Devices, ociDevice)

	ociDeviceCgroup := specs.LinuxDeviceCgroup{
		Allow:  true,
		Type:   "c",
		Major:  &deviceMajor,
		Minor:  &deviceMinor,
		Access: "rwm",
	}

	// TODO: We have to handle the case where we are updating the cgroups for multiple devices
	// leading to duplicates in the spec
	spec.Linux.Resources.Devices = append(spec.Linux.Resources.Devices, ociDeviceCgroup)

	return nil
}

// Modify applies the modifications required for a Mount to the specified OCI specification
func (m ProcMount) Modify(spec oci.Spec) error {
	return spec.Modify(m.specModifier)
}

func (m ProcMount) specModifier(spec *specs.Spec) error {
	m.logger.Infof("Mounting read-only proc path %v", m.ProcPath)
	spec.Linux.ReadonlyPaths = append(spec.Linux.ReadonlyPaths, string(m.ProcPath))
	return nil
}
