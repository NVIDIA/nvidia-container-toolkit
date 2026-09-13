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

#include <dlfcn.h>
#include <stddef.h>

#include "dlsym_static.h"

nvSandboxUtilsRet_t nvSandboxUtilsInit_dl(nvSandboxUtilsInitInput_t *input)
{
    nvSandboxUtilsRet_t (*fn)(nvSandboxUtilsInitInput_t *) = dlsym(RTLD_DEFAULT, "nvSandboxUtilsInit");
    if (fn == NULL) {
        return NVSANDBOXUTILS_ERROR_LIBRARY_LOAD;
    }

    return fn(input);
}

nvSandboxUtilsRet_t nvSandboxUtilsShutdown_dl(void)
{
    nvSandboxUtilsRet_t (*fn)(void) = dlsym(RTLD_DEFAULT, "nvSandboxUtilsShutdown");
    if (fn == NULL) {
        return NVSANDBOXUTILS_ERROR_LIBRARY_LOAD;
    }

    return fn();
}

nvSandboxUtilsRet_t nvSandboxUtilsGetDriverVersion_dl(char *version, unsigned int length)
{
    nvSandboxUtilsRet_t (*fn)(char *, unsigned int) = dlsym(RTLD_DEFAULT, "nvSandboxUtilsGetDriverVersion");
    if (fn == NULL) {
        return NVSANDBOXUTILS_ERROR_LIBRARY_LOAD;
    }

    return fn(version, length);
}

nvSandboxUtilsRet_t nvSandboxUtilsGetGpuResource_dl(nvSandboxUtilsGpuRes_t *request)
{
    nvSandboxUtilsRet_t (*fn)(nvSandboxUtilsGpuRes_t *) = dlsym(RTLD_DEFAULT, "nvSandboxUtilsGetGpuResource");
    if (fn == NULL) {
        return NVSANDBOXUTILS_ERROR_LIBRARY_LOAD;
    }

    return fn(request);
}

nvSandboxUtilsRet_t nvSandboxUtilsGetFileContent_dl(char *filePath, char *content, unsigned int *contentSize)
{
    nvSandboxUtilsRet_t (*fn)(char *, char *, unsigned int *) = dlsym(RTLD_DEFAULT, "nvSandboxUtilsGetFileContent");
    if (fn == NULL) {
        return NVSANDBOXUTILS_ERROR_LIBRARY_LOAD;
    }

    return fn(filePath, content, contentSize);
}
