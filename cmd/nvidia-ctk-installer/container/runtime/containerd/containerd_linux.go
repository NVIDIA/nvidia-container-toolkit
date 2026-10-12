/**
# Copyright 2020-2023 NVIDIA CORPORATION
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
	"fmt"
	"net"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	reloadBackoff     = 5 * time.Second
	maxReloadAttempts = 6

	socketMessageToGetPID = ""
)

// SignalContainerd sends a SIGHUP signal to the containerd daemon
func SignalContainerd(socket string) error {
	log.Infof("Sending SIGHUP signal to containerd")

	// Wrap the logic to perform the SIGHUP in a function so we can retry it on failure
	retriable := func() error {
		pid, err := getContainerdPid(socket)
		if err != nil {
			return fmt.Errorf("failed to get containerd pid: %w", err)
		}

		err = syscall.Kill(pid, syscall.SIGHUP)
		if err != nil {
			return fmt.Errorf("unable to send SIGHUP to 'containerd' process: %v", err)
		}

		return nil
	}

	// Try to send a SIGHUP up to maxReloadAttempts times
	var err error
	for i := range maxReloadAttempts {
		err = retriable()
		if err == nil {
			break
		}
		if i == maxReloadAttempts-1 {
			break
		}
		log.Warningf("Error signaling containerd, attempt %v/%v: %v", i+1, maxReloadAttempts, err)
		time.Sleep(reloadBackoff)
	}
	if err != nil {
		log.Warningf("Max retries reached %v/%v, aborting", maxReloadAttempts, maxReloadAttempts)
		return err
	}

	log.Infof("Successfully signaled containerd")

	return nil
}

// getContainerdPid resolves the PID of the containerd daemon from the peer
// credentials of its socket. SO_PASSCRED must be enabled before connect():
// the kernel only records credentials on data queued while it is set, and the
// server writes immediately on accept, so data queued earlier carries pid 0 —
// which kill(2) would interpret as the calling process's own process group.
func getContainerdPid(socket string) (int, error) {
	dialer := net.Dialer{
		Control: func(network, address string, c syscall.RawConn) error {
			var serr error
			if cerr := c.Control(func(fd uintptr) {
				serr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_PASSCRED, 1)
			}); cerr != nil {
				return fmt.Errorf("unable to issue call on socket fd: %w", cerr)
			}
			if serr != nil {
				return fmt.Errorf("unable to SetsockoptInt on socket fd: %w", serr)
			}
			return nil
		},
	}
	conn, err := dialer.Dial("unix", socket)
	if err != nil {
		return 0, fmt.Errorf("unable to dial: %w", err)
	}
	defer conn.Close()

	_, _, err = conn.(*net.UnixConn).WriteMsgUnix([]byte(socketMessageToGetPID), nil, nil)
	if err != nil {
		return 0, fmt.Errorf("unable to WriteMsgUnix on socket fd: %w", err)
	}

	oob := make([]byte, 1024)
	_, oobn, _, _, err := conn.(*net.UnixConn).ReadMsgUnix(nil, oob)
	if err != nil {
		return 0, fmt.Errorf("unable to ReadMsgUnix on socket fd: %w", err)
	}

	oob = oob[:oobn]
	scm, err := syscall.ParseSocketControlMessage(oob)
	if err != nil {
		return 0, fmt.Errorf("unable to ParseSocketControlMessage from message received on socket fd: %w", err)
	}

	ucred, err := syscall.ParseUnixCredentials(&scm[0])
	if err != nil {
		return 0, fmt.Errorf("unable to ParseUnixCredentials from message received on socket fd: %w", err)
	}

	if ucred.Pid <= 0 {
		return 0, fmt.Errorf("received invalid peer pid %d from socket credentials", ucred.Pid)
	}

	return int(ucred.Pid), nil
}
