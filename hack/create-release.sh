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
./hack/generate-changelog.sh --version ${VERSION} | \
    gh release create ${VERSION} --notes-file "-" \
                --draft \
                --title "${VERSION}" \
                -R "${REPOSITORY}" \
                --verify-tag \
                --prerelease

echo "Uploading release artifacts for ${VERSION}"

PACKAGE_ROOT=release-${VERSION}-${REPO}

gh release upload ${VERSION} \
    ${PACKAGE_ROOT}/nvidia-container-toolkit_${VERSION#v}_*.tar.gz \
    ${PACKAGE_ROOT}/nvidia-container-toolkit_${VERSION#v}_checksums.txt \
    --clobber \
    -R ${REPOSITORY}
