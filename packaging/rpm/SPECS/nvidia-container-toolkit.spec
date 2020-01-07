Name: nvidia-container-toolkit
Version: %{version}
Release: %{release}
Group: Development Tools

Vendor: NVIDIA CORPORATION
Packager: NVIDIA CORPORATION <cudatools@nvidia.com>

Summary: NVIDIA container runtime hook
URL: https://github.com/NVIDIA/nvidia-container-runtime
License: BSD

Source0: nvidia-container-toolkit
Source1: config.toml
Source2: oci-nvidia-hook
Source3: oci-nvidia-hook.json
Source4: LICENSE

Obsoletes: nvidia-container-runtime < 2.0.0, nvidia-container-runtime-hook
Provides: nvidia-container-runtime-hook
Requires: libnvidia-container-tools >= 0.1.0, libnvidia-container-tools < 2.0.0

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
/etc/nvidia-container-runtime/config.toml
/usr/libexec/oci/hooks.d/oci-nvidia-hook
/usr/share/containers/oci/hooks.d/oci-nvidia-hook.json

%changelog
* Fri May 15 2020 NVIDIA CORPORATION <cudatools@nvidia.com> 1.1.0-1
 - Add ability to merge envars of the form NVIDIA_VISIBLE_DEVICES_*
 - Extend fields we inspect in the runc spec to include linux capabilities
 - Add support for MIG
