# NVIDIA Container Toolkit CLI

The NVIDIA Container Toolkit CLI `nvidia-ctk` provides a number of utilities that are useful for working with the NVIDIA Container Toolkit.

## Functionality

### Configure runtimes

The `runtime` command of the `nvidia-ctk` CLI provides a set of utilities to related to the configuration
and management of supported container engines.

For example, running the following command:
```bash
nvidia-ctk runtime configure --set-as-default
```
will ensure that the NVIDIA Container Runtime is added as the default runtime to the default container
engine.
