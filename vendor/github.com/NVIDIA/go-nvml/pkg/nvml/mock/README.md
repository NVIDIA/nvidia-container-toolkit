# NVML Mock Framework

This package provides mock implementations of NVIDIA's NVML (NVIDIA Management Library) for testing and development purposes. The framework uses a shared factory system to define GPU configurations that can be easily extended and customized.

## Architecture

```
pkg/nvml/mock/
├── gpus/                         # GPU configuration definitions
│   ├── gpu.go                   # Config type and helpers
│   ├── a100.go                  # A100 GPU variants (Ampere)
│   ├── a30.go                   # A30 GPU variants (Ampere)
│   ├── h100.go                  # H100 GPU variants (Hopper)
│   ├── h200.go                  # H200 GPU variants (Hopper)
│   └── b200.go                  # B200 GPU variants (Blackwell)
├── server/                       # Shared server factory
│   ├── shared.go                # Core server types and mock functions
│   └── options.go               # Functional options (WithGPUs, etc.)
├── dgxa100/                      # DGX A100 implementation
│   ├── dgxa100.go               # Server and device implementation
│   └── dgxa100_test.go          # Comprehensive tests
├── dgxh100/                      # DGX H100 implementation
│   ├── dgxh100.go               # Server and device implementation
│   └── dgxh100_test.go          # Comprehensive tests
├── dgxh200/                      # DGX H200 implementation
│   ├── dgxh200.go               # Server and device implementation
│   └── dgxh200_test.go          # Comprehensive tests
└── dgxb200/                      # DGX B200 implementation
    ├── dgxb200.go               # Server and device implementation
    └── dgxb200_test.go          # Comprehensive tests
```

## Core Concepts

### Shared Factory (`shared.Config`)
Define the characteristics of individual GPU models including:

- Device properties (name, architecture, brand, PCI device ID)
- Compute capabilities (CUDA version, compute capability)
- Memory configuration
- MIG (Multi-Instance GPU) profiles and placements

### Server Configuration (`shared.ServerConfig`)
Define complete system configurations including:

- GPU configuration and count
- Driver, NVML, and CUDA versions

### MIG Profile Configuration (`shared.MIGProfileConfig`)
Define Multi-Instance GPU capabilities including:

- GPU instance profiles (slice configurations)
- Compute instance profiles
- Placement constraints and possibilities

## Usage Examples

### Basic Usage

```go
import (
    "github.com/NVIDIA/go-nvml/pkg/nvml/mock/dgxa100"
    "github.com/NVIDIA/go-nvml/pkg/nvml/mock/dgxh100"
    "github.com/NVIDIA/go-nvml/pkg/nvml/mock/dgxh200"
    "github.com/NVIDIA/go-nvml/pkg/nvml/mock/dgxb200"
    "github.com/NVIDIA/go-nvml/pkg/nvml/mock/gpus"
)

// Create default systems
serverA100 := dgxa100.New()   // A100-SXM4-40GB (8 GPUs)
serverH100 := dgxh100.New()   // H100-SXM5-80GB (8 GPUs)
serverH200 := dgxh200.New()   // H200-SXM5-141GB (8 GPUs)
serverB200 := dgxb200.New()   // B200-SXM5-180GB (8 GPUs)
```

> **Note:** The GPU configuration definitions in `internal/shared/gpus/` are internal
> to this module and cannot be imported by external consumers. Use the public
> `dgx*` package APIs shown above.

### Device Creation

```go
// Create devices with default configurations
deviceA100 := dgxa100.NewDevice(0)
deviceH100 := dgxh100.NewDevice(0)
deviceH200 := dgxh200.NewDevice(0)
deviceB200 := dgxb200.NewDevice(0)
```

### Available GPU Configurations (internal)

The following GPU configurations are defined in `internal/shared/gpus/` and used
internally by the `dgx*` packages:

| Family | Config | Memory | Architecture |
|--------|--------|--------|--------------|
| A100 | `A100_SXM4_40GB`, `A100_SXM4_80GB`, `A100_PCIE_40GB`, `A100_PCIE_80GB` | 40/80 GB | Ampere (8.0) |
| A30 | `A30_PCIE_24GB` | 24 GB | Ampere (8.0) |
| H100 | `H100_SXM5_80GB` | 80 GB | Hopper (9.0) |
| H200 | `H200_SXM5_141GB` | 141 GB | Hopper (9.0) |
| B200 | `B200_SXM5_180GB` | 180 GB | Blackwell (10.0) |

## Available GPU Models

### A100 Family (Ampere Architecture, 108 SMs)

- **A100 SXM4 40GB** (`gpus.A100_SXM4_40GB`)
  - Form factor: SXM4
  - Memory: 40GB HBM2
  - PCI Device ID: 0x20B010DE
  - CUDA Capability: 8.0
  - SMs per slice: 14 (1-slice), 28 (2-slice), 42 (3-slice), 56 (4-slice), 98 (7-slice)
  - MIG P2P: Not supported (`IsP2pSupported: 0`)

- **A100 SXM4 80GB** (`gpus.A100_SXM4_80GB`)
  - Form factor: SXM4
  - Memory: 80GB HBM2e
  - PCI Device ID: 0x20B210DE
  - CUDA Capability: 8.0

- **A100 PCIe 40GB** (`gpus.A100_PCIE_40GB`)
  - Form factor: PCIe
  - Memory: 40GB HBM2
  - PCI Device ID: 0x20F110DE
  - CUDA Capability: 8.0

- **A100 PCIe 80GB** (`gpus.A100_PCIE_80GB`)
  - Form factor: PCIe
  - Memory: 80GB HBM2e
  - PCI Device ID: 0x20B510DE
  - CUDA Capability: 8.0

### A30 Family (Ampere Architecture, 56 SMs)

- **A30 PCIe 24GB** (`gpus.A30_PCIE_24GB`)
  - Form factor: PCIe
  - Memory: 24GB HBM2
  - PCI Device ID: 0x20B710DE
  - CUDA Capability: 8.0
  - SMs per slice: 14 (1-slice), 28 (2-slice), 56 (4-slice)
  - MIG P2P: Not supported (`IsP2pSupported: 0`)
  - MIG slices: 1, 2, 4 (no 3-slice or 7-slice support)

### H100 Family (Hopper Architecture, 132 SMs)

- **H100 SXM5 80GB** (`gpus.H100_SXM5_80GB`)
  - Form factor: SXM5
  - Memory: 80GB HBM3
  - PCI Device ID: 0x233010DE
  - CUDA Capability: 9.0
  - SMs per slice: 16 (1-slice), 32 (2-slice), 48 (3-slice), 64 (4-slice), 112 (7-slice)
  - MIG P2P: Supported (`IsP2pSupported: 1`)
  - Includes REV1 (media extensions) and REV2 (expanded memory) profiles

### H200 Family (Hopper Architecture, 132 SMs)

- **H200 SXM5 141GB** (`gpus.H200_SXM5_141GB`)
  - Form factor: SXM5
  - Memory: 141GB HBM3e
  - PCI Device ID: 0x233310DE
  - CUDA Capability: 9.0
  - SMs per slice: 16 (1-slice), 32 (2-slice), 48 (3-slice), 64 (4-slice), 112 (7-slice)
  - MIG P2P: Supported (`IsP2pSupported: 1`)
  - Includes REV1 (media extensions) and REV2 (expanded memory) profiles

### B200 Family (Blackwell Architecture, 144 SMs)

- **B200 SXM5 180GB** (`gpus.B200_SXM5_180GB`)
  - Form factor: SXM5
  - Memory: 180GB HBM3e
  - PCI Device ID: 0x2B0010DE
  - CUDA Capability: 10.0
  - SMs per slice: 18 (1-slice), 36 (2-slice), 54 (3-slice), 72 (4-slice), 126 (7-slice)
  - MIG P2P: Supported (`IsP2pSupported: 1`)
  - Includes REV1 (media extensions) and REV2 (expanded memory) profiles

## Available Server Models

### DGX A100 Family

- **DGX A100 40GB** (default)
  - 8x A100 SXM4 40GB GPUs
  - Driver: 550.54.15
  - NVML: 12.550.54.15
  - CUDA: 12040

### DGX H100 Family

- **DGX H100 80GB** (default)
  - 8x H100 SXM5 80GB GPUs
  - Driver: 550.54.15
  - NVML: 12.550.54.15
  - CUDA: 12040

### DGX H200 Family

- **DGX H200 141GB** (default)
  - 8x H200 SXM5 141GB GPUs
  - Driver: 550.54.15
  - NVML: 12.550.54.15
  - CUDA: 12040

### DGX B200 Family

- **DGX B200 180GB** (default)
  - 8x B200 SXM5 180GB GPUs
  - Driver: 560.28.03
  - NVML: 12.560.28.03
  - CUDA: 12060

## MIG (Multi-Instance GPU) Support

All GPU configurations include comprehensive MIG profile definitions:

- **A100**: No P2P support in MIG (`IsP2pSupported: 0`)
  - Memory profiles differ between 40GB and 80GB variants
  - Supports standard NVIDIA MIG slice configurations (1, 2, 3, 4, 7 slices)
  - 108 SMs total with 14 SMs per slice
- **A30**: No P2P support in MIG (`IsP2pSupported: 0`)
  - Supports limited MIG slice configurations (1, 2, 4 slices only)
  - 56 SMs total with 14 SMs per slice
- **H100**: Full P2P support in MIG (`IsP2pSupported: 1`)
  - 80GB HBM3 memory with optimized slice allocations
  - Supports standard NVIDIA MIG slice configurations (1, 2, 3, 4, 7 slices)
  - 132 SMs total with 16 SMs per slice
  - Includes REV1 (media extensions) and REV2 (expanded memory) profiles
- **H200**: Full P2P support in MIG (`IsP2pSupported: 1`)
  - 141GB HBM3e memory with enhanced capacity
  - Supports standard NVIDIA MIG slice configurations (1, 2, 3, 4, 7 slices)
  - 132 SMs total with 16 SMs per slice
  - Includes REV1 (media extensions) and REV2 (expanded memory) profiles
- **B200**: Full P2P support in MIG (`IsP2pSupported: 1`)
  - 180GB HBM3e memory with next-generation capacity
  - Supports standard NVIDIA MIG slice configurations (1, 2, 3, 4, 7 slices)
  - 144 SMs total with 18 SMs per slice
  - Includes REV1 (media extensions) and REV2 (expanded memory) profiles

### MIG Operations

```go
// Create server with MIG support
server := dgxa100.New()
device, _ := server.DeviceGetHandleByIndex(0)

// Enable MIG mode
device.SetMigMode(1)

// Get available GPU instance profiles
profileInfo, ret := device.GetGpuInstanceProfileInfo(nvml.GPU_INSTANCE_PROFILE_1_SLICE)

// Create GPU instance
gi, ret := device.CreateGpuInstance(&profileInfo)

// Create compute instance within GPU instance
ciProfileInfo, ret := gi.GetComputeInstanceProfileInfo(
    nvml.COMPUTE_INSTANCE_PROFILE_1_SLICE,
    nvml.COMPUTE_INSTANCE_ENGINE_PROFILE_SHARED
)
ci, ret := gi.CreateComputeInstance(&ciProfileInfo)
```

## Testing

The framework includes comprehensive tests covering:

- Server creation and device enumeration
- Device properties and capabilities
- MIG mode operations and lifecycle
- GPU and compute instance management
- Memory and PCI information
- Multi-device scenarios

```bash
# Run all mock tests
go test ./pkg/nvml/mock/...

# Run generation specific tests
go test -v ./pkg/nvml/mock/dgxa100/
go test -v ./pkg/nvml/mock/dgxh100/
go test -v ./pkg/nvml/mock/dgxh200/
go test -v ./pkg/nvml/mock/dgxb200/

# Run specific test
go test -v ./pkg/nvml/mock/dgxa100/ -run TestMIGProfilesExist
go test -v ./pkg/nvml/mock/dgxh100/ -run TestMIGProfilesExist
```

## Extending the Framework

### Adding GPU Variants

Add new configurations to the appropriate file in `internal/shared/gpus/`:

```go
var A100_PCIE_24GB = shared.Config{
    Name:         "NVIDIA A100-PCIE-24GB",
    Architecture: nvml.DEVICE_ARCH_AMPERE,
    Brand:        nvml.BRAND_NVIDIA,
    MemoryMB:     24576, // 24GB
    CudaMajor:    8,
    CudaMinor:    0,
    PciDeviceId:  0x20F010DE,
    MIGProfiles:  a100_24gb_MIGProfiles,
}
```

### Adding GPU Generations

1. **Create new package** (e.g., `dgxb200/`)
2. **Define GPU configurations** in `internal/shared/gpus/b200.go`
3. **Define MIG profiles** with appropriate memory and SM allocations
4. **Implement server and device factory functions**
5. **Add comprehensive tests**

Example structure for B200 generation:

```go
// In internal/shared/gpus/b200.go
var B200_SXM5_180GB = shared.Config{
    Name:         "NVIDIA B200 180GB HBM3e",
    Architecture: nvml.DEVICE_ARCH_BLACKWELL,
    Brand:        nvml.BRAND_NVIDIA,
    MemoryMB:     184320, // 180GB
    CudaMajor:    10,
    CudaMinor:    0,
    PciDeviceId:  0x2B0010DE,
    MIGProfiles:  b200_180gb_MIGProfiles,
}

// In dgxb200/dgxb200.go
func New() *Server {
    return shared.NewServerFromConfig(shared.ServerConfig{
        Config:            gpus.B200_SXM5_180GB,
        GPUCount:          8,
        DriverVersion:     "560.28.03",
        NvmlVersion:       "12.560.28.03",
        CudaDriverVersion: 12060,
    })
}
```

## Backward Compatibility

The framework maintains full backward compatibility:

- All existing `dgxa100.New()`, `dgxh100.New()`, `dgxh200.New()`, `dgxb200.New()` calls continue to work unchanged
- Legacy global variables (`MIGProfiles`, `MIGPlacements`) are preserved for all generations
- Legacy A100 SXM4 40GB config retains "Mock" name prefix for backward compatibility
- All existing tests pass without modification
- All GPU configurations reference `internal/shared/gpus` package for consistency
- Type aliases ensure seamless transition from generation-specific types

## Performance Considerations

- Configurations are defined as static variables (no runtime overhead)
- Device creation uses shared factory (fast)
- MIG profiles are shared between devices of the same type
- Mock functions use direct field access (minimal latency)

## Implementation Notes

- **Thread Safety**: Device implementations include proper mutex usage
- **Memory Management**: No memory leaks in device/instance lifecycle
- **Error Handling**: Proper NVML return codes for all operations
- **Standards Compliance**: Follows official NVML API patterns and behaviors
- **Separation of Concerns**: GPU configs in `internal/shared/gpus`, server logic in package-specific files
