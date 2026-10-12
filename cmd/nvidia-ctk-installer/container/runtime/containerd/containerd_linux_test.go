/**
# SPDX-FileCopyrightText: Copyright (c) NVIDIA CORPORATION & AFFILIATES. All rights reserved.
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

package containerd

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// startMockServer listens on a unix socket and, like containerd's gRPC
// server, writes data immediately on accept.
func startMockServer(t *testing.T) string {
	t.Helper()

	socket := filepath.Join(t.TempDir(), "mock.sock")
	l, err := net.Listen("unix", socket)
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				_, _ = conn.Write([]byte("\x00\x00\x00\x04\x00\x00\x00\x00\x00"))
				_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
				buf := make([]byte, 64)
				_, _ = conn.Read(buf)
			}(conn)
		}
	}()

	return socket
}

func TestGetContainerdPid(t *testing.T) {
	socket := startMockServer(t)

	pid, err := getContainerdPid(socket)
	require.NoError(t, err)
	require.Equal(t, os.Getpid(), pid)
}

// TestGetContainerdPidNeverZero is a regression test for resolving pid 0 from
// data the server queued before SO_PASSCRED was enabled.
func TestGetContainerdPidNeverZero(t *testing.T) {
	socket := startMockServer(t)

	for i := range 200 {
		pid, err := getContainerdPid(socket)
		require.NoError(t, err, "attempt %d", i)
		require.Equal(t, os.Getpid(), pid, "attempt %d", i)
	}
}
