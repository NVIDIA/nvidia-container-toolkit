#!/usr/bin/env bash

# Copyright (c) 2024, NVIDIA CORPORATION.  All rights reserved.
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

set -o pipefail

this=`basename $0`

usage () {
cat << EOF
Prepare for an NVIDIA Container Toolkit release

Usage: $this [-h] --previous-version <previous_version> --version <version>

Options:
  --previous-version    specify the previous version
  --version             specify the version for this release.
  --help/-h             show this help and exit

Example:

  $this --previous-version {{ PREVIOUS_VERSION}} --version {{ VERSION }}

EOF
}

# Parse command line options
previous_version=
version=
while [[ $# -gt 0 ]]; do
    key="$1"
    case $key in
        --previous-version)
            previous_version="$2"
            shift 2
            ;;
        --version)
            version="$2"
            shift 2
            ;;
        --help/-h)
            usage
            exit 0
            ;;
        *)  usage
            exit 1
            ;;
    esac
done

# Check that no extra args were provided
if [ -z "$version" ]; then
    echo -e "ERROR: --version is required"
    usage
    exit 1
fi

if [ -z $previous_version ]; then
    echo -e "ERROR: --previous-version is required"
    usage
    exit 1
fi

#
# Modify files in the repo to point to new release
#
# Darwin or Linux
DOCKER="docker"
if [[ "$(uname)" == "Darwin" ]]; then
    SED="$DOCKER run -i --rm -v $(PWD):$(PWD) -w $(PWD) alpine:latest sed"
else
    SED="sed"
fi

if [[ "$FORCE" != "yes" ]]; then
    current_head=$(git rev-parse --abbrev-ref HEAD)
    if [[ "${current_head}" != "main" && "${current_head}" != release-* ]]; then
        echo "Release scripts should be run on 'main' or on a 'release-*' branch"
        exit 1
    fi
    git fetch
    git diff --quiet FETCH_HEAD
    if [[ $? -ne 0 ]]; then
        echo "Local changes detected:"
        git diff FETCH_HEAD | cat
        echo "Exiting"
        exit 1
    fi
fi

# Create a release issue.
echo "Creating release tracking issue"
cat RELEASE.md | sed "s/{{ .VERSION }}/${version}/g" | \
    gh issue create -F - \
        -R NVIDIA/nvidia-container-toolkit \
        --title "Release nvidia-container-toolkit ${version}" \
        --label release \
        --milestone ${version}

echo "Creating a version bump branch: bump-release-${version}"
git checkout -f -b bump-release-${version}

# Patch versions.mk
LIB_VERSION=${release%-*}
LIB_VERSION=${LIB_VERSION#v}
if [[ ${release} == v*-rc.* ]]; then
    LIB_TAG_STRING=" ${version#*-}"
else
    LIB_TAG_STRING=
fi

echo Patching versions.mk to refer to ${version}
$SED -i "s/^LIB_VERSION.*$/LIB_VERSION := $LIB_VERSION/" versions.mk
$SED -i "s/^LIB_TAG.*$/LIB_TAG :=$LIB_TAG_STRING/" versions.mk

git add versions.mk
git commit -s -m "Bump version for ${version} release"

if [[ ${version} != *-rc.* ]]; then
    # Patch README.md
    echo Patching README.md to refer to ${version}
    $SED -E -i -e "s/([^[:space:]])$previous_version([^[:alnum:]]|$)/\1${version}\2/g" README.md
    $SED -E -i -e "s/$pre_semver/$semver/g" README.md

    git add -u README.md
    git commit -s -m "Bump version to ${version} in README"
else
    echo "Skipping README update for prerelease version"
fi

echo "Please validated changes and create a pull request"
