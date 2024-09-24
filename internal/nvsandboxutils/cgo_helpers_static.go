/**
# Copyright 2024 NVIDIA CORPORATION
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

package nvsandboxutils

var cgoAllocsUnknown = new(struct{})

func clen(n []byte) int {
	for i := 0; i < len(n); i++ {
		if n[i] == 0 {
			return i
		}
	}
	return len(n)
}

// Creates an int8 array of fixed input length to store the Go string.
// TODO: Add error check if input string has a length greater than INPUT_LENGTH
func convertStringToFixedArray(str string) [INPUT_LENGTH]int8 {
	var output [INPUT_LENGTH]int8
	for i, s := range str {
		output[i] = int8(s)
	}
	return output
}
