# NVIDIA Container Toolkit Changelog

## v1.13.0-rc.1

* Bump version to v1.13.0-rc.1

## v1.12.0

* Promote `v1.12.0-rc.5` to `v1.12.0`
* Rename `nvidia cdi generate` `--root` flag to `--driver-root` to better indicate intent
* [libnvidia-container] Add nvcubins.bin to DriverStore components under WSL2
* [toolkit-container] Bump CUDA base images to 12.0.1

## v1.12.0-rc.5

* Fix bug here the `nvidia-ctk` path was not properly resolved. This causes failures to run containers when the runtime is configured in `csv` mode or if the `NVIDIA_DRIVER_CAPABILITIES` includes `graphics` or `display` (e.g. `all`).

## v1.12.0-rc.4

* Generate a minimum CDI spec version for improved compatibility.
* Add `--device-name-strategy` options to the `nvidia-ctk cdi generate` command that can be used to control how device names are constructed.
* Set default for CDI device name generation to `index` to generate device names such as `nvidia.com/gpu=0` or `nvidia.com/gpu=1:0` by default.

## v1.12.0-rc.3

* Don't fail if by-path symlinks for DRM devices do not exist
* Replace the --json flag with a --format [json|yaml] flag for the nvidia-ctk cdi generate command
* Ensure that the CDI output folder is created if required
* When generating a CDI specification use a blank host path for devices to ensure compatibility with the v0.4.0 CDI specification
* Add injection of Wayland JSON files
* Add GSP firmware paths to generated CDI specification
* Add --root flag to nvidia-ctk cdi generate command

## v1.12.0-rc.2

* Inject Direct Rendering Manager (DRM) devices into a container using the NVIDIA Container Runtime
* Improve logging of errors from the NVIDIA Container Runtime
* Improve CDI specification generation to support rootless podman
* Use `nvidia-ctk cdi generate` to generate CDI specifications instead of `nvidia-ctk info generate-cdi`
* [libnvidia-container] Skip creation of existing files when these are already mounted

## v1.12.0-rc.1

* Add support for multiple Docker Swarm resources
* Improve injection of Vulkan configurations and libraries
* Add `nvidia-ctk info generate-cdi` command to generated CDI specification for available devices
* [libnvidia-container] Include NVVM compiler library in compute libs

## v1.11.0

* Promote v1.11.0-rc.3 to v1.11.0

## v1.11.0-rc.3

* Build fedora35 packages
* Introduce an `nvidia-container-toolkit-base` package for better dependency management
* Fix removal of `nvidia-container-runtime-hook` on RPM-based systems
* Inject platform files into container on Tegra-based systems
* [toolkit container] Update CUDA base images to 11.7.1
* [libnvidia-container] Preload libgcc_s.so.1 on arm64 systems

## v1.11.0-rc.2

* Allow `accept-nvidia-visible-devices-*` config options to be set by toolkit container
* [libnvidia-container] Fix bug where LDCache was not updated when the `--no-pivot-root` option was specified

## v1.11.0-rc.1

* Add discovery of GPUDirect Storage (`nvidia-fs*`) devices if the `NVIDIA_GDS` environment variable of the container is set to `enabled`
* Add discovery of MOFED Infiniband devices if the `NVIDIA_MOFED` environment variable of the container is set to `enabled`
* Fix bug in CSV mode where libraries listed as `sym` entries in mount specification are not added to the LDCache.
* Rename `nvidia-container-toolkit` executable to `nvidia-container-runtime-hook` and create `nvidia-container-toolkit` as a symlink to `nvidia-container-runtime-hook` instead.
* Add `nvidia-ctk runtime configure` command to configure the Docker config file (e.g. `/etc/docker/daemon.json`) for use with the NVIDIA Container Runtime.

## v1.10.0

* Promote v1.10.0-rc.3 to v1.10.0

## v1.10.0-rc.3

* Use default config instead of raising an error if config file cannot be found
* Ignore NVIDIA_REQUIRE_JETPACK* environment variables for requirement checks
* Fix bug in detection of Tegra systems where `/sys/devices/soc0/family` is ignored
* Fix bug where links to devices were detected as devices
* [libnvida-container] Fix bug introduced when adding libcudadebugger.so to list of libraries

## v1.10.0-rc.2

* Add support for NVIDIA_REQUIRE_* checks for cuda version and arch to csv mode
* Switch to debug logging to reduce log verbosity
* Support logging to logs requested in command line
* Fix bug when launching containers with relative root path (e.g. using containerd)
* Allow low-level runtime path to be set explicitly as nvidia-container-runtime.runtimes option
* Fix failure to locate low-level runtime if PATH envvar is unset
* Replace experimental option for NVIDIA Container Runtime with nvidia-container-runtime.mode = csv option
* Use csv as default mode on Tegra systems without NVML
* Add --version flag to all CLIs
* [libnvidia-container] Bump libtirpc to 1.3.2
* [libnvidia-container] Fix bug when running host ldconfig using glibc compiled with a non-standard prefix
* [libnvidia-container] Add libcudadebugger.so to list of compute libraries

## v1.10.0-rc.1

* Include nvidia-ctk CLI in installed binaries
* Add experimental option to NVIDIA Container Runtime

## v1.9.0

* [libnvidia-container] Add additional check for Tegra in /sys/.../family file in CLI
* [libnvidia-container] Update jetpack-specific CLI option to only load Base CSV files by default
* [libnvidia-container] Fix bug (from 1.8.0) when mounting GSP firmware into containers without /lib to /usr/lib symlinks
* [libnvidia-container] Update nvml.h to CUDA 11.6.1 nvML_DEV 11.6.55
* [libnvidia-container] Update switch statement to include new brands from latest nvml.h
* [libnvidia-container] Process all --require flags on Jetson platforms
* [libnvidia-container] Fix long-standing issue with running ldconfig on Debian systems

## v1.8.1

* [libnvidia-container] Fix bug in determining cgroup root when running in nested containers
* [libnvidia-container] Fix permission issue when determining cgroup version

## v1.8.0

* Promote 1.8.0-rc.2-1 to 1.8.0

## v1.8.0-rc.2

* Remove support for building amazonlinux1 packages

## v1.8.0-rc.1

* [libnvidia-container] Add support for cgroupv2
* Release toolkit-container images from nvidia-container-toolkit repository

## v1.7.0

* Promote 1.7.0-rc.1-1 to 1.7.0
 * Bump Golang version to 1.16.4

## v1.7.0-rc.1

* Specify containerd runtime type as string in config tools to remove dependency on containerd package
* Add supported-driver-capabilities config option to allow for a subset of all driver capabilities to be specified

## v1.6.0

* Promote 1.6.0-rc.3-1 to 1.6.0
 * Fix unnecessary logging to stderr instead of configured nvidia-container-runtime log file

## v1.6.0-rc.3

* Add supported-driver-capabilities config option to the nvidia-container-toolkit
* Move OCI and command line checks for runtime to internal oci package

## v1.6.0-rc.2

* Use relative path to OCI specification file (config.json) if bundle path is not specified as an argument to the nvidia-container-runtime

## v1.6.0-rc.1

* Add AARCH64 package for Amazon Linux 2
* Include nvidia-container-runtime into nvidia-container-toolkit package

## v1.5.1

* Fix bug where Docker Swarm device selection is ignored if NVIDIA_VISIBLE_DEVICES is also set
* Improve unit testing by using require package and adding coverage reports
* Remove unneeded go dependencies by running go mod tidy
* Move contents of pkg directory to cmd for CLI tools
* Ensure make binary target explicitly sets GOOS

## v1.5.0

* Add dependence on libnvidia-container-tools >= 1.4.0
* Add golang check targets to Makefile
* Add Jenkinsfile definition for build targets
* Move docker.mk to docker folder

## v1.4.2

* Add dependence on libnvidia-container-tools >= 1.3.3

## v1.4.1

* Ignore NVIDIA_VISIBLE_DEVICES for containers with insufficent privileges
* Add dependence on libnvidia-container-tools >= 1.3.2

## v1.4.0

* Add 'compute' capability to list of defaults
* Add dependence on libnvidia-container-tools >= 1.3.1

## v1.3.0

* Promote 1.3.0-rc.2-1 to 1.3.0
* Add dependence on libnvidia-container-tools >= 1.3.0

## v1.3.0-rc.2

* 2c180947 Add more tests for new semantics with device list from volume mounts
* 7c003857 Refactor accepting device lists from volume mounts as a boolean

## v1.3.0-rc.1

* b50d86c1 Update build system to accept a TAG variable for things like rc.x
* fe65573b Add common CI tests for things like golint, gofmt, unit tests, etc.
* da6fbb34 Revert "Add ability to merge envars of the form NVIDIA_VISIBLE_DEVICES_*"
* a7fb3330 Flip build-all targets to run automatically on merge requests
* 8b248b66 Rename github.com/NVIDIA/container-toolkit to nvidia-container-toolkit
* da36874e Add new config options to pull device list from mounted files instead of ENVVAR

## v1.2.1

* 4e6e0ed4 Add 'ngx' to list of*all* driver capabilities
* 2f4af743 List config.toml as a config file in the RPM SPEC

## v1.2.0

*  8e0aab46 Fix repo listed in changelog for debian distributions
*  320bb6e4 Update dependence on libnvidia-container to 1.2.0
*  6cfc8097 Update package license to match source license
*  e7dc3cbb Fix debian copyright file
*  d3aee3e0 Add the 'ngx' driver capability

## v1.1.2

* c32237f3 Add support for parsing Linux Capabilities for older OCI specs

## v1.1.1

* d202aded Update dependence to libnvidia-container 1.1.1

## v1.1.0

* 4e4de762 Update build system to support multi-arch builds
* fcc1d116 Add support for MIG (Multi-Instance GPUs)
* d4ff0416 Add ability to merge envars of the form NVIDIA_VISIBLE_DEVICES_*
* 60f165ad Add no-pivot option to toolkit

## v1.0.5

* Initial release. Replaces older package nvidia-container-runtime-hook. (Closes: #XXXXXX)
