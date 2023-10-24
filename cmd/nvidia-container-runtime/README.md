# The NVIDIA Container Runtime

The NVIDIA Container Runtime is a shim for OCI-compliant low-level runtimes such as [runc](https://github.com/opencontainers/runc). When a `create` command is detected, the incoming [OCI runtime specification](https://github.com/opencontainers/runtime-spec) is modified in place and the command is forwarded to the low-level runtime.

## Configuration

The NVIDIA Container Runtime uses file-based configuration, with the config stored in `/etc/nvidia-container-runtime/config.toml`. The `/etc` path can be overridden using the `XDG_CONFIG_HOME` environment variable with the `${XDG_CONFIG_HOME}/nvidia-container-runtime/config.toml` file used instead if this environment variable is set.

This config file may contain options for other components of the NVIDIA container stack and for the NVIDIA Container Runtime, the relevant config section is `nvidia-container-runtime`

### Logging

The `log-level` config option (default: `"info"`) specifies the log level to use and the `debug` option, if set, specifies a log file to which logs for the NVIDIA Container Runtime must be written.

In addition to this, the NVIDIA Container Runtime considers the value of `--log` and `--log-format` flags that may be passed to it by a container runtime such as docker or containerd. If the `--debug` flag is present the log-level specified in the config file is overridden as `"debug"`.

### Low-level Runtime Path

The `runtimes` config option allows for the low-level runtime to be specified. The first entry in this list that is an existing executable file is used as the low-level runtime. If the entry is not a path, the `PATH` is searched for a matching executable. If the entry is a path this is checked instead.

The default value for this setting is:
```toml
runtimes = [
    "docker-runc",
    "runc",
]
```

and if, for example, `crun` is to be used instead this can be changed to:
```toml
runtimes = [
    "crun",
]
```

### Runtime Mode

The `mode` config option (default `"auto"`) controls the high-level behaviour of the runtime.

#### Auto Mode

When `mode` is set to `"auto"`, the runtime employs heuristics to determine which mode to use based on, for example, the platform where the runtime is being run.

#### Legacy Mode

When `mode` is set to `"legacy"`, the NVIDIA Container Runtime adds a [`prestart` hook](https://github.com/opencontainers/runtime-spec/blob/master/config.md#prestart) to the incomming OCI specification that invokes the NVIDIA Container Runtime Hook for all containers created. This hook checks whether NVIDIA devices are requested and ensures GPU access is configured using the `nvidia-container-cli` from the [libnvidia-container](https://github.com/NVIDIA/libnvidia-container) project.

#### CSV Mode

When `mode` is set to `"csv"`, CSV files at `/etc/nvidia-container-runtime/host-files-for-container.d` define the devices and mounts that are to be injected into a container when it is created. The search path for the files can be overridden by modifying the `nvidia-container-runtime.modes.csv.mount-spec-path` in the config as below:

```toml
[nvidia-container-runtime]
    [nvidia-container-runtime.modes.csv]
    mount-spec-path = "/etc/nvidia-container-runtime/host-files-for-container.d"
```

This mode is primarily targeted at Tegra-based systems without NVML available.

### Notes on using the docker CLI

Note that only the `"legacy"` NVIDIA Container Runtime mode is directly compatible with the `--gpus` flag implemented by the `docker` CLI (assuming the NVIDIA Container Runtime is not used). The reason for this is that `docker` inserts the same NVIDIA Container Runtime Hook into the OCI runtime specification.


If a different mode is explicitly set or detected, the NVIDIA Container Runtime Hook will raise the following error when `--gpus` is set:
```
$ docker run --rm --gpus all ubuntu:18.04
docker: Error response from daemon: failed to create shim: OCI runtime create failed: container_linux.go:380: starting container process caused: process_linux.go:545: container init caused: Running hook #0:: error running hook: exit status 1, stdout: , stderr: Auto-detected mode as 'csv'
invoking the NVIDIA Container Runtime Hook directly (e.g. specifying the docker --gpus flag) is not supported. Please use the NVIDIA Container Runtime instead.: unknown.
```
Here NVIDIA Container Runtime must be used explicitly. The recommended way to do this is to specify the `--runtime=nvidia` command line argument as part of the `docker run` commmand as follows:
```
$ docker run --rm --gpus all --runtime=nvidia ubuntu:18.04
```

Alternatively the NVIDIA Container Runtime can be set as the default runtime for docker. This can be done by modifying the `/etc/docker/daemon.json` file as follows:
```json
{
    "default-runtime": "nvidia",
    "runtimes": {
        "nvidia": {
            "path": "nvidia-container-runtime",
            "runtimeArgs": []
        }
    }
}
```

## Environment variables (OCI spec)

Each environment variable maps to an command-line argument for `nvidia-container-cli` from [libnvidia-container](https://github.com/NVIDIA/libnvidia-container).
These variables are already set in our [official CUDA images](https://hub.docker.com/r/nvidia/cuda/).

### `NVIDIA_VISIBLE_DEVICES`
This variable controls which GPUs will be made accessible inside the container.

#### Possible values
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

### `NVIDIA_MIG_CONFIG_DEVICES`
This variable controls which of the visible GPUs can have their MIG
configuration managed from within the container. This includes enabling and
disabling MIG mode, creating and destroying GPU Instances and Compute
Instances, etc.

#### Possible values
* `all`: Allow all MIG-capable GPUs in the visible device list to have their
  MIG configurations managed.

**Note**:
* This feature is only available on MIG capable devices (e.g. the A100).
* To use this feature, the container must be started with `CAP_SYS_ADMIN` privileges.
* When not running as `root`, the container user must have read access to the
  `/proc/driver/nvidia/capabilities/mig/config` file on the host.

### `NVIDIA_MIG_MONITOR_DEVICES`
This variable controls which of the visible GPUs can have aggregate information
about all of their MIG devices monitored from within the container. This
includes inspecting the aggregate memory usage, listing the aggregate running
processes, etc.

#### Possible values
* `all`: Allow all MIG-capable GPUs in the visible device list to have their
  MIG devices monitored.

**Note**:
* This feature is only available on MIG capable devices (e.g. the A100).
* To use this feature, the container must be started with `CAP_SYS_ADMIN` privileges.
* When not running as `root`, the container user must have read access to the
  `/proc/driver/nvidia/capabilities/mig/monitor` file on the host.

### `NVIDIA_DRIVER_CAPABILITIES`
This option controls which driver libraries/binaries will be mounted inside the container.

#### Possible values
* `compute,video`, `graphics,utility` …: a comma-separated list of driver features the container needs.
* `all`: enable all available driver capabilities.
* *empty* or *unset*: use default driver capability: `utility,compute`.

#### Supported driver capabilities
* `compute`: required for CUDA and OpenCL applications.
* `compat32`: required for running 32-bit applications.
* `graphics`: required for running OpenGL and Vulkan applications.
* `utility`: required for using `nvidia-smi` and NVML.
* `video`: required for using the Video Codec SDK.
* `display`: required for leveraging X11 display.

### `NVIDIA_REQUIRE_*`
A logical expression to define constraints on the configurations supported by the container.

#### Supported constraints
* `cuda`: constraint on the CUDA driver version.
* `driver`: constraint on the driver version.
* `arch`: constraint on the compute architectures of the selected GPUs.
* `brand`: constraint on the brand of the selected GPUs (e.g. GeForce, Tesla, GRID).

#### Expressions
Multiple constraints can be expressed in a single environment variable: space-separated constraints are ORed, comma-separated constraints are ANDed.
Multiple environment variables of the form `NVIDIA_REQUIRE_*` are ANDed together.

### `NVIDIA_DISABLE_REQUIRE`
Single switch to disable all the constraints of the form `NVIDIA_REQUIRE_*`.

### `NVIDIA_REQUIRE_CUDA`

The version of the CUDA toolkit used by the container. It is an instance of the generic `NVIDIA_REQUIRE_*` case and it is set by official CUDA images.
If the version of the NVIDIA driver is insufficient to run this version of CUDA, the container will not be started.

#### Possible values
* `cuda>=7.5`, `cuda>=8.0`, `cuda>=9.0` …: any valid CUDA version in the form `major.minor`.

### `CUDA_VERSION`
Similar to `NVIDIA_REQUIRE_CUDA`, for legacy CUDA images.
In addition, if `NVIDIA_REQUIRE_CUDA` is not set, `NVIDIA_VISIBLE_DEVICES` and `NVIDIA_DRIVER_CAPABILITIES` will default to `all`.

## Usage example

**NOTE:** The use of the `nvidia-container-runtime` as CLI replacement for `runc` is uncommon and is only provided for completeness.

Although the `nvidia-container-runtime` is typically configured as a replacement for `runc` or `crun` in various container engines, it can also be
invoked from the command line as `runc` would. For example:

```sh
# Setup a rootfs based on Ubuntu 16.04
cd $(mktemp -d) && mkdir rootfs
curl -sS http://cdimage.ubuntu.com/ubuntu-base/releases/16.04/release/ubuntu-base-16.04-core-amd64.tar.gz | tar --exclude 'dev/*' -C rootfs -xz

# Create an OCI runtime spec
nvidia-container-runtime spec
sed -i 's;"sh";"nvidia-smi";' config.json
sed -i 's;\("TERM=xterm"\);\1, "NVIDIA_VISIBLE_DEVICES=0";' config.json

# Run the container
sudo nvidia-container-runtime run nvidia_smi
```
