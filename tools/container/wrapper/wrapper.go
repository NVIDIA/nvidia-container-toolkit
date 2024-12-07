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
	"strings"

	"golang.org/x/sys/unix"
)

func main() {
	program, err := os.Executable()
	if err != nil {
		log.Fatalf("failed to get executable: %v", err)
	}
	argv := makeArgv(program)
	envv := makeEnvv(program)
	if err := unix.Exec(program+".real", argv, envv); err != nil {
		log.Fatalf("failed to exec %s: %v", program+".real", err)
	}

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
