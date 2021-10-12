# NVIDIA Container Toolkit Release Tooling

This repository allows for the components of the NVIDIA container stack to be
built and released as the NVIDIA Container Toolkit from a single repository. The components:
* `libnvidia-container`
* `nvidia-container-runtime`
* `nvidia-docker`
are included as submodules in the `third_party` folder.

The `nvidia-container-toolkit` resides in this repo directly.

## Building

In oder to build the packages, the following command is executed
```sh
./scripts/build-all-components.sh TARGET
```
where `TARGET` is a make target that is valid for each of the sub-components.

These include:
* `ubuntu18.04-amd64`
* `centos8-x86_64`

The packages are generated in the `dist` folder.

## Testing local changes

In oder to use the same build logic to be used to generate packages with local changes,
the location of the individual components can be overridded using the: `LIBNVIDIA_CONTAINER_ROOT`,
`NVIDIA_CONTAINER_TOOLKIT_ROOT`, `NVIDIA_CONTAINER_RUNTIME_ROOT`, and `NVIDIA_DOCKER_ROOT`
environment variables.

## Testing packages locally

The [test/release](./test/release/) folder contains documentation on how the installation of local or staged packages can be tested.


## Releasing

A utility script [`scripts/release.sh`](./scripts/release.sh) is provided to build
packages required for release. If run without arguments, all supported distribution-architecture combinations are built. A specific distribution-architecture pair can also be provided
```sh
./scripts/release.sh ubuntu18.04-amd64
```
where the `amd64` builds for `ubuntu18.04` are provided as an example.
