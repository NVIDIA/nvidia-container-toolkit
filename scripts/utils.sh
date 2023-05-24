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

# This list represents the distribution-architecture pairs that are actually published
# to the relevant repositories. This targets forwarded to the build-all-components script
# can be overridden by specifying command line arguments.
all=(
    centos7-aarch64
    centos7-x86_64
    centos8-aarch64
    centos8-ppc64le
    centos8-x86_64
    ubuntu18.04-amd64
    ubuntu18.04-arm64
    ubuntu18.04-ppc64le
)

# package_type returns the packaging type (deb or rpm) for the specfied distribution.
# An error is returned if the ditribution is unsupported.
function package_type() {
    local pkg_type
    case ${1} in
    amazonlinux*) pkg_type=rpm
        ;;
    centos*) pkg_type=rpm
        ;;
    debian*) pkg_type=deb
        ;;
    fedora*) pkg_type=rpm
        ;;
    opensuse-leap*) pkg_type=rpm
        ;;
    ubuntu*) pkg_type=deb
        ;;
    deb) pkg_type=deb
        ;;
    rpm) pkg_type=rpm
        ;;
    *) exit 1
        ;;
    esac
    echo "${pkg_type}"
}

function get_artifactory_repository() {
    local pkg_type=$(package_type $1)

    case ${pkg_type} in
    deb) echo "sw-gpu-cloudnative-debian-local"
        ;;
    rpm) echo "sw-gpu-cloudnative-rpm-local"
        ;;
    *) echo "sw-gpu-cloudnative-generic-local"
        ;;
    esac
}

function get_package_target() {
    local target=$1
    local dist=${target%-*}
    local arch=${target##*-}

    case ${target} in
    deb) echo ""
        ;;
    rpm) echo ""
        ;;
    *) echo "${dist}/${arch}"
        ;;
    esac
}

# extract-file copies a file from a specified image.
# If regctl is available this is used, otherwise a docker container is run and the file is copied from
# there.
function copy_file() {
    local image=$1
    local path_in_image=$2
    local path_on_host=$3
    if command -v regctl > /dev/null; then
        regctl image get-file "${image}" "${path_in_image}" "${path_on_host}"
    else
        # Note this will only work for destinations where the `path_on_host` is in `pwd`
        docker run --rm \
        -v "$(pwd):$(pwd)" \
        -w "$(pwd)" \
        -u "$(id -u):$(id -g)" \
        --entrypoint="bash" \
            "${image}" \
            -c "cp ${path_in_image} ${path_on_host}"
    fi
}

# extract_info extracts the value of the specified variable from the manifest.txt file.
function extract_from_manifest() {
    local variable=$1
    local manifest=$2
    local value=$(cat ${manifest} | grep "#${variable}=" | sed -e "s/#${variable}=//" | tr -d '\r')
    echo $value
}

# extract_info extracts the value of the specified variable from the manifest.txt file.
function extract_info() {
    extract_from_manifest $1 "${ARTIFACTS_DIR}/manifest.txt"
}

function get_version_from_image() {
    local image=$1
    local manifest="manifest-${2}.txt"
    copy_file ${image} "/artifacts/manifest.txt" ${manifest}
    version=$(extract_from_manifest "PACKAGE_VERSION" ${manifest})
    echo "v${version/\~/-}"
    rm -f ${manifest}
}

function join_by { local IFS="$1"; shift; echo "$*"; }