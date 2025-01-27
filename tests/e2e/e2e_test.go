/*
* Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
*     http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package e2e

import (
	"context"
	"flag"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Test context
var (
	ctx context.Context

	installCTK bool

	imageRepo string
	imageTag  string

	sshKey     string
	sshUser    string
	remoteHost string
)

func init() {
	flag.BoolVar(&installCTK, "install-ctk", false, "Install the NVIDIA Container Toolkit")
	flag.StringVar(&imageRepo, "image-repo", "", "Repository of the image to test")
	flag.StringVar(&imageTag, "image-tag", "", "Tag of the image to test")
	flag.StringVar(&sshKey, "ssh-key", "", "SSH key to use for remote login")
	flag.StringVar(&sshUser, "ssh-user", "", "SSH user to use for remote login")
	flag.StringVar(&remoteHost, "remote-host", "", "Hostname of the remote machine")
}

func TestMain(t *testing.T) {
	suiteName := "NVIDIA Container Toolkit E2E"

	RegisterFailHandler(Fail)
	RunSpecs(t,
		suiteName,
	)
}

// BeforeSuite runs before the test suite
var _ = BeforeSuite(func() {
	ctx = context.Background()
})
