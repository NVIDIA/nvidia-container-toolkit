## Introduction

This repository contains tools that allow docker, containerd, or cri-o to be configured to use the NVIDIA Container Toolkit.

*Note*: These were copied from the [`container-config` repository](https://gitlab.com/nvidia/container-toolkit/container-config/-/tree/383587f766a55177ede0e39e3810a974043e503e) are being migrated to commands installed with the NVIDIA Container Toolkit.

These will be migrated into an upcoming `nvidia-ctk` CLI as required.

### Docker

After building the `docker` binary, run:
```bash
docker setup \
    --runtime-name NAME \
        /run/nvidia/toolkit
```

Configure the `nvidia-container-runtime` as a docker runtime named `NAME`. If the `--runtime-name` flag is not specified, this runtime would be called `nvidia`.

Since `--set-as-default` is enabled by default, the specified runtime name will also be set as the default docker runtime. This can be disabled by explicityly specifying `--set-as-default=false`.

The following table describes the behaviour for different `--runtime-name` and `--set-as-default` flag combinations.

| Flags                                                       | Installed Runtimes              | Default Runtime       |
|-------------------------------------------------------------|:--------------------------------|:----------------------|
| **NONE SPECIFIED**                                          | `nvidia`                        | `nvidia`              |
| `--runtime-name nvidia`                                     | `nvidia`                        | `nvidia`              |
| `--runtime-name NAME`                                       | `NAME`                          | `NAME`                |
| `--set-as-default`                                          | `nvidia`                        | `nvidia`              |
| `--set-as-default --runtime-name nvidia`                    | `nvidia`                        | `nvidia`              |
| `--set-as-default --runtime-name NAME`                      | `NAME`                          | `NAME`                |
| `--set-as-default=false`                                    | `nvidia`                        | **NOT SET**           |
| `--set-as-default=false --runtime-name NAME`                | `NAME`                          | **NOT SET**           |
| `--set-as-default=false --runtime-name nvidia`              | `nvidia`                        | **NOT SET**           |

These combinations also hold for the environment variables that map to the command line flags: `DOCKER_RUNTIME_NAME`, `DOCKER_SET_AS_DEFAULT`.

### Containerd
After running the `containerd` binary, run:
```bash
containerd setup \
    --runtime-class NAME \
        /run/nvidia/toolkit
```

Configure the `nvidia-container-runtime` as a runtime class named `NAME`. If the `--runtime-class` flag is not specified, this runtime would be called `nvidia`.

Adding the `--set-as-default` flag as follows:
```bash
containerd setup \
    --runtime-class NAME \
    --set-as-default \
        /run/nvidia/toolkit
```
will set the runtime class `NAME` (or `nvidia` if not specified) as the default runtime class.

The following table describes the behaviour for different `--runtime-class` and `--set-as-default` flag combinations.

| Flags                                                  | Installed Runtime Classes       | Default Runtime Class |
|--------------------------------------------------------|:--------------------------------|:----------------------|
| **NONE SPECIFIED**                                     | `nvidia`                        | **NOT SET**           |
| `--runtime-class NAME`                                 | `NAME`                          | **NOT SET**           |
| `--runtime-class nvidia`                               | `nvidia`                        | **NOT SET**           |
| `--set-as-default`                                     | `nvidia`                        | `nvidia`              |
| `--set-as-default --runtime-class NAME`                | `NAME`                          | `NAME`                |
| `--set-as-default --runtime-class nvidia`              | `nvidia`                        | `nvidia`              |

These combinations also hold for the environment variables that map to the command line flags.
