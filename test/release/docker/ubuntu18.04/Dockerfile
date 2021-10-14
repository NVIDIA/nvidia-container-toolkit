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

FROM ubuntu:18.04

ARG DEBIAN_FRONTEND noninteractive
RUN apt-get update && apt-get install --no-install-recommends -y \
    curl \
    gnupg2 \
    apt-transport-https \
    ca-certificates \
    apt-utils \
    ruby ruby-dev rubygems build-essential

RUN gem install --no-document fpm

# We create and install a dummy docker package since these dependencies are out of
# scope for the tests performed here.
RUN fpm -s empty \
    -t deb \
    --description "A dummy package for docker.io_18.06.0" \
    -n docker.io --version 18.06.0 \
    -p /tmp/docker.deb \
    --deb-no-default-config-files \
    && \
    dpkg -i /tmp/docker.deb \
    && \
    rm -f /tmp/docker.deb


ARG WORKFLOW=nvidia-docker
RUN curl -s -L https://nvidia.github.io/${WORKFLOW}/gpgkey | apt-key add - \
   && curl -s -L https://nvidia.github.io/${WORKFLOW}/ubuntu18.04/nvidia-docker.list | tee /etc/apt/sources.list.d/nvidia-docker.list \
   && apt-get update

COPY entrypoint.sh /
COPY install_repo.sh /

ENTRYPOINT [ "/entrypoint.sh" ]