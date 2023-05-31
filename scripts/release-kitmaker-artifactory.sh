#!/bin/bash

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
cat >&2 << EOF
Incorrect arguments: $*
$(basename "${BASH_SOURCE[0]}") KITMAKER_ARTIFACTORY_REPO
    KITMAKER_ARTIFACTORY_REPO must contain repo path for package, including hostname.

Environment Variables
    ARTIFACTORY_TOKEN: must contain an auth token. [required]
EOF
    exit 1
}

set -e

SCRIPTS_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../scripts && pwd )"
PROJECT_ROOT="$( cd "${SCRIPTS_DIR}/.." && pwd )"

if [[ $# -ne 1 ]]; then
    assert_usage "$@"
fi

source "${SCRIPTS_DIR}"/utils.sh

# KITMAKER_ARTIFACTORY_REPO=https://urm.nvidia.com/artifactory/sw-gpu-cloudnative-generic-local/testing
KITMAKER_ARTIFACTORY_REPO=$1

: ${CURL:=curl}

# ARTIFACTS_DIR represents the root of the artifacts (deb and rpm packages)
# extracted from the packaging image.
# TODO: accept ARTIFACTS_DIR as a command-line argument
: "${ARTIFACTS_DIR="${PROJECT_ROOT}/artifacts"}"

if [[ ! -d "${ARTIFACTS_DIR}" ]]; then
    echo "ERROR: ARTIFACTS_DIR does not exist." >&2
    assert_usage "$@"
fi

if [[ ! -f "${ARTIFACTS_DIR}/manifest.txt" ]]; then
    echo "ERROR: Manifest file not found." >&2
    assert_usage "$@"
fi

if [[ -z "${ARTIFACTORY_TOKEN}" ]]; then
    echo "ERROR: ARTIFACTORY_TOKEN must be defined." >&2
    assert_usage "$@"
fi

# TODO: accept KITMAKER_DIR as a command-line argument
: "${KITMAKER_DIR="${PROJECT_ROOT}/artifacts/kitmaker"}"

KITMAKER_SCRATCH="${KITMAKER_DIR}/.scratch"

IMAGE_EPOCH=$(extract_info "IMAGE_EPOCH")
GIT_COMMIT=$(extract_info "GIT_COMMIT")
GIT_COMMIT_SHORT=$(extract_info "GIT_COMMIT_SHORT")
VERSION=$(extract_info "PACKAGE_VERSION")


# add_distro adds the specified component, os, and arch to the .package folder from which a kitmaker archive is generated.
function add_distro() {
    local component=$1
    local branch=$2
    local os=$3
    local arch=$4

    local package_dist=$5
    local package_arch=$6

    local name="${component}-${os}-${arch}"

    local scratch_dir="${KITMAKER_SCRATCH}/${name}/${branch}/${component}"
    local packages_dir="${scratch_dir}/.packages"

    mkdir -p "${packages_dir}"

    # Copy the extracted files to the .packages directory so that a kitmaker file can be created.
    source="${ARTIFACTS_DIR}/packages/${package_dist}/${package_arch}"
    cp -r "${source}/"* "${packages_dir}/"
}

# create_archive creates a kitmaker archive for the specified component, os, and arch.
function create_archive() {
    local component=$1
    local branch=$2
    local os=$3
    local arch=$4
    local version=$5

    local name="${component}-${os}-${arch}"
    local archive="${KITMAKER_DIR}/${branch}/${name}-${version}.tar.gz"

    local scratch_dir="${KITMAKER_SCRATCH}/${name}/${branch}/${component}"
    local packages_dir="${scratch_dir}/.packages/"

    mkdir -p $(dirname "${archive}")

    tar zcvf "${archive}" -C "${scratch_dir}/.." "${component}"
    echo "Created: ${archive}"
    ls -l "${archive}"
    echo "With contents:"
    tar -tzvf "${archive}"
    echo ""

    # Clean up the scratch directories:
    rm -f "${scratch_dir}/.packages/"*
    rmdir "${scratch_dir}/.packages"
    rmdir "${scratch_dir}"
}

function optionally_add_property() {
    local property=$1
    local value=$2
    if [[ -n "${value}" ]]; then
        props+=("${property}=${value}")
    fi
}

function upload_archive() {
    local component=$1
    local branch=$2
    local os=$3
    local arch=$4
    local version=$5
    local package_builds=$(join_by , ${@:6})

    local name="${component}-${os}-${arch}"
    local archive="${KITMAKER_DIR}/${branch}/${name}-${version}.tar.gz"

    if [ ! -r "${archive}" ]; then
        echo "ERROR: File not found or not readable: ${archive}"
        exit 1
    fi
    local sha1_checksum=$(sha1sum -b "${archive}" | awk '{ print $1 }')

    local upload_url="${KITMAKER_ARTIFACTORY_REPO}/${branch}/${component}/${os}-${arch}/${version}/$(basename ${archive})"

    local props=()
    # Required KITMAKER properties:
    props+=("component_name=${component}")
    props+=("version=${version}")
    props+=("os=${os}")
    props+=("arch=${arch}")
    props+=("platform=${os}-${arch}")
    props+=("changelist=${GIT_COMMIT_SHORT}")
    props+=("branch=${branch}")
    props+=("source=https://gitlab.com/nvidia/container-toolkit/container-toolkit")
    # Package properties:
    props+=("package.epoch=${IMAGE_EPOCH}")
    props+=("package.version=${VERSION}")
    props+=("package.commit=${GIT_COMMIT}")
    optionally_add_property "package.builds" "${package_builds}"

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

component="nvidia_container_toolkit"
version="${VERSION%~rc.*}"
version_suffix=$(date -r "${IMAGE_EPOCH}" '+%Y.%m.%d.%s' || date -d @"${IMAGE_EPOCH}" '+%Y.%m.%d.%s')
kitmaker_version="${VERSION%~rc.*}.${version_suffix}"
kitmaker_os="linux"

# create_and_upload creates a kitmaker archive for the specified component, os, and arch and uploads it.
function create_and_upload() {
    local branch=$1
    local kitmaker_arch=$2
    local builds=${@:3}

    for build in ${builds}; do
        local package_dist=$(echo ${build} | cut -d- -f1)
        local package_arch=$(echo ${build} | cut -d- -f2)

        add_distro "${component}" "${branch}" "${kitmaker_os}" "${kitmaker_arch}" "${package_dist}" "${package_arch}"
    done

    create_archive "${component}" "${branch}" "${kitmaker_os}" "${kitmaker_arch}" "${kitmaker_version}"
    upload_archive "${component}" "${branch}" "${kitmaker_os}" "${kitmaker_arch}" "${kitmaker_version}" ${builds}
}

# Create archive for x86_64 linux distributions
create_and_upload "main" "x86_64" "ubuntu18.04-amd64" "centos7-x86_64"

# Create archive for sbsa linux distributions
create_and_upload "main" "sbsa" "ubuntu18.04-arm64" "centos7-aarch64"
# Create archive for aarch64 linux distributions
# NOTE: From the perspective of the NVIDIA Container Toolkit aarch64 is just a duplicate of sbsa
create_and_upload "main" "aarch64" "ubuntu18.04-arm64" "centos7-aarch64"

# Create archive for ppc64le linux distributions
create_and_upload "main" "ppc64le" "ubuntu18.04-ppc64le" "centos8-ppc64le"
