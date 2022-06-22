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

set -e

SCRIPTS_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../scripts && pwd )"
PROJECT_ROOT="$( cd ${SCRIPTS_DIR}/.. && pwd )"

# This list represents the distribution-architecture pairs that are actually published
# to the relevant repositories. This targets forwarded to the build-all-components script
# can be overridden by specifying command line arguments.
all=(
    amazonlinux2-aarch64
    amazonlinux2-x86_64
    centos7-ppc64le
    centos7-x86_64
    centos8-aarch64
    centos8-ppc64le
    centos8-x86_64
    debian10-amd64
    debian9-amd64
    opensuse-leap15.1-x86_64
    ubuntu16.04-amd64
    ubuntu16.04-ppc64le
    ubuntu18.04-amd64
    ubuntu18.04-arm64
    ubuntu18.04-ppc64le
)

if [[ $# -gt 0 ]]; then
    targets=($*)
else
    targets=${all[@]}
fi

echo "Updating components"
${SCRIPTS_DIR}/update-components.sh
if [[ -n $(git status -s third_party) && ${ALLOW_LOCAL_COMPONENT_CHANGES} != "true" ]]; then
    echo "ERROR: Building with local component changes."
    echo "Commit pending changes or rerun with ALLOW_LOCAL_COMPONENT_CHANGES='true'"
    exit 1
fi

eval $(${SCRIPTS_DIR}/get-component-versions.sh)


if [[ -n ${NVIDIA_CONTAINER_TOOLKIT_TAG} ]]; then
echo "Allowing mismatched versions for release candidate "
: ${ALLOW_VERSION_MISMATCH:=true}
fi

if [[ "${NVIDIA_CONTAINER_TOOLKIT_PACKAGE_VERSION}" != "${LIBNVIDIA_CONTAINER_PACKAGE_VERSION}" ]]; then
    set +x
    echo "The libnvidia-container and nvidia-container-toolkit versions do not match."
    echo "lib: '${LIBNVIDIA_CONTAINER_PACKAGE_VERSION}'"
    echo "toolkit: '${NVIDIA_CONTAINER_TOOLKIT_PACKAGE_VERSION}'"
    set -x
    [[ ${ALLOW_VERSION_MISMATCH} == "true" ]] || exit 1
    echo "Continuing with mismatched version"
fi

export NVIDIA_CONTAINER_TOOLKIT_VERSION
export NVIDIA_CONTAINER_TOOLKIT_TAG
export LIBNVIDIA_CONTAINER_VERSION
export LIBNVIDIA_CONTAINER_TAG
export NVIDIA_CONTAINER_RUNTIME_VERSION
export NVIDIA_DOCKER_VERSION

for target in ${targets[@]}; do
    ${SCRIPTS_DIR}/build-all-components.sh ${target}
done
