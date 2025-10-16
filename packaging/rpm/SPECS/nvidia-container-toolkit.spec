Name: nvidia-container-toolkit
Version: %{_version}
Release: %{_release}
Group: Development Tools

Vendor: NVIDIA CORPORATION
Packager: NVIDIA CORPORATION <cudatools@nvidia.com>

Summary: NVIDIA Container Toolkit
URL: https://github.com/NVIDIA/nvidia-container-toolkit
License: Apache-2.0

Source0: nvidia-container-runtime-hook
Source1: nvidia-ctk
Source2: LICENSE
Source3: nvidia-container-runtime
Source4: nvidia-container-runtime.cdi
Source5: nvidia-container-runtime.legacy
Source6: nvidia-cdi-hook
Source7: nvidia-cdi-refresh.service
Source8: nvidia-cdi-refresh.path
Source9: nvidia-cdi-refresh.env

Obsoletes: nvidia-container-runtime <= 3.5.0-1, nvidia-container-runtime-hook <= 1.4.0-2
Provides: nvidia-container-runtime
Provides: nvidia-container-runtime-hook
Requires: libnvidia-container-tools == %{version}-%{release}, libnvidia-container-tools < 2.0.0
Requires: nvidia-container-toolkit-base == %{version}-%{release}

%description
Provides tools and utilities to enable GPU support in containers.

%prep
cp %{SOURCE0} %{SOURCE1} %{SOURCE2} %{SOURCE3} %{SOURCE4} %{SOURCE5} %{SOURCE6} %{SOURCE7} %{SOURCE8} %{SOURCE9} .

%install
mkdir -p %{buildroot}%{_bindir}
mkdir -p %{buildroot}%{_sysconfdir}/systemd/system/
mkdir -p %{buildroot}%{_sysconfdir}/nvidia-container-toolkit

install -m 755 -t %{buildroot}%{_bindir} nvidia-container-runtime-hook
install -m 755 -t %{buildroot}%{_bindir} nvidia-container-runtime
install -m 755 -t %{buildroot}%{_bindir} nvidia-container-runtime.cdi
install -m 755 -t %{buildroot}%{_bindir} nvidia-container-runtime.legacy
install -m 755 -t %{buildroot}%{_bindir} nvidia-ctk
install -m 755 -t %{buildroot}%{_bindir} nvidia-cdi-hook
install -m 644 -t %{buildroot}%{_sysconfdir}/systemd/system nvidia-cdi-refresh.service
install -m 644 -t %{buildroot}%{_sysconfdir}/systemd/system nvidia-cdi-refresh.path
install -m 644 -t %{buildroot}%{_sysconfdir}/nvidia-container-toolkit nvidia-cdi-refresh.env

%post
if [ $1 -gt 1 ]; then  # only on package upgrade
  mkdir -p %{_localstatedir}/lib/rpm-state/nvidia-container-toolkit
  cp -af %{_bindir}/nvidia-container-runtime-hook %{_localstatedir}/lib/rpm-state/nvidia-container-toolkit
fi

%posttrans
if [ ! -e %{_bindir}/nvidia-container-runtime-hook ]; then
  # repairing lost file nvidia-container-runtime-hook
  cp -avf %{_localstatedir}/lib/rpm-state/nvidia-container-toolkit/nvidia-container-runtime-hook %{_bindir}
fi
rm -rf %{_localstatedir}/lib/rpm-state/nvidia-container-toolkit
ln -sf %{_bindir}/nvidia-container-runtime-hook %{_bindir}/nvidia-container-toolkit

%postun
if [ "$1" = 0 ]; then  # package is uninstalled, not upgraded
  if [ -L %{_bindir}/nvidia-container-toolkit ]; then rm -f %{_bindir}/nvidia-container-toolkit; fi
fi

%files
%license LICENSE
%{_bindir}/nvidia-container-runtime-hook

%changelog
# As of 1.10.0-1 we generate the release information automatically
* %{release_date} NVIDIA CORPORATION <cudatools@nvidia.com> %{version}-%{release}
- See https://github.com/NVIDIA/nvidia-container-toolkit/blob/%{git_commit}/CHANGELOG.md
- Bump libnvidia-container dependency to libnvidia-container-tools == %{version}-%{release}

# The BASE package consists of the NVIDIA Container Runtime and the NVIDIA Container Toolkit CLI.
# This allows the package to be installed on systems where no NVIDIA Container CLI is available.
%package base
Summary: NVIDIA Container Toolkit Base
Obsoletes: nvidia-container-runtime <= 3.5.0-1, nvidia-container-runtime-hook <= 1.4.0-2
Provides: nvidia-container-runtime
# Since this package allows certain components of the NVIDIA Container Toolkit to be installed separately
# it conflicts with older versions of the nvidia-container-toolkit package that also provide these files.
Conflicts: nvidia-container-toolkit <= 1.10.0-1

%description base
Provides tools such as the NVIDIA Container Runtime and NVIDIA Container Toolkit CLI to enable GPU support in containers.

%post base
# Generate the default config; If this file already exists no changes are made.
%{_bindir}/nvidia-ctk --quiet config --config-file=%{_sysconfdir}/nvidia-container-runtime/config.toml --in-place

# Reload systemd unit cache and enable nvidia-cdi-refresh services on both install and upgrade
if command -v systemctl >/dev/null 2>&1; then
  SYSTEMD_STATE=$(systemctl is-system-running 2>/dev/null || true)
  case "$SYSTEMD_STATE" in
    running|degraded)
      systemctl daemon-reload || echo "Warning: Failed to reload systemd daemon" >&2
      systemctl enable --now nvidia-cdi-refresh.path || echo "Warning: Failed to enable nvidia-cdi-refresh.path" >&2
      systemctl enable --now nvidia-cdi-refresh.service || echo "Warning: Failed to enable nvidia-cdi-refresh.service" >&2
      
      # Trigger CDI spec regeneration immediately after install/upgrade
      echo "Regenerating NVIDIA CDI specification..."
      systemctl start nvidia-cdi-refresh.service || echo "Warning: Failed to trigger CDI refresh" >&2
      ;;
  esac
fi

%files base
%license LICENSE
%{_bindir}/nvidia-container-runtime
%{_bindir}/nvidia-ctk
%{_bindir}/nvidia-cdi-hook
%{_sysconfdir}/systemd/system/nvidia-cdi-refresh.service
%{_sysconfdir}/systemd/system/nvidia-cdi-refresh.path
%config(noreplace) %{_sysconfdir}/nvidia-container-toolkit/nvidia-cdi-refresh.env

# The OPERATOR EXTENSIONS package consists of components that are required to enable GPU support in Kubernetes.
# This package is not distributed as part of the NVIDIA Container Toolkit RPMs.
%package operator-extensions
Summary: NVIDIA Container Toolkit Operator Extensions
Requires: nvidia-container-toolkit-base == %{version}-%{release}

%description operator-extensions
Provides tools for using the NVIDIA Container Toolkit with the GPU Operator

%files operator-extensions
%license LICENSE
%{_bindir}/nvidia-container-runtime.cdi
%{_bindir}/nvidia-container-runtime.legacy
