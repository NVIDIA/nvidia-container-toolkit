Name: nvidia-container-runtime
Version: %{version}
Release: %{release}
BuildArch: noarch
Group: Development Tools

Vendor: NVIDIA CORPORATION
Packager: NVIDIA CORPORATION <cudatools@nvidia.com>

Summary: NVIDIA container runtime
URL: https://github.com/NVIDIA/nvidia-container-runtime
# runc NOTICE file: https://github.com/opencontainers/runc/blob/main/NOTICE
License: ASL 2.0

Source0: LICENSE

Obsoletes: nvidia-container-runtime < 2.0.0
Requires: nvidia-container-toolkit >= %{toolkit_version}, nvidia-container-toolkit < 2.0.0

%description
A meta-package that allows installation flows expecting the nvidia-container-runtime
to be migrated to installing the NVIDIA Container Toolkit packages directly.

%prep
cp %{SOURCE0} .

%install

%files
%license LICENSE

%changelog
# As of 3.6.0-1 we generate the release information automatically
* %{release_date} NVIDIA CORPORATION <cudatools@nvidia.com> %{version}-%{release}
- As of 3.6.0-1 the package changelog is generated automatically. This means that releases since 3.6.0-1 all contain this same changelog entry updated for the version being released.
- Bump nvidia-container-toolkit dependency to %{toolkit_version}
