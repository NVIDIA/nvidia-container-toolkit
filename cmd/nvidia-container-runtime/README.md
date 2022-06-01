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
