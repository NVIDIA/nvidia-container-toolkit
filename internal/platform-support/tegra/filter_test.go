/**
# Copyright (c) NVIDIA CORPORATION.  All rights reserved.
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

package tegra

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIgnorePatterns(t *testing.T) {
	filtered := ignoreFilenamePatterns{"*.so", "*.so.[0-9]"}.Apply("/foo/bar/libsomething.so", "libsometing.so", "libsometing.so.1", "libsometing.so.1.2.3")

	require.ElementsMatch(t, []string{"libsometing.so.1.2.3"}, filtered)
}
