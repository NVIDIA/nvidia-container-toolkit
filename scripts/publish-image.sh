#!/usr/bin/env bash

# Copyright (c) NVIDIA CORPORATION.  All rights reserved.
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

# This script is used to publish images to a registry. It checks whether the image
# already exists in the registry and skips publishing if it does. This can be overridden
# using the FORCE_PUBLISH_IMAGES environment variable.

set -ex

: ${DOCKER=docker}
: ${REGCTL=regctl}

INPUT_IMAGE=$1
OUTPUT_IMAGE=$2

function publish_docker() {
    ${DOCKER} tag ${1} ${2}
    ${DOCKER} push ${2}
}

function publish_regctl() {
    ${REGCTL} image copy ${1} ${2}
}

function publish() {
    if [[ x"${BUILD_MULTI_ARCH_IMAGES}" == x"true" || $(command -v ${REGCTL}) ]]; then
        publish_regctl $@
    else
        publish_docker $@
    fi
}

# image_exists returns 0 if the image exists in a registry and a non-zero return
# code if this is not the case.
function image_exists() {
    local image=$1
    ${DOCKER} manifest inspect ${image}
}

if [[ -z ${FORCE_PUBLISH_IMAGES} && $(image_exists ${OUTPUT_IMAGE}) ]]; then
    echo "Skipping publishing of ${INPUT_IMAGE} as ${OUTPUT_IMAGE} already exists"
    exit 0
fi

echo "Publishing ${INPUT_IMAGE} as ${OUTPUT_IMAGE}"
publish ${INPUT_IMAGE} ${OUTPUT_IMAGE}
