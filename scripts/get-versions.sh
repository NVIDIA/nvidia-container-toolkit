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
    exit 1
}

set -e

SCRIPTS_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../scripts && pwd )"
PROJECT_ROOT="$( cd ${SCRIPTS_DIR}/.. && pwd )"

: ${LIBNVIDIA_CONTAINER_ROOT:=${PROJECT_ROOT}/third_party/libnvidia-container}
: ${NVIDIA_CONTAINER_TOOLKIT_ROOT:=${PROJECT_ROOT}}
: ${NVIDIA_CONTAINER_RUNTIME_ROOT:=${PROJECT_ROOT}/third_party/nvidia-container-runtime}
: ${NVIDIA_DOCKER_ROOT:=${PROJECT_ROOT}/third_party/nvidia-docker}

# Get version for libnvidia-container
libnvidia_container_version=$(grep "#define NVC_VERSION" ${LIBNVIDIA_CONTAINER_ROOT}/src/nvc.h \
    | sed -e 's/#define NVC_VERSION[[:space:]]"\(.*\)"/\1/')

# Get version for nvidia-container-toolit
nvidia_container_toolkit_version=$(grep -m 1 "^LIB_VERSION := " ${NVIDIA_CONTAINER_TOOLKIT_ROOT}/Makefile | sed -e 's/LIB_VERSION :=[[:space:]]\(.*\)[[:space:]]*/\1/')
nvidia_container_toolkit_tag=$(grep -m 1 "^LIB_TAG .= " ${NVIDIA_CONTAINER_TOOLKIT_ROOT}/Makefile | sed -e 's/LIB_TAG .=[[:space:]]\(.*\)[[:space:]]*/\1/')
nvidia_container_toolkit_version="${nvidia_container_toolkit_version}${nvidia_container_toolkit_tag:+~${nvidia_container_toolkit_tag}}"

# Get version for nvidia-container-runtime
nvidia_container_runtime_version=$(grep -m 1 "^LIB_VERSION := " ${NVIDIA_CONTAINER_RUNTIME_ROOT}/Makefile | sed -e 's/LIB_VERSION :=[[:space:]]\(.*\)[[:space:]]*/\1/')
nvidia_container_runtime_tag=$(grep -m 1 "^LIB_TAG .= " ${NVIDIA_CONTAINER_RUNTIME_ROOT}/Makefile | sed -e 's/LIB_TAG .=[[:space:]]\(.*\)[[:space:]]*/\1/')
nvidia_container_runtime_version="${nvidia_container_runtime_version}${nvidia_container_runtime_tag:+~${nvidia_container_runtime_tag}}"

# Get version for nvidia-docker
nvidia_docker_version=$(grep -m 1 "^LIB_VERSION := " ${NVIDIA_DOCKER_ROOT}/Makefile | sed -e 's/LIB_VERSION :=[[:space:]]\(.*\)[[:space:]]*/\1/')
nvidia_docker_tag=$(grep -m 1 "^LIB_TAG .= " ${NVIDIA_DOCKER_ROOT}/Makefile | sed -e 's/LIB_TAG .=[[:space:]]\(.*\)[[:space:]]*/\1/')
nvidia_docker_version="${nvidia_docker_version}${nvidia_docker_tag:+~${nvidia_docker_tag}}"


echo "libnvidia-container version=${libnvidia_container_version}"
echo "nvidia-container-toolkit version=${nvidia_container_toolkit_version}"
echo "nvidia-container-runtime version=${nvidia_container_runtime_version}"
echo "nvidia-docker version=${nvidia_docker_version}"
