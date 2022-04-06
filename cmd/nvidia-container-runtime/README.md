# The NVIDIA Container Runtime

The NVIDIA Container Runtime is a shim for OCI-compliant low-level runtimes such as [runc](https://github.com/opencontainers/runc). When a `create` command is detected, the incoming [OCI runtime specification](https://github.com/opencontainers/runtime-spec) is modified in place and the command is forwarded to the low-level runtime.

## Standard Mode

In the standard mode configuration, the NVIDIA Container Runtime adds a [`prestart` hook](https://github.com/opencontainers/runtime-spec/blob/master/config.md#prestart) to the incomming OCI specification that invokes the NVIDIA Container Runtime Hook for all containers created. This hook checks whether NVIDIA devices are requested and ensures GPU access is configured using the `nvidia-container-cli` from project [libnvidia-container](https://github.com/NVIDIA/libnvidia-container).

## Experimental Mode

The NVIDIA Container Runtime can be configured in an experimental mode by setting the following options in the runtime's `config.toml` file:

```toml
[nvidia-container-runtime]
experimental = true
```

When this setting is enabled, the modifications made to the OCI specification are controlled by the `nvidia-container-runtime.discover-mode` option, with the following mode supported:

* `"legacy"`: This mode mirrors the behaviour of the standard mode, inserting the NVIDIA Container Runtime Hook as a `prestart` hook into the container's OCI specification.

### Notes on using the docker CLI

The `docker` CLI supports the `--gpus` flag to select GPUs for inclusion in a container. Since specifying this flag inserts the same NVIDIA Container Runtime Hook into the OCI runtime specification. When experimental mode is activated, the NVIDIA Container Runtime detects the presence of the hook and raises an error. This requirement will be relaxed in the near future.
