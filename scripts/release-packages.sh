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

function assert_usage() {
    echo "Incorrect arguments: $*"
    echo "$(basename ${BASH_SOURCE[0]}) PACKAGE_REPO_ROOT [SHA]"
    echo "\tPACKAGE_REPO_ROOT: The path to the libnvidia-container repository"
    echo "\tSHA: The SHA / reference to release. [Default: HEAD]"
    exit 1
}

set -e

SCRIPTS_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../scripts && pwd )"
PROJECT_ROOT="$( cd ${SCRIPTS_DIR}/.. && pwd )"

if [[ $# -lt 1 || $# -gt 2 ]]; then
    assert_usage $*
fi

PACKAGE_REPO_ROOT=$1
if [[ ! -d ${PACKAGE_REPO_ROOT} ]]; then
    echo "The specified PACKAGE_REPO_ROOT '${PACKAGE_REPO_ROOT}' must exist"
    exit 1
fi

: ${REFERENCE:="HEAD"}
if [[ $# -ge 2 ]]; then
    REFERENCE=$2
fi

eval $(${SCRIPTS_DIR}/get-component-versions.sh)

TAG=v"${NVIDIA_CONTAINER_TOOLKIT_PACKAGE_VERSION}"
SHA=$(git rev-parse --short=8 ${REFERENCE})

REPO="experimental"
if [[ ${TAG/rc./} == ${TAG} ]]; then
    REPO="stable"
fi

PACKAGE_CACHE=release-${TAG}-${REPO}

echo "Fetching packages with SHA '${SHA}' as tag '${TAG}' to ${PACKAGE_CACHE}"
IMAGE_NAME="registry.gitlab.com/nvidia/container-toolkit/container-toolkit/staging/container-toolkit"
IMAGE_TAG=${SHA}-packaging
${SCRIPTS_DIR}/pull-packages.sh \
    ${IMAGE_NAME}:${IMAGE_TAG} \
    ${PACKAGE_CACHE}

: ${ALL_RPMS:="$(find ${PACKAGE_CACHE} -name "*.rpm" -exec basename {} \; | sort | uniq)"}
: ${ALL_DEBS:="$(find ${PACKAGE_CACHE} -name "*.deb" -exec basename {} \; | sort | uniq)"}


PACKAGE_REPO_ROOT=$(cd "${PACKAGE_REPO_ROOT}" && pwd)
echo "Updating ${REPO} repo at ${PACKAGE_REPO_ROOT}"

docker build \
    -t nvidia/toolkit-deb-pkg-signer \
    -f ${SCRIPTS_DIR}/Dockerfile.sign.deb \
        ${SCRIPTS_DIR}

docker build \
    -t nvidia/toolkit-rpm-pkg-signer \
    -f ${SCRIPTS_DIR}/Dockerfile.sign.rpm \
        ${SCRIPTS_DIR}

function sync() {
    local target=$1
    local src_root=$2
    local dst_root=$3

    local src_dist=${target%-*}
    local dst_dist=${src_dist/amazonlinux/amzn}

    local pkg_type
    case ${src_dist} in
    amazonlinux*) pkg_type=rpm
        ;;
    centos*) pkg_type=rpm
        ;;
    debian*) pkg_type=deb
        ;;
    opensuse-leap*) pkg_type=rpm
        ;;
    ubuntu*) pkg_type=deb
        ;;
    *) echo "ERROR: unexpected distribution ${src_dist}"
        ;;
    esac

    local arch=${target##*-}
    local dst_arch=${arch}
    case ${src_dist} in
    ubuntu*) dst_arch=${arch//ppc64le/ppc64el}
    esac

    local src=${src_root}/${src_dist}/${arch}
    local dst=${dst_root}/${dst_dist}/${dst_arch}

    if [[ ! -d ${src} || -z $(ls ${src}/*.${pkg_type}) ]]; then
        echo "Skipping ${src}"
        return
    fi
    mkdir -p ${dst}
    cp ${src}/libnvidia-container*.${pkg_type} ${dst}
    cp ${src}/nvidia-container-toolkit*.${pkg_type} ${dst}
    if [[ ${REPO} == "stable" ]]; then
        cp ${src}/nvidia-container-runtime*.${pkg_type} ${dst}
        cp ${src}/nvidia-docker*.${pkg_type} ${dst}
    fi
}

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

targets=${all[@]}

_current_branch=$(git -C ${PACKAGE_REPO_ROOT} rev-parse --abbrev-ref HEAD)
if [[ x"${_current_branch}" != x"gh-pages" ]]; then
    echo "It is expected that the gh-pages branch be checked out"
    exit 1
fi

: ${UPSTREAM_REMOTE:="origin"}
_remote_name=$( git remote -v | grep "git@gitlab.com:nvidia/container-toolkit/libnvidia-container.git (push)" | cut -d$'\t' -f1 )
if [[ x"${_remote_name}" != x"${UPSTREAM_REMOTE}" ]]; then
    echo "Identified ${_remote_name} as git@gitlab.com:nvidia/container-toolkit/libnvidia-container.git remote."
    echo "Set UPSTREAM_REMOTE=${_remote_name} instead of ${UPSTREAM_REMOTE}"
fi

: ${UPSTREAM_REFERENCE:="${UPSTREAM_REMOTE}/gh-pages"}
git -C ${PACKAGE_REPO_ROOT} reset --hard ${UPSTREAM_REFERENCE}
git -C ${PACKAGE_REPO_ROOT} clean -fdx ${REPO}

for target in ${targets[@]}; do
    sync ${target} ${PACKAGE_CACHE} ${PACKAGE_REPO_ROOT}/${REPO}
done

git -C ${PACKAGE_REPO_ROOT} add ${REPO}

if [[ ${REPO} == "stable" ]]; then
# Stable release
git -C ${PACKAGE_REPO_ROOT} commit -s -F- <<EOF
Add packages for NVIDIA Container Toolkit ${TAG} release

These include:
* libnvidia-container* ${LIBNVIDIA_CONTAINER_PACKAGE_VERSION}
* nvidia-container-toolkit ${NVIDIA_CONTAINER_TOOLKIT_PACKAGE_VERSION}
* nvidia-container-runtime ${NVIDIA_CONTAINER_RUNTIME_PACKAGE_VERSION}
* nvidia-docker ${NVIDIA_DOCKER_PACKAGE_VERSION}
EOF
else
# Experimental / release candidate release
git -C ${PACKAGE_REPO_ROOT} commit -s -F- <<EOF
Add packages for NVIDIA Container Toolkit ${TAG} ${REPO} release

These include:
* libnvidia-container* ${LIBNVIDIA_CONTAINER_PACKAGE_VERSION}
* nvidia-container-toolkit ${NVIDIA_CONTAINER_TOOLKIT_PACKAGE_VERSION}
EOF
fi

: ${MASTER_KEY_PATH:? Path to master key MASTER_KEY_PATH must be set}
: ${SUB_KEY_PATH:? Path to sub key SUB_KEY_PATH must be set}
: ${GPG_LOCAL_USER:? GPG_LOCAL_USER must be set}
: ${GNUPG_CONF:=$(mktemp -d -t nvidia-container-toolkit-package-XXXXXXXXXX)}

function sign() {
    local pkg_type=$1
    docker run -it --rm \
        -e ALL_DEBS="${ALL_DEBS}" \
        -e ALL_RPMS="${ALL_RPMS}" \
        -e GPG_LOCAL_USER="${GPG_LOCAL_USER}" \
        -e TARGETS="${targets}" \
        -v ${PACKAGE_REPO_ROOT}/${REPO}:/sign-packages \
        -v ${MASTER_KEY_PATH}:/keys/master.key:ro \
        -v ${SUB_KEY_PATH}:/keys/sub.key:ro \
        -v ${SCRIPTS_DIR}:/helpers \
        -w /sign-packages \
            nvidia/toolkit-${pkg_type}-pkg-signer \
        bash -x -c "
        export GPG_TTY=\$(tty);
        gpg --import /keys/master.key;
        gpg --import /keys/sub.key;
        /helpers/packages-sign-all.sh;
        "

}

sign deb

git -C ${PACKAGE_REPO_ROOT} add ${REPO}
git -C ${PACKAGE_REPO_ROOT} commit -s -m "TOFIX: Sign deb packages for ${TAG} in ${REPO}"

sign rpm

git -C ${PACKAGE_REPO_ROOT} add ${REPO}
git -C ${PACKAGE_REPO_ROOT} commit -s -m "TOFIX: Sign rpm packages for ${TAG} in ${REPO}"

echo "To publish changes, go to ${PACKAGE_REPO_ROOT} and run: "
echo "   git rebase -i ${UPSTREAM_REFERENCE}"
