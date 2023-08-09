#!/usr/bin/env bash

# Copyright (c) 2021, NVIDIA CORPORATION.  All rights reserved.
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

# This script is used to build the packages for the components of the NVIDIA
# Container Stack. These include the nvidia-container-toolkit in this repository
# as well as the components included in the third_party folder.
# All required packages are generated in the specified dist folder.

function assert_usage() {
    echo "Missing argument $1"
    echo "$(basename "${BASH_SOURCE[0]}") TARGET"
    exit 1
}

set -e

SCRIPTS_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../scripts && pwd )"
PROJECT_ROOT="$( cd "${SCRIPTS_DIR}"/.. && pwd )"

if [[ $# -ne 1 ]]; then
    assert_usage "TARGET"
fi

TARGET=$1

source "${SCRIPTS_DIR}"/utils.sh

: "${DIST_DIR:=${PROJECT_ROOT}/dist}"
export DIST_DIR

echo "Building ${TARGET} for all packages to ${DIST_DIR}"

: "${LIBNVIDIA_CONTAINER_ROOT:=${PROJECT_ROOT}/third_party/libnvidia-container}"
: "${NVIDIA_CONTAINER_TOOLKIT_ROOT:=${PROJECT_ROOT}}"
: "${NVIDIA_CONTAINER_RUNTIME_ROOT:=${PROJECT_ROOT}/third_party/nvidia-container-runtime}"
: "${NVIDIA_DOCKER_ROOT:=${PROJECT_ROOT}/third_party/nvidia-docker}"


"${SCRIPTS_DIR}/get-component-versions.sh"

if [[ -z "${NVIDIA_CONTAINER_TOOLKIT_VERSION}" || -z "${LIBNVIDIA_CONTAINER_VERSION}" ]]; then
eval $(${SCRIPTS_DIR}/get-component-versions.sh)
fi

# Build libnvidia-container
if [[ -z ${SKIP_LIBNVIDIA_CONTAINER} ]]; then
    make -C "${LIBNVIDIA_CONTAINER_ROOT}" -f mk/docker.mk \
        LIB_VERSION=${NVIDIA_CONTAINER_TOOLKIT_VERSION} \
        LIB_TAG=${NVIDIA_CONTAINER_TOOLKIT_TAG} \
        "${TARGET}"
fi

if [[ -z ${SKIP_NVIDIA_CONTAINER_TOOLKIT} ]]; then
# Build nvidia-container-toolkit
make -C "${NVIDIA_CONTAINER_TOOLKIT_ROOT}" \
    LIBNVIDIA_CONTAINER_VERSION="${NVIDIA_CONTAINER_TOOLKIT_VERSION}" \
    LIBNVIDIA_CONTAINER_TAG="${NVIDIA_CONTAINER_TOOLKIT_TAG}" \
        "${TARGET}"
fi

# If required we also build the nvidia-container-runtime and nvidia-docker packages.
# Since these are essentially meta packages intended to allow for users to
# transition from older installation workflows, we skip these for rc builds
# (NVIDIA_CONTAINER_TOOLKIT_TAG != "") and releases with a non-zero patch
# version of 0.
if [[ -n ${FORCE_META_PACKAGES} || -z ${NVIDIA_CONTAINER_TOOLKIT_TAG} && "${NVIDIA_CONTAINER_TOOLKIT_VERSION%.0}" != "${NVIDIA_CONTAINER_TOOLKIT_VERSION}" ]]; then
    package_format=$(package_type ${TARGET})
    package_target=$(get_package_target ${TARGET})

    # We set the TOOLKIT_VERSION, TOOLKIT_TAG for the nvidia-container-runtime and nvidia-docker targets
    # The LIB_TAG is also overridden to match the TOOLKIT_TAG.

    # Build nvidia-container-runtime if required
    package_name="nvidia-container-runtime"
    package_version=${NVIDIA_CONTAINER_RUNTIME_VERSION}${NVIDIA_CONTAINER_TOOLKIT_TAG:+~${NVIDIA_CONTAINER_TOOLKIT_TAG}}-1
    package_pattern=${DIST_DIR}/${package_format}/all/${package_name}?${package_version}?*.${package_format}
    package=$(ls ${package_pattern}) || echo ""
    if [[ -z ${package} ]]; then
        echo "${package_name} does not exist"
        make -C ${NVIDIA_CONTAINER_RUNTIME_ROOT} \
            LIB_VERSION="${NVIDIA_CONTAINER_RUNTIME_VERSION}" \
            LIB_TAG="${NVIDIA_CONTAINER_TOOLKIT_TAG}" \
            TOOLKIT_VERSION="${NVIDIA_CONTAINER_TOOLKIT_VERSION}" \
            TOOLKIT_TAG="${NVIDIA_CONTAINER_TOOLKIT_TAG}" \
                ${TARGET}
    fi
    if [[ -n ${package_target} ]]; then
        mkdir -p ${DIST_DIR}/${package_target}/
        cp -p ${package_pattern} ${DIST_DIR}/${package_target}/
    fi

    # Build nvidia-docker2 if required
    package_name="nvidia-docker2"
    package_version=${NVIDIA_DOCKER_VERSION}${NVIDIA_CONTAINER_TOOLKIT_TAG:+~${NVIDIA_CONTAINER_TOOLKIT_TAG}}-1
    package_pattern=${DIST_DIR}/${package_format}/all/${package_name}?${package_version}?*.${package_format}
    package=$(ls ${package_pattern}) || echo ""
    if [[ -z ${package} ]]; then
        echo "${package_name} does not exist"
        make -C ${NVIDIA_DOCKER_ROOT} \
            LIB_VERSION="${NVIDIA_DOCKER_VERSION}" \
            LIB_TAG="${NVIDIA_CONTAINER_TOOLKIT_TAG}" \
            TOOLKIT_VERSION="${NVIDIA_CONTAINER_TOOLKIT_VERSION}" \
            TOOLKIT_TAG="${NVIDIA_CONTAINER_TOOLKIT_TAG}" \
                ${TARGET}
    fi
    if [[ -n ${package_target} ]]; then
        mkdir -p ${DIST_DIR}/${package_target}/
        cp -p ${package_pattern} ${DIST_DIR}/${package_target}/
    fi

else
    echo "Skipping nvidia-container-runtime and nvidia-docker builds."
fi
