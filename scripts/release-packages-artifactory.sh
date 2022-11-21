#!/usr/bin/env bash

# Copyright (c) 2021-2022, NVIDIA CORPORATION.  All rights reserved.
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

# Dependencies:
#   regctl
#

function assert_usage() {
cat >&2 << EOF
Incorrect arguments: $*
$(basename "${BASH_SOURCE[0]}") DIST-ARCH
    DIST: The distribution.
    ARCH: The architecture.

Environment Variables
    ARTIFACTORY_TOKEN: must contain an auth token. [required]
    LIB_TAG: optional package tag.
    CI_COMMIT_REF_NAME: provided by CI/CD system.
    CI_COMMIT_SHA: provided by CI/CD system.
EOF
    exit 1
}

SCRIPTS_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../scripts && pwd )"
PROJECT_ROOT="$( cd "${SCRIPTS_DIR}/.." && pwd )"

source "${SCRIPTS_DIR}"/utils.sh

if [[ $# -ne 1 ]]; then
    assert_usage "$@"
fi

DISTARCH=$1
ARTIFACTORY_PATH=$2
DIST=${DISTARCH%-*}
ARCH=${DISTARCH##*-}

CURL=${CURL:-curl}

if [[ -z "${DIST}" || -z "${ARCH}" ]]; then
    echo "ERROR: Distro and Architecture must be specified." >&2
    assert_usage "$@"
fi

if [[ -z "${ARTIFACTORY_PATH}" ]]; then
    echo "ERROR: Package repo must be specified." >&2
    assert_usage "$@"
fi

if [[ -z "${ARTIFACTORY_TOKEN}" ]]; then
    echo "ERROR: ARTIFACTORY_TOKEN must be defined." >&2
    assert_usage "$@"
fi

# TODO: accept PACKAGES_DIR as a command-line argument
: "${ARTIFACTS_DIR="${PROJECT_ROOT}/artifacts"}"
: "${PACKAGES_DIR="${ARTIFACTS_DIR}/packages"}"

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
    local file=$3
    local component_name="${file%%.*}"
    component_name="${component_name%-*}"
    local pkg_type="$(package_type $dist)"

    ## Component owner is free to define these
    # PROPS+=$(set_prop_value "version" "${VERSION}")
    # PROPS+=$(set_prop_value "lws_version" "${LWS_VER}")
    # PROPS+=$(set_prop_value "platform" "${DISTARCH}")

    # TODO: Use `git describe` to get this information if it's not available.
    PROPS+=$(set_prop_value "changelist" "${CI_COMMIT_SHA}")
    PROPS+=$(set_prop_value "branch" "${CI_COMMIT_REF_NAME}")

    # PROPS+=$(set_prop_value "category" "utils")
    # PROPS+=$(set_prop_value "platform" "${DISTARCH}")

    # Gitlab variables to expose
    for var in CI_PROJECT_ID CI_PIPELINE_ID CI_JOB_ID CI_JOB_URL CI_PROJECT_PATH; do
        if [ -n "${!var}" ]; then
            PROPS+=$(set_prop_value "${var}" "${!var}")
        fi
    done

    # We also set the package-specific properties to allow this to be used for other artifactory repositories
    PROPS+=$(set_prop_value "${pkg_type}.distribution" "${dist}")
    PROPS+=$(set_prop_value "${pkg_type}.architecture" "${arch}")
    PROPS+=$(set_prop_value "${pkg_type}.component" "${component_name}")
}

# Uploads file ARTIFACTORY_PATH
# Relies on global variables: DIST, ARCH, ARTIFACTORY_TOKEN, ARTIFACTORY_PATH
upload_file() {
    local dist=$1
    local arch=$2
    local file=$3

    # TODO: These should be set by envvars
    local artifactory_host="urm.nvidia.com"
    local artifactory_repo="$(get_artifactory_repository $dist)"

    if [ ! -r "${file}" ]; then
        echo "ERROR: File not found or not readable: ${file}"
        exit 1
    fi

    local PROPS
    process_props "${dist}" "${arch}" "${file}"

    # Collect sum
    SHA1_SUM=$(sha1sum -b "${file}" | awk '{ print $1 }')

    url="https://${artifactory_host}/artifactory/${artifactory_repo}/${dist}/${arch}/$(basename "${file}")"
    # NOTE: The URL to set the properties through the API is:
    # "https://${artifactory_host}/artifactory/api/storage/${artifactory_repo}/${dist}/${arch}/$(basename ${file})"

    echo "Uploading ${file} to ${url}"
    if ! ${CURL} -f \
          -H "X-JFrog-Art-Api: ${ARTIFACTORY_TOKEN}" \
          -H "X-Checksum-Sha1: ${SHA1_SUM}" \
          ${file:+-T ${file}} -X PUT \
            "${url};${PROPS}" ;
        then
            echo "ERROR: upload file failed: ${file}"
            exit 1
    fi
}

function push-artifactory() {
    local dist="$1"
    local arch="$2"

    source="${ARTIFACTS_DIR}/packages/${dist}/${arch}"

    find "${source}" -maxdepth 1 | while read -r f ; do
        upload_file "$dist" "$arch" "$f"
    done
}

# TODO: use this to adapt as a general purpose command-line tool
# case "${COMMAND}" in
#   set)
#     set_props
#     ;;
#   upload)
#     if [ -z "${UPLOAD_FILE}" ]; then
#       echo "ERROR: Upload package filename must be set using -f"
#       usage
#     fi
#
#     upload_file
#     ;;
#   *)
#     echo "ERROR: Invalid command ${COMMAND}"
#     usage
#     ;;
# esac

push-artifactory "${DIST}" "${ARCH}"
