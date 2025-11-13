/**
# SPDX-FileCopyrightText: Copyright (c) 2025 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
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
**/

package main

/*
__attribute__((section(".nvctkiwc")))
char wrapper_config[4096] = {0};
extern char wrapper_config[4096];
*/
import "C"

import (
	"log"
	"os"
	"os/exec"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"

	"github.com/NVIDIA/nvidia-container-toolkit/cmd/nvidia-ctk-installer/toolkit/installer/wrappercore"
)

func main() {
	config := loadConfig()
	if config.RequiresKernelModule && !isNvidiaModuleLoaded() {
		log.Println("nvidia driver modules are not yet loaded, invoking default runtime")
		runtimePath := config.DefaultRuntimeExecutablePath
		if runtimePath == "" {
			runtimePath = "runc"
		}
		program, err := exec.LookPath(runtimePath)
		if err != nil {
			log.Fatalf("failed to find %s: %v", runtimePath, err)
		}
		argv := []string{runtimePath}
		argv = append(argv, os.Args[1:]...)
		if err := unix.Exec(program, argv, os.Environ()); err != nil {
			log.Fatalf("failed to exec %s: %v", program, err)
		}
	}
	program, err := os.Executable()
	if err != nil {
		log.Fatalf("failed to get executable: %v", err)
	}
	argv := makeArgv(config)
	envv := makeEnvv(config)
	if err := unix.Exec(program+".real", argv, envv); err != nil {
		log.Fatalf("failed to exec %s: %v", program+".real", err)
	}
}

func loadConfig() *wrappercore.WrapperConfig {
	ptr := unsafe.Pointer(&C.wrapper_config[0])
	sectionData := unsafe.Slice((*byte)(ptr), 4096)
	config, err := wrappercore.ReadWrapperConfigSection(sectionData)
	if err != nil {
		log.Fatalf("failed to load wrapper config: %v", err)
	}
	return config
}

func isNvidiaModuleLoaded() bool {
	_, err := os.Stat("/proc/driver/nvidia/version")
	return err == nil
}

func makeArgv(config *wrappercore.WrapperConfig) []string {
	argv := []string{os.Args[0] + ".real"}
	argv = append(argv, config.Argv...)
	return append(argv, os.Args[1:]...)
}

func makeEnvv(config *wrappercore.WrapperConfig) []string {
	var env []string
	for k, v := range config.Envm {
		value := v
		if strings.HasPrefix(k, "<") {
			k = k[1:]
			value = value + ":" + os.Getenv(k)
		} else if strings.HasPrefix(k, ">") {
			k = k[1:]
			value = os.Getenv(k) + ":" + value
		}
		env = append(env, k+"="+value)
	}
	return append(env, os.Environ()...)
}
