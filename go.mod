module github.com/NVIDIA/nvidia-container-toolkit

go 1.25.0

require (
	github.com/NVIDIA/go-nvlib v0.9.1-0.20251202135446-d0f42ba016dd
	github.com/NVIDIA/go-nvml v0.13.0-1
	github.com/containerd/nri v0.11.0
	github.com/google/uuid v1.6.0
	github.com/moby/sys/mountinfo v0.7.2
	github.com/moby/sys/reexec v0.1.0
	github.com/moby/sys/symlink v0.3.0
	github.com/opencontainers/cgroups v0.0.6
	github.com/opencontainers/runc v1.4.0
	github.com/opencontainers/runtime-spec v1.3.0
	github.com/pelletier/go-toml v1.9.5
	github.com/prometheus/procfs v0.19.2
	github.com/sirupsen/logrus v1.9.4
	github.com/stretchr/testify v1.11.1
	github.com/urfave/cli-altsrc/v3 v3.1.0
	github.com/urfave/cli/v3 v3.6.2
	golang.org/x/mod v0.33.0
	golang.org/x/sys v0.41.0
	tags.cncf.io/container-device-interface v1.1.0
	tags.cncf.io/container-device-interface/specs-go v1.1.0
)

require (
	cyphar.com/go-pathrs v0.2.1 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/ttrpc v1.2.7 // indirect
	github.com/cyphar/filepath-securejoin v0.6.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/knqyf263/go-plugin v0.9.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/moby/sys/capability v0.4.0 // indirect
	github.com/opencontainers/runtime-tools v0.9.1-0.20251114084447-edf4cb3d2116 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.11.0 // indirect
	github.com/tetratelabs/wazero v1.10.1 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230731190214-cbb8c96f2d6d // indirect
	google.golang.org/grpc v1.57.1 // indirect
	google.golang.org/protobuf v1.36.8 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)

replace tags.cncf.io/container-device-interface => ../container-device-interface
