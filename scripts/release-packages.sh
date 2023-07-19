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

source "${SCRIPTS_DIR}"/utils.sh

PACKAGE_REPO_ROOT=$1
if [[ ! -d ${PACKAGE_REPO_ROOT} ]]; then
    echo "The specified PACKAGE_REPO_ROOT '${PACKAGE_REPO_ROOT}' must exist"
    exit 1
fi

: ${REFERENCE:="HEAD"}
if [[ $# -ge 2 ]]; then
    REFERENCE=$2
fi

SHA=$(git rev-parse --short=8 ${REFERENCE})
IMAGE_NAME="registry.gitlab.com/nvidia/container-toolkit/container-toolkit/staging/container-toolkit"
IMAGE_TAG=${SHA}-packaging

: ${VERSION:="$(get_version_from_image ${IMAGE_NAME}:${IMAGE_TAG} ${SHA})"}

REPO="experimental"
if [[ ${VERSION/rc./} == ${VERSION} ]]; then
    REPO="stable"
fi

PACKAGE_CACHE=release-${VERSION}-${REPO}

echo "Fetching packages with SHA '${SHA}' as tag '${VERSION}' to ${PACKAGE_CACHE}"
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
    local by_package_type=$4

    local src_dist=${target%-*}
    local dst_dist=${src_dist/amazonlinux/amzn}

    local pkg_type=unknown
    local arch=${target##*-}
    local dst_arch=${arch}

    case ${src_dist} in
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
        dst_arch=${arch//ppc64le/ppc64el}
        ;;
    *) echo "ERROR: unexpected distribution ${src_dist}"
       exit 1
        ;;
    esac

    if [[ x"${by_package_type}" == x"true" ]]; then
        dst_dist=${pkg_type}
    fi

    local src=${src_root}/${src_dist}/${arch}
    local dst=${dst_root}/${dst_dist}/${dst_arch}

    if [[ ! -d ${src} || -z $(ls ${src}/*.${pkg_type}) ]]; then
        echo "Skipping ${src}"
        return
    fi
    mkdir -p ${dst}

    for f in $(ls ${src}/libnvidia-container*.${pkg_type} ${src}/nvidia-container-toolkit*.${pkg_type}); do
        # We never release nvidia-container-toolkit-operator-extensions packages
        if [[ "${f/"nvidia-container-toolkit-operator-extensions"/}" != "${f}" ]]; then
            echo "Skipping ${f}"
            continue
        fi

        df=${dst}/$(basename ${f})
        df_stable=${df//"/experimental/"/"/stable/"}
        if [[ -f "${df}" ]]; then
            echo "${df} already exists; skipping"
        elif [[ ${REPO} == "experimental" && -f ${df_stable} ]]; then
            echo "${df_stable} already exists; skipping"
        else
            cp ${f} ${df}
        fi

    done
    if [[ ${REPO} == "stable" ]]; then
        for f in $(ls ${src}/nvidia-container-runtime*.${pkg_type} ${src}/nvidia-docker*.${pkg_type}); do
            df=${dst}/$(basename ${f})
            df_stable=${df//"/experimental/"/"/stable/"}
            if [[ -f "${df}" ]]; then
                echo "${df} already exists; skipping"
            elif [[ ${REPO} == "experimental" && -f ${df_stable} ]]; then
                echo "${df_stable} already exists; skipping"
            else
                cp ${f} ${df}
            fi
        done
    fi
}

targets=${all[@]}

_current_branch=$(git -C ${PACKAGE_REPO_ROOT} rev-parse --abbrev-ref HEAD)
if [[ x"${_current_branch}" != x"gh-pages" ]]; then
    echo "It is expected that the gh-pages branch be checked out"
    exit 1
fi

: ${UPSTREAM_REMOTE:="origin"}

: ${UPSTREAM_REFERENCE:="${UPSTREAM_REMOTE}/gh-pages"}
git -C ${PACKAGE_REPO_ROOT} reset --hard ${UPSTREAM_REFERENCE}
git -C ${PACKAGE_REPO_ROOT} clean -fdx ${REPO}

for target in ${targets[@]}; do
    echo "checking target=${target}"
    by_package_type=
    case ${target} in
    ubuntu18.04-* | centos7-*)
        by_package_type="true"
        ;;
    centos8-ppc64le)
        by_package_type="false"
        ;;
    *)
        echo "Skipping target ${target}"
        continue
        ;;
    esac
    sync ${target} ${PACKAGE_CACHE}/packages ${PACKAGE_REPO_ROOT}/${REPO} ${by_package_type}
done

git -C ${PACKAGE_REPO_ROOT} add ${REPO}

if [[ "${REPO}" == "stable" ]]; then
# Stable release
git -C ${PACKAGE_REPO_ROOT} commit -s -F- <<EOF
Add packages for NVIDIA Container Toolkit ${VERSION} release

These include:
* libnvidia-container* ${LIBNVIDIA_CONTAINER_PACKAGE_VERSION}
* nvidia-container-toolkit ${NVIDIA_CONTAINER_TOOLKIT_PACKAGE_VERSION}
* nvidia-container-runtime ${NVIDIA_CONTAINER_RUNTIME_PACKAGE_VERSION}
* nvidia-docker ${NVIDIA_DOCKER_PACKAGE_VERSION}
EOF
else
# Experimental / release candidate release
git -C ${PACKAGE_REPO_ROOT} commit -s -F- <<EOF
Add packages for NVIDIA Container Toolkit ${VERSION} ${REPO} release

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
git -C ${PACKAGE_REPO_ROOT} commit -s -m "TOFIX: Sign deb packages for ${VERSION} in ${REPO}"

sign rpm

git -C ${PACKAGE_REPO_ROOT} add ${REPO}
git -C ${PACKAGE_REPO_ROOT} commit -s -m "TOFIX: Sign rpm packages for ${VERSION} in ${REPO}"

echo "To publish changes, go to ${PACKAGE_REPO_ROOT} and run: "
echo "   git rebase -i ${UPSTREAM_REFERENCE}"
