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
    echo "$(basename ${BASH_SOURCE[0]}) IMAGE DIST_DIR"
    exit 1
}

set -e -x

SCRIPTS_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../scripts && pwd )"
PROJECT_ROOT="$( cd ${SCRIPTS_DIR}/.. && pwd )"

if [[ $# -ne 2 ]]; then
    assert_usage $*
fi

IMAGE=$1
DIST_DIR=$2

if [[ -z ${IMAGE} ]]; then
    echo "ERROR: IMAGE must be non-empty"
    exit 1
fi

if [[ -z ${DIST_DIR} ]]; then
    echo "ERROR: DIST_DIR must be non-empty"
    exit 1
fi

if [[ -e ${DIST_DIR} ]]; then
    echo "ERROR: The specified DIST_DIR ${DIST_DIR} exists."
    exit 1
fi

echo "Copying package files from ${IMAGE} to ${DIST_DIR}"
mkdir -p ${DIST_DIR}
docker run --rm \
    -v $(pwd):$(pwd) \
    -w $(pwd) \
    -u $(id -u):$(id -g) \
    --entrypoint="bash" \
        ${IMAGE} \
        -c "cp -R /artifacts/packages/* ${DIST_DIR}"
