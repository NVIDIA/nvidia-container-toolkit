# NVIDIA Container Toolkit

[![GitHub license](https://img.shields.io/github/license/NVIDIA/nvidia-container-toolkit?style=flat-square)](https://raw.githubusercontent.com/NVIDIA/nvidia-container-toolkit/main/LICENSE)
[![Documentation](https://img.shields.io/badge/documentation-wiki-blue.svg?style=flat-square)](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/overview.html)
[![Package repository](https://img.shields.io/badge/packages-repository-b956e8.svg?style=flat-square)](https://nvidia.github.io/libnvidia-container)

![nvidia-container-stack](https://cloud.githubusercontent.com/assets/3028125/12213714/5b208976-b632-11e5-8406-38d379ec46aa.png)

## Introduction

The NVIDIA Container Toolkit allows users to build and run GPU accelerated containers. The toolkit includes a container runtime [library](https://github.com/NVIDIA/libnvidia-container) and utilities to automatically configure containers to leverage NVIDIA GPUs.

Product documentation including an architecture overview, platform support, and installation and usage guides can be found in the [documentation repository](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/overview.html).

## Getting Started

**Make sure you have installed the [NVIDIA driver](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html#nvidia-drivers) for your Linux Distribution**
**Note that you do not need to install the CUDA Toolkit on the host system, but the NVIDIA driver needs to be installed**

For instructions on getting started with the NVIDIA Container Toolkit, refer to the [installation guide](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html#installation-guide).

## Usage

The [user guide](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/user-guide.html) provides information on the configuration and command line options available when running GPU containers with Docker.

## Issues and Contributing

[Checkout the Contributing document!](CONTRIBUTING.md)

* Please let us know by [filing a new issue](https://github.com/NVIDIA/nvidia-container-toolkit/issues/new)
* You can contribute by creating a [merge request](https://gitlab.com/nvidia/container-toolkit/container-toolkit/-/merge_requests/new) to our public GitLab repository
