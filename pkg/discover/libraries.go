/*
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
*/

package discover

import (
	"fmt"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup"
	log "github.com/sirupsen/logrus"
)

// NewLibraries constructs discoverer for libraries
func NewLibraries(root string) (Discover, error) {
	return NewLibrariesWithLogger(log.StandardLogger(), root)
}

// NewLibrariesWithLogger constructs discoverer for libraries with the specified logger
func NewLibrariesWithLogger(logger *log.Logger, root string) (Discover, error) {
	lookup, err := lookup.NewLibraryLocatorWithLogger(logger, root)
	if err != nil {
		return nil, fmt.Errorf("error constructing locator: %v", err)
	}

	d := mounts{
		logger:   logger,
		lookup:   lookup,
		required: requiredLibraries,
	}
	return &d, nil
}

// requiredLibraries defines a set of libraries and their labels
var requiredLibraries = map[string][]string{
	"utility": {
		"libnvidia-ml.so",  /* Management library */
		"libnvidia-cfg.so", /* GPU configuration */
	},
	"compute": {
		"libcuda.so",                   /* CUDA driver library */
		"libnvidia-opencl.so",          /* NVIDIA OpenCL ICD */
		"libnvidia-ptxjitcompiler.so",  /* PTX-SASS JIT compiler (used by libcuda) */
		"libnvidia-fatbinaryloader.so", /* fatbin loader (used by libcuda) */
		"libnvidia-allocator.so",       /* NVIDIA allocator runtime library */
		"libnvidia-compiler.so",        /* NVVM-PTX compiler for OpenCL (used by libnvidia-opencl) */
	},
	"video": {
		"libvdpau_nvidia.so",       /* NVIDIA VDPAU ICD */
		"libnvidia-encode.so",      /* Video encoder */
		"libnvidia-opticalflow.so", /* NVIDIA Opticalflow library */
		"libnvcuvid.so",            /* Video decoder */
	},
	"graphics": {
		//"libnvidia-egl-wayland.so",       /* EGL wayland platform extension (used by libEGL_nvidia) */
		"libnvidia-eglcore.so", /* EGL core (used by libGLES*[_nvidia] and libEGL_nvidia) */
		"libnvidia-glcore.so",  /* OpenGL core (used by libGL or libGLX_nvidia) */
		"libnvidia-tls.so",     /* Thread local storage (used by libGL or libGLX_nvidia) */
		"libnvidia-glsi.so",    /* OpenGL system interaction (used by libEGL_nvidia) */
		"libnvidia-fbc.so",     /* Framebuffer capture */
		"libnvidia-ifr.so",     /* OpenGL framebuffer capture */
		"libnvidia-rtcore.so",  /* Optix */
		"libnvoptix.so",        /* Optix */
	},
	"glvnd": {
		//"libGLX.so",                      /* GLX ICD loader */
		//"libOpenGL.so",                   /* OpenGL ICD loader */
		//"libGLdispatch.so",               /* OpenGL dispatch (used by libOpenGL, libEGL and libGLES*) */
		"libGLX_nvidia.so",       /* OpenGL/GLX ICD */
		"libEGL_nvidia.so",       /* EGL ICD */
		"libGLESv2_nvidia.so",    /* OpenGL ES v2 ICD */
		"libGLESv1_CM_nvidia.so", /* OpenGL ES v1 common profile ICD */
		"libnvidia-glvkspirv.so", /* SPIR-V Lib for Vulkan */
		"libnvidia-cbl.so",       /* VK_NV_ray_tracing */
	},
	"compat32": {
		"libGL.so",        /* OpenGL/GLX legacy _or_ compatibility wrapper (GLVND) */
		"libEGL.so",       /* EGL legacy _or_ ICD loader (GLVND) */
		"libGLESv1_CM.so", /* OpenGL ES v1 common profile legacy _or_ ICD loader (GLVND) */
		"libGLESv2.so",    /* OpenGL ES v2 legacy _or_ ICD loader (GLVND) */
	},
	"ngx": {
		"libnvidia-ngx.so", /* NGX library */
	},
	"dxcore": {
		"libdxcore.so", /* Core library for dxcore support */
	},
}
