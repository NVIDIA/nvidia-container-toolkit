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
./scripts/build-packages.sh TARGET
```
where `TARGET` is a make target that is valid for each of the sub-components.

These include:
* `ubuntu18.04-amd64`
* `centos8-x86_64`

If no `TARGET` is specified, all valid release targets are built.

The packages are generated in the `dist` folder.

## Testing local changes

In oder to use the same build logic to be used to generate packages with local changes,
the location of the individual components can be overridded using the: `LIBNVIDIA_CONTAINER_ROOT`,
`NVIDIA_CONTAINER_TOOLKIT_ROOT`, `NVIDIA_CONTAINER_RUNTIME_ROOT`, and `NVIDIA_DOCKER_ROOT`
environment variables.

## Testing packages locally

The [test/release](./test/release/) folder contains documentation on how the installation of local or staged packages can be tested.


## Releasing

In order to release packages required for a release, a utility script
[`scripts/release-packages.sh`](./scripts/release-packages.sh) is provided.
This script can be executed as follows:

```bash
GPG_LOCAL_USER="GPG_USER" \
MASTER_KEY_PATH=/path/to/gpg-master.key \
SUB_KEY_PATH=/path/to/gpg-subkey.key \
    ./scripts/release-packages.sh REPO PACKAGE_REPO_ROOT [REFERENCE]
```

Where `REPO` is one of `stable` or `experimental`, `PACKAGE_REPO_ROOT` is the local path to the `libnvidia-container` repository checked out to the `gh-pages` branch, and `REFERENCE` is the git SHA that is to be released. If reference is not specified `HEAD` is assumed.

This scripts performs the following basic functions:
* Pulls the package image defined by the `REFERENCE` git SHA from the staging registry,
* Copies the required packages to the package repository at `PACKAGE_REPO_ROOT/REPO`,
* Signs the packages using the specified GPG keys

While the last two are performed, commits are added to the package repository. These can be pushed to the relevant repository.

