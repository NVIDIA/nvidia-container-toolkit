# The Experimental NVIDIA Container Runtime

## Introduction

The experimental NVIDIA Container Runtime is a proof-of-concept runtime that
approaches the problem of making GPUs (or other NVIDIA devices) available in
containerized environments in a different manner to the existing
[NVIDIA Container Runtime](../nvidia-container-runtime). Wherease the current
runtime relies on the [NVIDIA Container Library](https://github.com/NVIDIA/libnvidia-container)
to perform the modifications to a container, the experiemental runtime aims to
express the required modifications in terms of changes to a container's [OCI
runtime specification](https://github.com/opencontainers/runtime-spec). This
also aligns with open initiatives such as the [Container Device Interface (CDI)](https://github.com/container-orchestrated-devices/container-device-interface).

## Known Limitations

* The path of NVIDIA CUDA libraries / binaries injected into the container currently match that of the host system. This means that on an Ubuntu-based host systems these would be at `/usr/lib/x86_64-linux-gnu`
even if the container distribution would normally expect these at another location (e.g. `/usr/lib64`)
* Tools such as `nvidia-smi` may create additional device nodes in the container when run. This is
prevented in the "classic" runtime (and the NVIDIA Container Library) by modifying
the `/proc/driver/nvidia/params` file in the container.
* Other `NVIDIA_*` environment variables (e.g. `NVIDIA_DRIVER_CAPABILITIES`) are
not considered to filter mounted libraries or binaries. This is equivalent to
always using `NVIDIA_DRIVER_CAPABILITIES=all`.

## Building / Installing

The experimental NVIDIA Container Runtime is a self-contained golang binary and
can thus be built from source, or installed directly using `go install`.

### From source

After cloning the `nvidia-container-toolkit` repository from [GitLab](https://gitlab.com/nvidia/container-toolkit/container-toolkit)
or from the read-only mirror on [GitHub](https://github.com/NVIDIA/nvidia-container-toolkit)
running the following make command in the repository root:
```bash
make cmd-nvidia-container-runtime.experimental
```
will create an executable file `nvidia-container-runtime.experimental` in the
root.

A dockerized target:
```bash
make docker-cmd-nvidia-container-runtime.experimental
```
will also create the executable file `nvidia-container-runtime.experimental`
without requiring the setup of a development environment (with the exception)
of having `make` and `docker` installed.

### Go install

The experimental NVIDIA Container Runtime can also be `go installed` by running
the following command:

```bash
go install github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-container-runtime.experimental@experimental
```
which will build and install the `nvidia-container-runtime.experimental`
executable in the `${GOPATH}/bin` folder.

## Using the Runtime

The experimental NVIDIA Container Runtime is intended as a drop-in replacement
for the "classic" NVIDIA Container Runtime. As such it is used in the same
way (with the exception of the known limitiations noted above).

In general terms, to use the experimental NVIDIA Container Runtime to launch a
container with GPU support, it should be inserted as a shim for the desired
low-level OCI-compliant runtime (e.g. `runc` or `crun`). How this is achieved
depends on how containers are being launched.

### Docker
In the case of `docker` for example, the runtime must be registered with the
Docker daemon. This can be done by modifying the `/etc/docker/daemon.json` file
to contain the following:
```json
{
    "runtimes": {
        "nvidia-experimental": {
                "path": "nvidia-container-runtime.experimental",
                "runtimeArgs": []
        }
    },
}
```
This can then be invoked from docker by including the `--runtime=nvidia-experimental`
option when executing a `docker run` command.

### Runc

If `runc` is being used to run a container directly substituting the `runc`
command for `nvidia-container-runtime.experimental` should be sufficient as
the latter will `exec` to `runc` once the required (in-place) modifications have
been made to the container's OCI spec (`config.json` file).

## Configuration

### Runtime Path
The experimental NVIDIA Container Runtime allows for the path to the low-level
runtime to be specified. This is done by setting the following option in the
`/etc/nvidia-container-runtime/config.toml` file or setting the
`NVIDIA_CONTAINER_RUNTIME_PATH` environment variable.

```toml
[nvidia-container-runtime.experimental]

runtime-path = "/path/to/low-level-runtime"
```
This path can be set to the path for `runc` or `crun` on a system and if it is
a relative path, the `PATH` is searched for a matching executable.

### Device Selection
In order to select a specific device, the experimental NVIDIA Container Runtime
mimics the behaviour of the "classic" runtime. That is to say that the values of
certain [environment variables](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/user-guide.html#environment-variables-oci-spec)
**in the container's OCI specification** control the behaviour of the runtime.

#### `NVIDIA_VISIBLE_DEVICES`
This variable controls which GPUs will be made accessible inside the container.

##### Possible values
* `0,1,2`, `GPU-fef8089b` …: a comma-separated list of GPU UUID(s) or index(es).
* `all`: all GPUs will be accessible, this is the default value in our container images.
* `none`: no GPU will be accessible, but driver capabilities will be enabled.
* `void` or *empty* or *unset*: `nvidia-container-runtime` will have the same behavior as `runc`.

**Note**: When running on a MIG capable device, the following values will also be available:
* `0:0,0:1,1:0`, `MIG-GPU-fef8089b/0/1` …: a comma-separated list of MIG Device UUID(s) or index(es).

Where the MIG device indices have the form `<GPU Device Index>:<MIG Device Index>` as seen in the example output:
```
$ nvidia-smi -L
GPU 0: Graphics Device (UUID: GPU-b8ea3855-276c-c9cb-b366-c6fa655957c5)
  MIG Device 0: (UUID: MIG-GPU-b8ea3855-276c-c9cb-b366-c6fa655957c5/1/0)
  MIG Device 1: (UUID: MIG-GPU-b8ea3855-276c-c9cb-b366-c6fa655957c5/1/1)
  MIG Device 2: (UUID: MIG-GPU-b8ea3855-276c-c9cb-b366-c6fa655957c5/11/0)
```

#### `NVIDIA_MIG_CONFIG_DEVICES`
This variable controls which of the visible GPUs can have their MIG
configuration managed from within the container. This includes enabling and
disabling MIG mode, creating and destroying GPU Instances and Compute
Instances, etc.

##### Possible values
* `all`: Allow all MIG-capable GPUs in the visible device list to have their
  MIG configurations managed.

**Note**:
* This feature is only available on MIG capable devices (e.g. the A100).
* To use this feature, the container must be started with `CAP_SYS_ADMIN` privileges.
* When not running as `root`, the container user must have read access to the
  `/proc/driver/nvidia/capabilities/mig/config` file on the host.

#### `NVIDIA_MIG_MONITOR_DEVICES`
This variable controls which of the visible GPUs can have aggregate information
about all of their MIG devices monitored from within the container. This
includes inspecting the aggregate memory usage, listing the aggregate running
processes, etc.

##### Possible values
* `all`: Allow all MIG-capable GPUs in the visible device list to have their
  MIG devices monitored.

**Note**:
* This feature is only available on MIG capable devices (e.g. the A100).
* To use this feature, the container must be started with `CAP_SYS_ADMIN` privileges.
* When not running as `root`, the container user must have read access to the
  `/proc/driver/nvidia/capabilities/mig/monitor` file on the host.

