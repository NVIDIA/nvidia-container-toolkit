# Copyright (c) 2019-2021, NVIDIA CORPORATION.  All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

ARG GOLANG_VERSION=x.x.x
ARG VERSION="N/A"

FROM nvcr.io/nvidia/cuda:12.8.1-base-ubi8 AS build

RUN yum install -y \
    wget make git gcc \
     && \
    rm -rf /var/cache/yum/*

ARG GOLANG_VERSION=x.x.x
RUN set -eux; \
    \
    arch="$(uname -m)"; \
    case "${arch##*-}" in \
        x86_64 | amd64) ARCH='amd64' ;; \
        ppc64el | ppc64le) ARCH='ppc64le' ;; \
        aarch64 | arm64) ARCH='arm64' ;; \
        *) echo "unsupported architecture" ; exit 1 ;; \
    esac; \
    wget -nv -O - https://storage.googleapis.com/golang/go${GOLANG_VERSION}.linux-${ARCH}.tar.gz \
    | tar -C /usr/local -xz


ENV GOPATH=/go
ENV PATH=$GOPATH/bin:/usr/local/go/bin:$PATH

WORKDIR /build
COPY . .

RUN mkdir /artifacts
ARG VERSION="N/A"
ARG GIT_COMMIT="unknown"
RUN make PREFIX=/artifacts cmd-nvidia-ctk-installer

# The packaging stage collects the deb and rpm packages built for supported
# architectures.
FROM nvcr.io/nvidia/cuda:12.8.1-base-ubi8 AS packaging

ARG ARTIFACTS_ROOT
COPY ${ARTIFACTS_ROOT} /artifacts/packages/

WORKDIR /artifacts/packages

# build-args are added to the manifest.txt file below.
ARG PACKAGE_VERSION
ARG GIT_BRANCH
ARG GIT_COMMIT
ARG GIT_COMMIT_SHORT
ARG SOURCE_DATE_EPOCH
ARG VERSION

# Create a manifest.txt file with the absolute paths of all deb and rpm packages in the container
RUN echo "#IMAGE_EPOCH=$(date '+%s')" > /artifacts/manifest.txt && \
    env | sed 's/^/#/g' >> /artifacts/manifest.txt && \
    find /artifacts/packages -iname '*.deb' -o -iname '*.rpm' >> /artifacts/manifest.txt

RUN mkdir /licenses && mv /NGC-DL-CONTAINER-LICENSE /licenses/NGC-DL-CONTAINER-LICENSE

# The debpackages stage is used to extract the contents of deb packages.
FROM nvcr.io/nvidia/cuda:12.8.1-base-ubuntu20.04 AS debpackages

ARG TARGETARCH
ARG PACKAGE_DIST_DEB=ubuntu18.04

COPY --from=packaging /artifacts/packages/${PACKAGE_DIST_DEB} /deb-packages

RUN mkdir -p /artifacts/deb
RUN set -eux; \
    \
    case "${TARGETARCH}" in \
        x86_64 | amd64) ARCH='amd64' ;; \
        ppc64el | ppc64le) ARCH='ppc64le' ;; \
        aarch64 | arm64) ARCH='arm64' ;; \
        *) echo "unsupported architecture" ; exit 1 ;; \
    esac; \
    for p in $(ls /deb-packages/${ARCH}/*.deb); do dpkg-deb -xv $p /artifacts/deb/; done

# The rpmpackages stage is used to extract the contents of the rpm packages.
FROM nvcr.io/nvidia/cuda:12.8.1-base-ubi8 AS rpmpackages
RUN dnf install -y cpio

ARG TARGETARCH
ARG PACKAGE_DIST_RPM=centos7

COPY --from=packaging /artifacts/packages/${PACKAGE_DIST_RPM} /rpm-packages

RUN mkdir -p /artifacts/rpm
RUN set -eux; \
    \
    case "${TARGETARCH}" in \
        x86_64 | amd64) ARCH='x86_64' ;; \
        ppc64el | ppc64le) ARCH='ppc64le' ;; \
        aarch64 | arm64) ARCH='aarch64' ;; \
        *) echo "unsupported architecture" ; exit 1 ;; \
    esac; \
    for p in $(ls /rpm-packages/${ARCH}/*.rpm); do rpm2cpio $p | cpio -idmv -D /artifacts/rpm; done

# The artifacts image serves as an intermediate stage to collect the artifacts
# From the previous stages:
# - The extracted deb packages
# - The extracted rpm packages
# - The nvidia-ctk-installer binary
FROM nvcr.io/nvidia/cuda:12.8.1-base-ubi8 AS artifacts

COPY --from=rpmpackages /artifacts/rpm /artifacts/rpm
COPY --from=debpackages /artifacts/deb /artifacts/deb
COPY --from=build /artifacts/bin /artifacts/build

FROM nvcr.io/nvidia/cuda:12.8.1-base-ubi8

ENV NVIDIA_DISABLE_REQUIRE="true"
ENV NVIDIA_VISIBLE_DEVICES=void
ENV NVIDIA_DRIVER_CAPABILITIES=utility

COPY --from=artifacts /artifacts/rpm /artifacts/rpm
COPY --from=artifacts /artifacts/deb /artifacts/deb
COPY --from=artifacts /artifacts/build /work

WORKDIR /work
ENV PATH=/work:$PATH

ARG VERSION
LABEL io.k8s.display-name="NVIDIA Container Runtime Config"
LABEL name="NVIDIA Container Runtime Config"
LABEL vendor="NVIDIA"
LABEL version="${VERSION}"
LABEL release="N/A"
LABEL summary="Automatically Configure your Container Runtime for GPU support."
LABEL description="See summary"

RUN mkdir /licenses && mv /NGC-DL-CONTAINER-LICENSE /licenses/NGC-DL-CONTAINER-LICENSE

ENTRYPOINT ["/work/nvidia-ctk-installer"]
