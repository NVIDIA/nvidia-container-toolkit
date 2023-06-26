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


function assert_usage() {
    echo "Incorrect arguments: $*"
    echo "$(basename ${BASH_SOURCE[0]}) ARTIFACTORY_REPO [GIT_REFERENCE]"
    echo "  ARTIFACTORY_REPO: URL to Artifactory repository"
    echo "  GIT_REFERENCE: Git reference to use for the package version"
    echo "               (if not specified, PACKAGE_IMAGE_TAG must be set)"
    exit 1
}

set -e

SCRIPTS_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../scripts && pwd )"
PROJECT_ROOT="$( cd ${SCRIPTS_DIR}/.. && pwd )"

if [[ $# -lt 1 ]]; then
    assert_usage "$@"
fi

source "${SCRIPTS_DIR}"/utils.sh

ARTIFACTORY_REPO=$1

if [[ $# -eq 2 ]]; then
    REFERENCE=$2
    SHA=$(git rev-parse --short=8 ${REFERENCE})
elif [[ -z ${PACKAGE_IMAGE_TAG} ]]; then
    echo "Either PACKAGE_IMAGE_TAG or REFERENCE must be specified"
    assert_usage "$@"
fi

: ${CURL:=curl}

: ${PACKAGE_IMAGE_NAME="registry.gitlab.com/nvidia/container-toolkit/container-toolkit/staging/container-toolkit"}
: ${PACKAGE_IMAGE_TAG=${SHA}-packaging}

VERSION="$(get_version_from_image ${PACKAGE_IMAGE_NAME}:${PACKAGE_IMAGE_TAG} ${SHA})"

REPO="experimental"
if [[ ${VERSION/rc./} == ${VERSION} ]]; then
    REPO="stable"
fi

PACKAGE_CACHE=release-${VERSION}-${REPO}
REMOVE_PACKAGE_CACHE=no
if [ ! -d ${PACKAGE_CACHE} ]; then
echo "Fetching packages with SHA '${SHA}' as tag '${VERSION}' to ${PACKAGE_CACHE}"
${SCRIPTS_DIR}/pull-packages.sh \
    ${PACKAGE_IMAGE_NAME}:${PACKAGE_IMAGE_TAG} \
    ${PACKAGE_CACHE}
    REMOVE_PACKAGE_CACHE=yes
else
    echo "Using existing package cache: ${PACKAGE_CACHE}"
fi

ARTIFACTS_DIR=${PROJECT_ROOT}/${PACKAGE_CACHE}

IMAGE_EPOCH=$(extract_info "IMAGE_EPOCH")
# Note we use the main branch for the kitmaker archive.
GIT_BRANCH=main
GIT_COMMIT=$(extract_info "GIT_COMMIT")
GIT_COMMIT_SHORT=$(extract_info "GIT_COMMIT_SHORT")
PACKAGE_VERSION=$(extract_info "PACKAGE_VERSION")

tar -czvf ${PACKAGE_CACHE}.tar.gz ${PACKAGE_CACHE}

if [[ ${REMOVE_PACKAGE_CACHE} == "yes" ]]; then
    rm -rf ${PACKAGE_CACHE}
fi

: ${PACKAGE_ARCHIVE_FOLDER=releases-testing}

function upload_archive() {
    local archive=$1
    local component=$2
    local version=$3

    if [ ! -r "${archive}" ]; then
        echo "ERROR: File not found or not readable: ${archive}"
        exit 1
    fi
    local sha1_checksum=$(sha1sum -b "${archive}" | awk '{ print $1 }')

    local upload_url="${ARTIFACTORY_REPO}/${PACKAGE_ARCHIVE_FOLDER}/${component}/${version}/$(basename ${archive})"

    local props=()
    # Required KITMAKER properties:
    props+=("component_name=${component}")
    props+=("version=${version}")
    props+=("changelist=${GIT_COMMIT_SHORT}")
    props+=("branch=${GIT_BRANCH}")
    props+=("source=https://gitlab.com/nvidia/container-toolkit/container-toolkit")
    # Package properties:
    props+=("package.epoch=${IMAGE_EPOCH}")
    props+=("package.version=${PACKAGE_VERSION}")
    props+=("package.commit=${GIT_COMMIT}")

    for var in "CI_PROJECT_ID" "CI_PIPELINE_ID" "CI_JOB_ID" "CI_JOB_URL" "CI_PROJECT_PATH"; do
        if [ -n "${!var}" ]; then
            optionally_add_property "${var}" "${!var}"
        fi
    done
    local PROPS=$(join_by ";" "${props[@]}")

    echo "Uploading ${upload_url} from ${archive}"
    echo -H "X-JFrog-Art-Api: REDACTED" \
        -H "X-Checksum-Sha1: ${sha1_checksum}" \
        ${archive:+-T ${archive}} -X PUT \
        "${upload_url};${PROPS}"
    if ! ${CURL} -f \
        -H "X-JFrog-Art-Api: ${ARTIFACTORY_TOKEN}" \
        -H "X-Checksum-Sha1: ${sha1_checksum}" \
        ${archive:+-T ${archive}} -X PUT \
        "${upload_url};${PROPS}" ;
    then
        echo "ERROR: upload file failed: ${archive}"
        exit 1
    fi
}

upload_archive "${PACKAGE_CACHE}.tar.gz" "nvidia_container_toolkit" "${VERSION}"

echo "Removing ${PACKAGE_CACHE}.tar.gz"
rm -f "${PACKAGE_CACHE}.tar.gz"

