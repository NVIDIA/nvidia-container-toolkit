#!/bin/bash

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

function assert_usage() {
    echo "Incorrect arguments: $*"
    echo "$(basename ${BASH_SOURCE[0]}) ROOT PACKAGE_NAME BUILD_VERSION"
    exit 1
}

set -e -x

SCRIPTS_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../scripts && pwd )"
PROJECT_ROOT="$( cd ${SCRIPTS_DIR}/.. && pwd )"

if [[ $# -ne 0 ]]; then
    assert_usage $*
fi

function pull_package_files() {
    local image=$1
    local dist_dir=$2
    echo "Copying package files from ${image} to ${dist_dir}"
    mkdir -p ${dist_dir}
    docker run \
        --rm \
        --entrypoint="bash" \
        -v $(pwd):$(pwd) \
        -w $(pwd) \
        -u $(id -u):$(id -g) \
            ${image} \
            -c "cp -R /artifacts/packages/* ${dist_dir}"
}

if [[ -z ${VERSION} ]]; then
eval $(${SCRIPTS_DIR}/get-component-versions.sh)
VERSION=${NVIDIA_CONTAINER_TOOLKIT_VERSION}${NVIDIA_CONTAINER_TOOLKIT_TAG:+-${NVIDIA_CONTAINER_TOOLKIT_TAG}}
fi

if [[ -z ${IMAGE} ]]; then
: ${REGISTRY:=""}
: ${IMAGE_NAME:="nvidia/container-toolkit"}
image_tag=${VERSION}-packaging
IMAGE=${REGISTRY:+${REGISTRY}/}${IMAGE_NAME}:${image_tag}
fi
: ${DIST_DIR:="dist-test"}

pull_package_files ${IMAGE} ${DIST_DIR}
