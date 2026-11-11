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

#include <string.h>

#include "nvsandboxutils.h"

#define MOCK_ROOTFS "/mock/rootfs"
#define MOCK_GPU_UUID "GPU-6fd91e39-6ba7-4d6b-b1d6-1dffb0bc7a2f"
#define MOCK_MIG_UUID "MIG-6fd91e39-6ba7-4d6b-b1d6-1dffb0bc7a2f"
#define MOCK_DRIVER_VERSION "999.88.77"

#define MOCK_SYMLINK_PATH "/dev/dri/by-path/pci-0000:41:00.0-card"
#define MOCK_SYMLINK_TARGET "../card1"

static int initialized = 0;

nvSandboxUtilsRet_t nvSandboxUtilsInit(nvSandboxUtilsInitInput_t *input)
{
    if (input == NULL) {
        return NVSANDBOXUTILS_ERROR_INVALID_ARG;
    }
    if (input->version != 1) {
        return NVSANDBOXUTILS_ERROR_VERSION_NOT_SUPPORTED;
    }
    if (input->type != NV_ROOTFS_PATH || strcmp(input->value, MOCK_ROOTFS) != 0) {
        return NVSANDBOXUTILS_ERROR_INVALID_ARG;
    }

    initialized = 1;
    return NVSANDBOXUTILS_SUCCESS;
}

nvSandboxUtilsRet_t nvSandboxUtilsShutdown(void)
{
    if (!initialized) {
        return NVSANDBOXUTILS_ERROR_UNINITIALIZED;
    }

    initialized = 0;
    return NVSANDBOXUTILS_SUCCESS;
}

nvSandboxUtilsRet_t nvSandboxUtilsGetDriverVersion(char *version, unsigned int length)
{
    if (!initialized) {
        return NVSANDBOXUTILS_ERROR_UNINITIALIZED;
    }
    if (version == NULL) {
        return NVSANDBOXUTILS_ERROR_INVALID_ARG;
    }
    if (length < sizeof(MOCK_DRIVER_VERSION)) {
        return NVSANDBOXUTILS_ERROR_INSUFFICIENT_SIZE;
    }

    strcpy(version, MOCK_DRIVER_VERSION);
    return NVSANDBOXUTILS_SUCCESS;
}

static char devNvidia0[] = "/dev/nvidia0";
static char devNvidiactl[] = "/dev/nvidiactl";

static nvSandboxUtilsGpuFileInfo_v1_t gpuFiles[] = {
    {
        .next = &gpuFiles[1],
        .fileType = NV_DEV,
        .fileSubType = NV_DEV_NVIDIA,
        .module = NV_GPU,
        .flags = NV_FILE_FLAG_HINT,
        .filePath = devNvidia0,
    },
    {
        .next = NULL,
        .fileType = NV_DEV,
        .fileSubType = NV_DEV_NVIDIA_CTL,
        .module = NV_DRIVER_NVIDIA,
        .flags = NV_FILE_FLAG_HINT,
        .filePath = devNvidiactl,
    },
};

nvSandboxUtilsRet_t nvSandboxUtilsGetGpuResource(nvSandboxUtilsGpuRes_t *request)
{
    if (!initialized) {
        return NVSANDBOXUTILS_ERROR_UNINITIALIZED;
    }
    if (request == NULL) {
        return NVSANDBOXUTILS_ERROR_INVALID_ARG;
    }
    if (request->version != 1) {
        return NVSANDBOXUTILS_ERROR_VERSION_NOT_SUPPORTED;
    }

    if (request->inputType == NV_GPU_INPUT_GPU_UUID && strcmp(request->input, MOCK_GPU_UUID) == 0) {
        request->files = &gpuFiles[0];
        return NVSANDBOXUTILS_SUCCESS;
    }
    if (request->inputType == NV_GPU_INPUT_MIG_UUID && strcmp(request->input, MOCK_MIG_UUID) == 0) {
        request->files = &gpuFiles[1];
        return NVSANDBOXUTILS_SUCCESS;
    }

    return NVSANDBOXUTILS_ERROR_DEVICE_NOT_FOUND;
}

nvSandboxUtilsRet_t nvSandboxUtilsGetFileContent(char *filePath, char *content, unsigned int *contentSize)
{
    if (!initialized) {
        return NVSANDBOXUTILS_ERROR_UNINITIALIZED;
    }
    if (filePath == NULL || content == NULL || contentSize == NULL) {
        return NVSANDBOXUTILS_ERROR_INVALID_ARG;
    }
    if (strcmp(filePath, MOCK_SYMLINK_PATH) != 0) {
        return NVSANDBOXUTILS_ERROR_FILEPATH_NOT_FOUND;
    }
    if (*contentSize < sizeof(MOCK_SYMLINK_TARGET)) {
        return NVSANDBOXUTILS_ERROR_INSUFFICIENT_SIZE;
    }

    strcpy(content, MOCK_SYMLINK_TARGET);
    *contentSize = sizeof(MOCK_SYMLINK_TARGET) - 1;
    return NVSANDBOXUTILS_SUCCESS;
}
