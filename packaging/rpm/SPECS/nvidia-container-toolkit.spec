Name: nvidia-container-toolkit
Version: %{version}
Release: %{release}
Group: Development Tools

Vendor: NVIDIA CORPORATION
Packager: NVIDIA CORPORATION <cudatools@nvidia.com>

Summary: NVIDIA container runtime hook
URL: https://github.com/NVIDIA/nvidia-container-runtime
License: Apache-2.0

Source0: nvidia-container-toolkit
Source1: config.toml
Source2: oci-nvidia-hook
Source3: oci-nvidia-hook.json
Source4: LICENSE

Obsoletes: nvidia-container-runtime < 2.0.0, nvidia-container-runtime-hook
Provides: nvidia-container-runtime-hook
Requires: libnvidia-container-tools >= 1.3.0, libnvidia-container-tools < 2.0.0

%description
Provides a OCI hook to enable GPU support in containers.

%prep
cp %{SOURCE0} %{SOURCE1} %{SOURCE2} %{SOURCE3} %{SOURCE4} .

%install
mkdir -p %{buildroot}%{_bindir}
install -m 755 -t %{buildroot}%{_bindir} nvidia-container-toolkit

mkdir -p %{buildroot}/etc/nvidia-container-runtime
install -m 644 -t %{buildroot}/etc/nvidia-container-runtime config.toml

mkdir -p %{buildroot}/usr/libexec/oci/hooks.d
install -m 755 -t %{buildroot}/usr/libexec/oci/hooks.d oci-nvidia-hook

mkdir -p %{buildroot}/usr/share/containers/oci/hooks.d
install -m 644 -t %{buildroot}/usr/share/containers/oci/hooks.d oci-nvidia-hook.json

%posttrans
ln -sf %{_bindir}/nvidia-container-toolkit %{_bindir}/nvidia-container-runtime-hook

%postun
rm -f %{_bindir}/nvidia-container-runtime-hook

%files
%license LICENSE
%{_bindir}/nvidia-container-toolkit
%config /etc/nvidia-container-runtime/config.toml
/usr/libexec/oci/hooks.d/oci-nvidia-hook
/usr/share/containers/oci/hooks.d/oci-nvidia-hook.json

%changelog
* Fri Dec 11 2020 NVIDIA CORPORATION <cudatools@nvidia.com> 1.4.0-1
- Add 'compute' capability to list of defaults
- Add dependence on libnvidia-container-tools >= 1.3.1

* Wed Sep 16 2020 NVIDIA CORPORATION <cudatools@nvidia.com> 1.3.0-1
- Promote 1.3.0-0.1.rc.2 to 1.3.0-1
- Add dependence on libnvidia-container-tools >= 1.3.0

* Mon Aug 10 2020 NVIDIA CORPORATION <cudatools@nvidia.com> 1.3.0-0.1.rc.2
- 2c180947 Add more tests for new semantics with device list from volume mounts
- 7c003857 Refactor accepting device lists from volume mounts as a boolean

* Fri Jul 24 2020 NVIDIA CORPORATION <cudatools@nvidia.com> 1.3.0-0.1.rc.1
- b50d86c1 Update build system to accept a TAG variable for things like rc.x
- fe65573b Add common CI tests for things like golint, gofmt, unit tests, etc.
- da6fbb34 Revert "Add ability to merge envars of the form NVIDIA_VISIBLE_DEVICES_*"
- a7fb3330 Flip build-all targets to run automatically on merge requests
- 8b248b66 Rename github.com/NVIDIA/container-toolkit to nvidia-container-toolkit
- da36874e Add new config options to pull device list from mounted files instead of ENVVAR

* Wed Jul 22 2020 NVIDIA CORPORATION <cudatools@nvidia.com> 1.2.1-1
- 4e6e0ed4 Add 'ngx' to list of *all* driver capabilities
- 2f4af743 List config.toml as a config file in the RPM SPEC

* Wed Jul 08 2020 NVIDIA CORPORATION <cudatools@nvidia.com> 1.2.0-1
- 8e0aab46 Fix repo listed in changelog for debian distributions
- 320bb6e4 Update dependence on libnvidia-container to 1.2.0
- 6cfc8097 Update package license to match source license
- e7dc3cbb Fix debian copyright file
- d3aee3e0 Add the 'ngx' driver capability

* Wed Jun 03 2020 NVIDIA CORPORATION <cudatools@nvidia.com> 1.1.2-1
- c32237f3 Add support for parsing Linux Capabilities for older OCI specs

* Tue May 19 2020 NVIDIA CORPORATION <cudatools@nvidia.com> 1.1.1-1
- d202aded Update dependence to libnvidia-container 1.1.1

* Fri May 15 2020 NVIDIA CORPORATION <cudatools@nvidia.com> 1.1.0-1
- 4e4de762 Update build system to support multi-arch builds
- fcc1d116 Add support for MIG (Multi-Instance GPUs)
- d4ff0416 Add ability to merge envars of the form NVIDIA_VISIBLE_DEVICES_* 
- 60f165ad Add no-pivot option to toolkit
