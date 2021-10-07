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
* `docker-all`
with the later generating for all supported distribution and platform combinations.

The packages are generated in the `dist` folder.

## Testing local changes

In oder to use the same build logic to be used to generate packages with local changes,
the location of the individual components can be overridded using the: `LIBNVIDIA_CONTAINER_ROOT`,
`NVIDIA_CONTAINER_TOOLKIT_ROOT`, `NVIDIA_CONTAINER_RUNTIME_ROOT`, and `NVIDIA_DOCKER_ROOT`
environment variables.

## Testing packages locally

### Ubuntu

Launch a docker container:

```
docker run --rm -it \
    -v $(pwd):/work \
    -v $(pwd)/dist/ubuntu18.04/amd64:/local-repository \
    -w /work \
        ubuntu:18.04
```


```
apt-get update && apt-get install -y apt-utils
```

```
echo "deb [trusted=yes] file:/local-repository/ ./" > /etc/apt/sources.list.d/local.list
```

```
cd /local-repository && apt-ftparchive packages . > Packages
```

```
apt-get update
```



### Centos

```
docker run --rm -it \
    -v $(pwd):/work \
    -v $(pwd)/dist/centos8/x86_64:/local-repository \
    -w /work \
        centos:8
```

```
yum install -y createrepo
```

```
createrepo /local-repository
```

```
cat >/etc/yum.repos.d/local.repo <<EOL
[local]
name=NVIDIA Container Toolkit Local Packages
baseurl=file:///local-repository
enabled=1
gpgcheck=0
protect=1
EOL
```
