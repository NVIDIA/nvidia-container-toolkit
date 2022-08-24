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

# This script is used to build the packages for the components of the NVIDIA
# Container Stack. These include the nvidia-container-toolkit in this repository
# as well as the components included in the third_party folder.
# All required packages are generated in the specified dist folder.

: ${LOCAL_REPO_DIRECTORY:=/local-repository}
if [[ -d ${LOCAL_REPO_DIRECTORY} ]]; then
    echo "Setting up local-repository"
    createrepo /local-repository

    cat >/etc/yum.repos.d/local.repo <<EOL
[local-repository]
name=NVIDIA Container Toolkit Local Packages
baseurl=file:///local-repository
enabled=0
gpgcheck=0
protect=1
EOL
    yum-config-manager --enable local-repository
elif [[ -n ${TEST_REPO} ]]; then
    ./install_repo.sh ${TEST_REPO}
else
    echo "Skipping repo setup"
fi

exec bash $@
