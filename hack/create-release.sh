# Copyright 2024 NVIDIA CORPORATION
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

if [ -z "$1" ]; then
  VERSION=$(awk -F= '/^VERSION/ { print $2 }' versions.mk | tr -d '[:space:]')
else
  VERSION=$1
fi


PRERELEASE_FLAG=""
REPO="stable"
if [[ ${VERSION} == v*-rc.* ]]; then
    PRERELEASE_FLAG="--prerelease"
    REPO="experimental"
fi

REPOSITORY=NVIDIA/nvidia-container-toolkit

echo "Creating draft release"
gh release create ${VERSION} \
            --draft \
            --title "${VERSION}" \
            -R "${REPOSITORY}" \
            --verify-tag \
            --prerelease

echo "Uploading release artifacts for ${VERSION}"

PACKAGE_ROOT=release-${VERSION}-${REPO}

# THIRD_PARTY_NOTICES.md is attached to the release rather than added to the
# packages or the image: the notices can be distributed alongside the artifacts,
# and this keeps the package and image contents unchanged. It is generated and
# committed from this tree, so the copy uploaded here is the one that describes
# the tagged commit.
gh release upload ${VERSION} \
    ${PACKAGE_ROOT}/nvidia-container-toolkit_${VERSION#v}_*.tar.gz \
    ${PACKAGE_ROOT}/nvidia-container-toolkit_${VERSION#v}_checksums.txt \
    THIRD_PARTY_NOTICES.md \
    --clobber \
    -R ${REPOSITORY}
