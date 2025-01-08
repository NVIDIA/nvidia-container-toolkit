/**
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

package toolkit

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNvidiaContainerRuntimeInstallerWrapper(t *testing.T) {
	r := newNvidiaContainerRuntimeInstaller(nvidiaContainerRuntimeSource)
	require.Equal(t, nvidiaContainerRuntimeSource, r.source)
	require.Equal(t, filepath.Base(nvidiaContainerRuntimeSource), r.target.wrapperName)
	require.Equal(t, filepath.Base(nvidiaContainerRuntimeSource), r.wrapperName())
	require.Equal(t, filepath.Base(nvidiaContainerRuntimeSource)+".real", r.dotRealFilename())
	require.Nil(t, r.argv)
	require.Equal(t, map[string]string{"XDG_CONFIG_HOME": filepath.Join(destDirPattern, ".config")}, r.envm)
}
