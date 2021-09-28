package modify

import (
	"strings"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/discover"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
	log "github.com/sirupsen/logrus"
)

// Mount is an alias to discover.Mount that allows for addition of a Modify method
type Mount struct {
	logger *log.Logger
	discover.Mount
	root string
}

var _ Modifier = (*Mount)(nil)

// Modify applies the modifications required for a Mount to the specified OCI specification
func (d Mount) Modify(spec oci.Spec) error {
	return spec.Modify(d.specModifier)
}

// TODO: We need to ensure that we are correctly mounting the proc paths
// Also — I’m not sure how this is done, but we will need a new tempfs mounted at /proc/driver/nvidia/ underneath which all of these other mounted directories get put
// Maybe this?
// https://github.com/opencontainers/runtime-spec/blob/master/specs-go/config.go#L175 (edited)
// specs-go/config.go:175
//     MaskedPaths []string `json:"maskedPaths,omitempty"`
// <https://github.com/opencontainers/runtime-spec|opencontainers/runtime-spec>opencontainers/runtime-spec | Added by GitHub
// 13:53
// Proabably, given…
// https://github.com/opencontainers/runtime-spec/blob/master/config-linux.md#masked-paths (edited)
// TODO: We can try masking all of /proc/driver/nvidia and then mounting the paths read-only
func (d Mount) specModifier(spec *specs.Spec) error {
	source := d.Path
	destination := strings.TrimPrefix(d.Path, d.root)
	d.logger.Infof("Mounting %v -> %v", source, destination)
	mount := specs.Mount{
		Destination: destination,
		Source:      source,
		Type:        "bind",
		Options: []string{
			"rbind",
			"rprivate",
		},
	}
	spec.Mounts = append(spec.Mounts, mount)

	return nil
}
