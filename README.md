# NVIDIA Container Toolkit

[![GitHub license](https://img.shields.io/github/license/NVIDIA/nvidia-container-toolkit?style=flat-square)](https://raw.githubusercontent.com/NVIDIA/nvidia-container-toolkit/main/LICENSE)
[![Documentation](https://img.shields.io/badge/documentation-wiki-blue.svg?style=flat-square)](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/overview.html)
[![Package repository](https://img.shields.io/badge/packages-repository-b956e8.svg?style=flat-square)](https://nvidia.github.io/libnvidia-container)

![nvidia-container-stack](https://cloud.githubusercontent.com/assets/3028125/12213714/5b208976-b632-11e5-8406-38d379ec46aa.png)

## Introduction

The NVIDIA Container Toolkit allows users to build and run GPU-accelerated containers. The toolkit includes a container runtime [library](https://github.com/NVIDIA/libnvidia-container) and utilities to automatically configure containers to leverage NVIDIA GPUs.

Product documentation including an architecture overview, platform support, and installation and usage guides can be found in the [documentation repository](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/overview.html).

## Getting Started

This section installs the NVIDIA Container Toolkit on Ubuntu or Debian and runs a GPU container with Docker.

Make sure your system meets the following prerequisites:

* The [NVIDIA driver](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html#prerequisites) for your Linux distribution is installed. You do not need to install the CUDA Toolkit on the host system.
* Your platform is listed on the [supported platforms page](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/supported-platforms.html).
* [Docker](https://docs.docker.com/engine/install/) is installed.

**Step 1: Configure the production repository**

```bash
sudo apt-get update && sudo apt-get install -y --no-install-recommends \
   ca-certificates \
   curl \
   gnupg2
curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | sudo gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg \
  && curl -s -L https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list | \
    sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | \
    sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list
```

**Step 2: Install the NVIDIA Container Toolkit packages**

```bash
sudo apt-get update
sudo apt-get install -y nvidia-container-toolkit
```

**Step 3: Configure Docker**

```bash
sudo nvidia-ctk runtime configure --runtime=docker
sudo systemctl restart docker
```

**Step 4: Run a sample workload**

```bash
sudo docker run --rm --runtime=nvidia --gpus all ubuntu nvidia-smi
```

The container prints the `nvidia-smi` table with the driver version, the CUDA version, and the GPUs it can access, as shown in the [sample workload guide](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/sample-workload.html).

To install on other distributions such as RHEL, Fedora, or SUSE, or to configure other container runtimes such as containerd, CRI-O, or Podman, refer to the [installation guide](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html).

## Usage

The [user guide](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/user-guide.html) provides information on the configuration and command line options available when running GPU containers with Docker.

## Issues and Contributing

[Checkout the Contributing document!](CONTRIBUTING.md)

* Please let us know by [filing a new issue](https://github.com/NVIDIA/nvidia-container-toolkit/issues/new)
* Contributions must adhere to the [Code of Conduct](CODE_OF_CONDUCT.md).

## Support and Getting Help
Please start [a discussion on the GitHub project](https://github.com/NVIDIA/nvidia-container-toolkit/discussions) for any questions. You can also join our [community Slack](https://inviter.co/dsx-ai-factory). Your feedback is appreciated.
