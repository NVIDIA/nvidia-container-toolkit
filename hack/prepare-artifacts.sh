#!/bin/bash -e

# Copyright (c) 2023, NVIDIA CORPORATION.  All rights reserved.
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

set -o pipefail

# if arg1 is set, it will be used as the version number
if [ -z "$1" ]; then
  VERSION=$(awk -F= '/^VERSION/ { print $2 }' versions.mk | tr -d '[:space:]')
else
  VERSION=$1
fi

if [[ -z ${VERSION} ]]; then
  echo "VERSION must be specified"
  exit 1
fi

SHA=$(git rev-parse --short=8 ${VERSION})

IMAGE_NAME="ghcr.io/nvidia/container-toolkit"
IMAGE_TAG=${SHA}-packaging

REPO="experimental"
if [[ ${VERSION/rc./} == ${VERSION} ]]; then
    REPO="stable"
fi

PACKAGE_ROOT=release-${VERSION}-${REPO}

./hack/pull-packages.sh \
    ${IMAGE_NAME}:${IMAGE_TAG} \
    ${PACKAGE_ROOT}

PACKAGE_VERSION=${VERSION/-/\~}
PACKAGE_VERSION=${PACKAGE_VERSION#v}

tar -czvf ${PACKAGE_ROOT}/nvidia-container-toolkit-${VERSION}.deb.amd64.tar.gz ${PACKAGE_ROOT}/packages/ubuntu18.04/amd64/*_${PACKAGE_VERSION}-1_amd64.deb
tar -czvf ${PACKAGE_ROOT}/nvidia-container-toolkit-${VERSION}.deb.arm64.tar.gz ${PACKAGE_ROOT}/packages/ubuntu18.04/arm64/*_${PACKAGE_VERSION}-1_arm64.deb
tar -czvf ${PACKAGE_ROOT}/nvidia-container-toolkit-${VERSION}.rpm.aarch64.tar.gz ${PACKAGE_ROOT}/packages/centos7/aarch64/*-${PACKAGE_VERSION}-1.aarch64.rpm
tar -czvf ${PACKAGE_ROOT}/nvidia-container-toolkit-${VERSION}.rpm.x86_64.tar.gz ${PACKAGE_ROOT}/packages/centos7/x86_64/*-${PACKAGE_VERSION}-1.x86_64.rpm
