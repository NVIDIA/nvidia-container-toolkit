# Testing Packaging Workflows

## Building the Docker images:

```bash
make images
```

This assumes that the `nvidia-docker` workflow is being tested and that the test packages have been published to the `elezar.github.io` GitHub pages page. These can be overridden
using make variables.

Valid values for `workflow` are:
* `nvidia-docker`
* `nvidia-container-runtime`

This follows the instructions for setting up the [`nvidia-docker` repostitory](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html#setting-up-nvidia-container-toolkit).


## Testing local package changes

Running:
```bash
make local-ubuntu18.04
```
will build the `ubuntu18.04-amd64` packages for release and launch a docker container with these packages added to a local APT repository.

The various `apt` workflows can then be tested as if the packages were released to the `libnvidia-container` experimental repository.

The `local-centos8` make target is available for testing the `centos8-x86_64` workflows as representative of `yum`-based installation workflows.


### Example

In the `centos8`-based container above we see the following `nvidia-docker2` packages available with the `local-repository` disabled:

```bash
$ yum list --showduplicates nvidia-docker2 | tail -2
nvidia-docker2.noarch           2.5.0-1                            nvidia-docker
nvidia-docker2.noarch           2.6.0-1                            nvidia-docker
```

Installing `nvidia-docker2` shows:
```bash
$ yum install -y nvidia-docker2
```

installs the following packages:
```bash
$ yum list installed | grep nvidia
libnvidia-container-tools.x86_64   1.5.1-1                                 @libnvidia-container
libnvidia-container1.x86_64        1.5.1-1                                 @libnvidia-container
nvidia-container-runtime.x86_64    3.5.0-1                                 @nvidia-container-runtime
nvidia-container-toolkit.x86_64    1.5.1-2                                 @nvidia-container-runtime
nvidia-docker2.noarch              2.6.0-1                                 @nvidia-docker
```
Note the repositories where these packages were installed from.

We now enable the `local-repository` to simulate the new packages being published to the `libnvidia-container` experimental repository and check the available `nvidia-docker2` versions:
```bash
$ yum-config-manager --enable local-repository
$ yum list --showduplicates nvidia-docker2 | tail -2
nvidia-docker2.noarch         2.6.0-1                           nvidia-docker
nvidia-docker2.noarch         2.6.1-0.1.rc.1                    local-repository
```
Showing the new version available in the local repository.

Running:
```
$ yum install nvidia-docker2
Last metadata expiration check: 0:01:15 ago on Fri Sep 24 12:49:20 2021.
Package nvidia-docker2-2.6.0-1.noarch is already installed.
Dependencies resolved.
===============================================================================================================================
 Package                                Architecture        Version                        Repository                     Size
===============================================================================================================================
Upgrading:
 libnvidia-container-tools              x86_64              1.6.0-0.1.rc.1                 local-repository               48 k
 libnvidia-container1                   x86_64              1.6.0-0.1.rc.1                 local-repository               95 k
 nvidia-container-toolkit               x86_64              1.6.0-0.1.rc.1                 local-repository              1.5 M
     replacing  nvidia-container-runtime.x86_64 3.5.0-1
 nvidia-docker2                         noarch              2.6.1-0.1.rc.1                 local-repository               13 k

Transaction Summary
===============================================================================================================================
Upgrade  4 Packages

Total size: 1.7 M
```
Showing that all the components of the stack will be updated with versions from the `local-repository`.

After installation the installed packages are shown as:
```bash
$ yum list installed | grep nvidia
libnvidia-container-tools.x86_64   1.6.0-0.1.rc.1                          @local-repository
libnvidia-container1.x86_64        1.6.0-0.1.rc.1                          @local-repository
nvidia-container-toolkit.x86_64    1.6.0-0.1.rc.1                          @local-repository
nvidia-docker2.noarch              2.6.1-0.1.rc.1                          @local-repository
```
Showing that:
1. All versions have been installed from the same repository
2. The `nvidia-container-runtime` package was removed as it is no longer required.

The `nvidia-container-runtime`  executable is, however, still present on the system:
```bash
# ls -l /usr/bin/nvidia-container-runtime
-rwxr-xr-x 1 root root 2256280 Sep 24 12:42 /usr/bin/nvidia-container-runtime
```
