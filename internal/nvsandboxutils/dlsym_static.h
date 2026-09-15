/**
# Copyright 2026 NVIDIA CORPORATION
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

// These wrappers find the entry points only because lib.go loads the library
// with RTLD_GLOBAL, which is the scope dlsym(RTLD_DEFAULT) searches.

#ifndef __NVSANDBOXUTILS_DLSYM_H__
#define __NVSANDBOXUTILS_DLSYM_H__

#include "nvsandboxutils.h"

nvSandboxUtilsRet_t nvSandboxUtilsInit_dl(nvSandboxUtilsInitInput_t *input);
nvSandboxUtilsRet_t nvSandboxUtilsShutdown_dl(void);
nvSandboxUtilsRet_t nvSandboxUtilsGetDriverVersion_dl(char *version, unsigned int length);
nvSandboxUtilsRet_t nvSandboxUtilsGetGpuResource_dl(nvSandboxUtilsGpuRes_t *request);
nvSandboxUtilsRet_t nvSandboxUtilsGetFileContent_dl(char *filePath, char *content, unsigned int *contentSize);

#endif // __NVSANDBOXUTILS_DLSYM_H__
