# SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
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
    echo "Missing argument $1"
    echo "$(basename "${BASH_SOURCE[0]}") TARGET"
    exit 1
}

set -e


if [ -n "${SKIP_REPACKAGE_RPMS}" ]; then
    exit 0
fi

SCRIPTS_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../scripts && pwd )"
PROJECT_ROOT="$( cd "${SCRIPTS_DIR}"/.. && pwd )"

if [[ $# -ne 1 ]]; then
    assert_usage "TARGET"
fi

TARGET=$1

source "${SCRIPTS_DIR}"/utils.sh

: "${DIST_DIR:=${PROJECT_ROOT}/dist}"
export DIST_DIR

case ${TARGET} in
    centos7-aarch64) platform="linux/aarch64"
        ;;
    centos7-x86_64) platform="linux/x86_64"
        ;;
    *) exit 0
        ;;
esac

arch="${TARGET/centos7-/}"
platform="${TARGET/centos7-/linux/}"
package_root="${DIST_DIR}/${TARGET/-//}"

# We build and rpmrebuild:${arch} image with no context.
docker build \
    --platform=${platform} \
    --build-arg arch="${arch}" \
    --tag rpmrebuild:${arch} \
    - < ${PROJECT_ROOT}/deployments/container/Dockerfile.rpmrebuild


updated=$(mktemp -d)
echo "Repackaging RPMs from ${package_root} to ${updated}:"
docker run \
    --platform=${platform} \
    --rm \
    -v ${package_root}:/dist \
    -v ${updated}:/updated \
    -w /dist \
        rpmrebuild:${arch}
