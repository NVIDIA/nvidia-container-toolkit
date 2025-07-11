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

package oci

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"syscall"
)

// shellMetachars represents a set of shell metacharacters that are commonly
// used for shell scripting and may lead to security vulnerabilities if not
// properly handled.
//
// These metacharacters include: | & ; ( ) < > \t \n $ \ `
const shellMetachars = "|&;()<> \t\n$\\`"

// metacharRegex matches any shell metacharcter.
var metacharRegex = regexp.MustCompile(`([` + regexp.QuoteMeta(shellMetachars) + `])`)

type syscallExec struct{}

var _ Runtime = (*syscallExec)(nil)

func (r syscallExec) Exec(args []string) error {
	args = Escape(args)
	err := syscall.Exec(args[0], args, os.Environ()) //nolint:gosec
	if err != nil {
		return fmt.Errorf("could not exec '%v': %v", args[0], err)
	}

	// syscall.Exec is not expected to return. This is an error state regardless of whether
	// err is nil or not.
	return fmt.Errorf("unexpected return from exec '%v'", args[0])
}

func (r syscallExec) String() string {
	return "exec"
}

// Escape1 escapes shell metacharacters in a single command-line argument.
func Escape1(arg string) string {
	if strings.ContainsAny(arg, shellMetachars) {
		e := metacharRegex.ReplaceAllString(arg, `\$1`)
		return fmt.Sprintf(`"%s"`, e)
	}
	return arg
}

// Escape escapes shell metacharacters in a slice of command-line arguments
// and returns a new slice containing the escaped arguments.
func Escape(args []string) []string {
	escaped := make([]string, len(args))
	for i := range args {
		escaped[i] = Escape1(args[i])
	}
	return escaped
}
