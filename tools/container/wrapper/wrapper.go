/**
# Copyright (c) 2024, NVIDIA CORPORATION.  All rights reserved.
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

package main

import (
	"bufio"
	"errors"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

func main() {
	program, err := os.Executable()
	if err != nil {
		log.Fatalf("failed to get executable: %v", err)
	}
	if isRuntimeWrapper(program) && !isNvidiaModuleLoaded() {
		log.Println("nvidia driver modules are not yet loaded, invoking runc directly")
		program, err := exec.LookPath("runc")
		if err != nil {
			log.Fatalf("failed to find runc: %v", err)
		}
		argv := []string{"runc"}
		argv = append(argv, os.Args[1:]...)
		execve(program, argv, os.Environ())
	}
	argv := makeArgv(program)
	envv := makeEnvv(program)
	execve(program+".real", argv, envv)
}

func isRuntimeWrapper(program string) bool {
	return filepath.Base(program) == "nvidia-container-runtime" ||
		filepath.Base(program) == "nvidia-container-runtime.cdi" ||
		filepath.Base(program) == "nvidia-container-runtime.legacy"
}

func isNvidiaModuleLoaded() bool {
	_, err := os.Stat("/proc/driver/nvidia/version")
	return err == nil
}

func makeArgv(program string) []string {
	argv := []string{os.Args[0] + ".real"}
	f, err := os.Open(program + ".argv")
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			log.Printf("failed to open argv file: %v", err)
		}
		return append(argv, os.Args[1:]...)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		argv = append(argv, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("failed to read argv file: %v", err)
	}
	return append(argv, os.Args[1:]...)
}

func makeEnvv(program string) []string {
	f, err := os.Open(program + ".envv")
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			log.Printf("failed to open env file: %v", err)
		}
		return os.Environ()
	}
	defer f.Close()
	var env []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		kv := strings.SplitN(scanner.Text(), "=", 2)
		if strings.HasPrefix(kv[0], "<") {
			kv[0] = kv[0][1:]
			kv[1] = kv[1] + ":" + os.Getenv(kv[0])
		} else if strings.HasPrefix(kv[0], ">") {
			kv[0] = kv[0][1:]
			kv[1] = os.Getenv(kv[0]) + ":" + kv[1]
		}
		env = append(env, kv[0]+"="+kv[1])
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("failed to read argv file: %v", err)
	}
	return append(env, os.Environ()...)
}

func execve(program string, argv []string, envv []string) {
	if err := unix.Exec(program, argv, envv); err != nil {
		log.Fatalf("failed to exec %s: %v", program, err)
	}
}
