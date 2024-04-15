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

NVIDIA_CONTAINER_TOOLKIT_ROOT=${PROJECT_ROOT}

versions_makefile=${NVIDIA_CONTAINER_TOOLKIT_ROOT}/versions.mk
# Get version for nvidia-container-toolit
nvidia_container_toolkit_version=$(grep -m 1 "^LIB_VERSION := " ${versions_makefile} | sed -e 's/LIB_VERSION :=[[:space:]]\(.*\)[[:space:]]*/\1/')
nvidia_container_toolkit_tag=$(grep -m 1 "^LIB_TAG .= " ${versions_makefile} | sed -e 's/LIB_TAG .=[[:space:]]\(.*\)[[:space:]]*/\1/')
nvidia_container_toolkit_version_tag="${nvidia_container_toolkit_version}${nvidia_container_toolkit_tag:+~${nvidia_container_toolkit_tag}}"

echo "NVIDIA_CONTAINER_TOOLKIT_VERSION=${nvidia_container_toolkit_version}"
echo "NVIDIA_CONTAINER_TOOLKIT_TAG=${nvidia_container_toolkit_tag}"
echo "NVIDIA_CONTAINER_TOOLKIT_PACKAGE_VERSION=${nvidia_container_toolkit_version_tag//\~/-}"
