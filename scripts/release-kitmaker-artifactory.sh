#!/usr/bin/env bash

# Copyright (c) 2022, NVIDIA CORPORATION.  All rights reserved.
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
$(basename "${BASH_SOURCE[0]}") DIST-ARCH ARTIFACTORY_URL
    DIST: The distribution.
    ARCH: The architecture.
    ARTIFACTORY_URL must contain repo path for package, including hostname.

Environment Variables
    ARTIFACTORY_TOKEN: must contain an auth token. [required]
EOF
    exit 1
}

set -e

SCRIPTS_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../scripts && pwd )"
PROJECT_ROOT="$( cd "${SCRIPTS_DIR}/.." && pwd )"
COMPONENT_NAME="nvidia-container-toolkit"

if [[ $# -ne 2 ]]; then
    assert_usage "$@"
fi

source "${SCRIPTS_DIR}"/utils.sh

DISTARCH=$1
DIST=${DISTARCH%-*}
ARCH=${DISTARCH#*-}
ARTIFACTORY_URL=$2

CURL=${CURL:-curl}

if [[ -z "${DIST}" || -z "${ARCH}" ]]; then
    echo "ERROR: Distro and Architecture must be specified." >&2
    assert_usage "$@"
fi

# TODO: accept ARTIFACTS_DIR as a command-line argument
: "${ARTIFACTS_DIR="${PROJECT_ROOT}/artifacts"}"

if [[ ! -d "${ARTIFACTS_DIR}" ]]; then
    echo "ERROR: ARTIFACTS_DIR does not exist." >&2
    assert_usage "$@"
fi

if [[ -z "${ARTIFACTORY_TOKEN}" ]]; then
    echo "ERROR: ARTIFACTORY_TOKEN must be defined." >&2
    assert_usage "$@"
fi

# TODO: accept KITMACKER_DIR as a command-line argument
: "${KITMAKER_DIR="${PROJECT_ROOT}/artifacts/kitmaker"}"

eval $(${SCRIPTS_DIR}/get-component-versions.sh)

# Returns the key=value property if the value isn't empty
# Prepends with ";" if needed
set_prop_value() {
    local key=$1
    local value=$2
    if [ -n "${value}" ]; then
        if [ -z "${PROPS}" ]; then
            echo "${key}=${value}"
        else
            echo ";${key}=${value}"
        fi
    fi
}

process_props() {
    local dist=$1
    local arch=$2

    PROPS+=$(set_prop_value "component_name" "${COMPONENT_NAME}")
    PROPS+=$(set_prop_value "version" "${VERSION}")
    PROPS+=$(set_prop_value "os" "${dist}")
    PROPS+=$(set_prop_value "arch" "${arch}")
    PROPS+=$(set_prop_value "platform" "${dist}-${arch}")
    # TODO: Use `git describe` to get this information if it's not available.
    PROPS+=$(set_prop_value "changelist" "${CI_COMMIT_SHA}")
    PROPS+=$(set_prop_value "branch" "${CI_COMMIT_REF_NAME}")

    # Gitlab variables to expose
    for var in CI_PROJECT_ID CI_PIPELINE_ID CI_JOB_ID CI_JOB_URL CI_PROJECT_PATH; do
        if [ -n "${!var}" ]; then
            PROPS+=$(set_prop_value "${var}" "${!var}")
        fi
    done
}

## NOT USED:
## can substitute this function place of upload_file to modify properties of
## existing file instead of uploading files.
# Sets the properties on a path
# Relies on global variables: ARTIFACTORY_TOKEN, ARTIFACTORY_URL
set_props() {
    local dist="$1"
    local arch="$2"
    local kitmakerfilename="$3"

    # extract the Artifactory hostname
    artifactory_host=$(echo "${ARTIFACTORY_URL##https://}" | awk -F'/' '{print $1}')
    local image_path="${ARTIFACTORY_URL#https://${artifactory_host}/}/${dist}/${arch}/${kitmakerfilename}"

    local PROPS
    process_props "${DIST}" "${ARCH}"

    echo "Setting ${image_path} with properties: ${PROPS}"
    if ! ${CURL} -fs -H "X-JFrog-Art-Api: ${ARTIFACTORY_TOKEN}" \
        -X PUT \
        "https://${artifactory_host}/artifactory/api/storage/${image_path}?properties=${PROPS}&recursive=0" ; then
        echo "ERROR: set props failed: ${image_path}"
        exit 1
    fi
}

# Uploads file to ARTIFACTS_DIR/<os>/<arch>/<filename>
# Relies on global variables: DIST, ARCH, ARTIFACTORY_TOKEN, ARTIFACTORY_URL
upload_file() {
    local dist=$1
    local arch=$2
    local file=$3

    # extract the Artifactory hostname
    artifactory_host=$(echo "${ARTIFACTORY_URL##https://}" | awk -F'/' '{print $1}')
    # get base part of the ARTIFACTORY_URL without hostname
    local image_path="${ARTIFACTORY_URL#https://${artifactory_host}/}/${dist}/${arch}"

    local PROPS
    process_props "${dist}" "${arch}"

    if [ ! -r "${file}" ]; then
        echo "ERROR: File not found or not readable: ${file}"
        exit 1
    fi

    # Collect sum
    SHA1_SUM=$(sha1sum -b "${file}" | awk '{ print $1 }')

    echo "Uploading ${image_path} from ${file}"
    if ! ${CURL} -f \
        -H "X-JFrog-Art-Api: ${ARTIFACTORY_TOKEN}" \
        -H "X-Checksum-Sha1: ${SHA1_SUM}" \
        ${file:+-T ${file}} -X PUT \
        "https://${artifactory_host}/artifactory/${image_path};${PROPS}" ;
        then
        echo "ERROR: upload file failed: ${file}"
        exit 1
    fi
}

function push-kitmaker-artifactory() {
    local dist=$1
    local arch=$2
    local archive=$3

    upload_file "${dist}" "${arch}" "${archive}"
}

# kitmakerize-distro creates a tar.gz archive for the specified dist-arch combination.
# The archive is created at ${KITMAKER_DIR}/${name}.tar.gz (where ${name} is the third positional argument)
function kitmakerize-distro() {
    local dist="$1"
    local arch="$2"
    local archive="$3"

    local name=$(basename "${archive%%.tar.gz}")
    ## Copy packages into directory layout for .tar.gz
    # TODO: make scratch_dir configurable
    local scratch_dir="$(dirname ${archive})/.scratch/${name}"
    local packages_dir="${scratch_dir}/.packages/"

    mkdir -p "${packages_dir}"

    # Copy the extracted files to the .packages directory so that a kitmaker file can be created.
    source="${ARTIFACTS_DIR}/packages/${dist}/${arch}"
    cp -r "${source}/"* "${packages_dir}/"

    ## Tar up the directory structure created above
    tar zcvf "${archive}" -C "${scratch_dir}/.." "${name}"
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

kitmaker_name="${COMPONENT_NAME//-/_}-${DIST}-${ARCH}-${NVIDIA_CONTAINER_TOOLKIT_PACKAGE_VERSION}"
kitmaker_archive="${KITMAKER_DIR}/${kitmaker_name}.tar.gz"
kitmakerize-distro "${DIST}" "${ARCH}" "${kitmaker_archive}"
push-kitmaker-artifactory "${DIST}" "${ARCH}" "${kitmaker_archive}"
