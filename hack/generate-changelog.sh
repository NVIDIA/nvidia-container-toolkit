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
Generate a changelog for the specified tag
Usage: $this --reference <tag> [--remote <remote_name>]

Options:
  --since     specify the tag to start the changelog from (default: latest tag)
  --remote    specify the remote to fetch tags from (default: upstream)
  --version   specify the version to be released
  --help/-h   show this help and exit

EOF
}

REMOTE="upstream"
VERSION=""
REFERENCE=

# Parse command line options
while [[ $# -gt 0 ]]; do
    key="$1"
    case $key in
        --since)
        REFERENCE="$2"
        shift # past argument
        shift # past value
        ;;
        --remote)
        REMOTE="$2"
        shift # past argument
        shift # past value
        ;;
        --version)
        VERSION="$2"
        shift # past argument
        shift # past value
        ;;
        --help/-h)  usage
            exit 0
            ;;
        *)  usage
            exit 1
            ;;
    esac
done

# Fetch the latest tags from the remote
git fetch $REMOTE --tags

# if REFERENCE is not set, get the latest tag
if [ -z "$REFERENCE" ]; then
    REFERENCE=$(git describe --tags $(git rev-list --tags --max-count=1))
fi

# Print the changelog
echo "## Changelog"
echo ""
echo "### Version $VERSION"

# Iterate over the commit messages and ignore the ones that start with "Merge" or "Bump"
git log --pretty=format:"%s" $REFERENCE..@ | grep -Ev "(^Merge )|(^Bump)" |  sed 's/^\(.*\)/- \1/g'
