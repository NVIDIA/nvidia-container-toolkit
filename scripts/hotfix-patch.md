# Patching a release

```
source ./scripts/utils.sh

export REFERENCE=main
```

```
SHA=$(git rev-parse --short=8 ${REFERENCE})
IMAGE_NAME="ghcr.io/nvidia/container-toolkit"
IMAGE_TAG=${SHA}-packaging
VERSION=$(get_version_from_image ${IMAGE_NAME}:${IMAGE_TAG} ${SHA})
PACKAGE_CACHE=release-${VERSION}-patch-$(git rev-parse --short=8 HEAD)
```

```
./hack/pull-packages.sh \
    ${IMAGE_NAME}:${IMAGE_TAG} \
    ${PACKAGE_CACHE}
```




1. Build the required components:

```
export DIST_DIR=$(pwd)/${PACKAGE_CACHE}/packages
./scripts/build-packages.sh ${TARGETS}
```

Where `${TARGETS}` is any combination of:
- `ubuntu18.04-arm64`
- `ubuntu18.04-amd64`
- `ubuntu18.04-ppc64le`
- `centos7-aarch64`
- `centos7-x86_64`
- `centos8-ppc64le`

The `ppc64le` targets are generally not covered by QA, and something like:
```
./scripts/build-packages.sh ubuntu18.04-amd64 ubuntu18.04-arm64 centos7-x86_64 centos7-aarch64
```

Should be sufficient.

1. Patch the container-toolkit images:
    1. The packaging image:
```
BUILD_MULTI_ARCH_IMAGES=true \
    ARTIFACTS_ROOT=${PACKAGE_CACHE}/packages \
    VERSION=$(git rev-parse --short=8 HEAD) \
    make -f deployments/container/Makefile build-packaging
```

The other images:
```
BUILD_MULTI_ARCH_IMAGES=true \
    ARTIFACTS_ROOT=${PACKAGE_CACHE}/packages \
    VERSION=$(git rev-parse --short=8 HEAD) \
    make -f deployments/container/Makefile build-ubuntu20.04
```

```
BUILD_MULTI_ARCH_IMAGES=true \
    ARTIFACTS_ROOT=${PACKAGE_CACHE}/packages \
    VERSION=$(git rev-parse --short=8 HEAD) \
    make -f deployments/container/Makefile build-ubi8
```

Note that even though the other packages were not updated we still regenearate
both images so as to ensure consistent image versioning.


1. Push the updated packages to the kitmaker repository:

```
export ARTIFACTS_DIR=$(pwd)/release-${VERSION}-patch-$(git rev-parse --short=8 HEAD)-artifacts
```

```
./scripts/extract-packages.sh nvidia/container-toolkit:$(git rev-parse --short=8 HEAD)-packaging
```


```
./scripts/release-kitmaker-artifactory.sh \
    "https://urm.nvidia.com/artifactory/sw-gpu-cloudnative-generic-local/kitmaker"
```

```
regctl login nvcr.io -u \$oauthtoken
```

```
BUILD_MULTI_ARCH_IMAGES=true \
    ARTIFACTS_ROOT=${PACKAGE_CACHE}/packages \
    VERSION=$(git rev-parse --short=8 HEAD) \
    OUT_IMAGE_NAME=nvcr.io/ea-cnt/nv_only/container-toolkit \
    make -f deployments/container/Makefile push-ubuntu20.04
```

