/*
 * Copyright (c) 2021, NVIDIA CORPORATION.  All rights reserved.
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

package nvpci

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/NVIDIA/go-nvlib/pkg/pciids"
)

const (
	// PCIDevicesRoot represents base path for all pci devices under sysfs.
	PCIDevicesRoot = "/sys/bus/pci/devices"
	// PCINvidiaVendorID represents PCI vendor id for NVIDIA.
	PCINvidiaVendorID uint16 = 0x10de
	// PCIVgaControllerClass represents the PCI class for VGA Controllers.
	PCIVgaControllerClass uint32 = 0x030000
	// PCI3dControllerClass represents the PCI class for 3D Graphics accellerators.
	PCI3dControllerClass uint32 = 0x030200
	// PCINvSwitchClass represents the PCI class for NVSwitches.
	PCINvSwitchClass uint32 = 0x068000
	// UnknownDeviceString is the device name to set for devices not found in the PCI database.
	UnknownDeviceString = "UNKNOWN_DEVICE"
	// UnknownClassString is the class name to set for devices not found in the PCI database.
	UnknownClassString = "UNKNOWN_CLASS"
)

// Interface allows us to get a list of all NVIDIA PCI devices.
//
//go:generate moq -rm -fmt=goimports -out nvpci_mock.go . Interface
type Interface interface {
	GetAllDevices() ([]*NvidiaPCIDevice, error)
	Get3DControllers() ([]*NvidiaPCIDevice, error)
	GetVGAControllers() ([]*NvidiaPCIDevice, error)
	GetNVSwitches() ([]*NvidiaPCIDevice, error)
	GetGPUs() ([]*NvidiaPCIDevice, error)
	GetGPUByIndex(int) (*NvidiaPCIDevice, error)
	GetGPUByPciBusID(string) (*NvidiaPCIDevice, error)
	GetNetworkControllers() ([]*NvidiaPCIDevice, error)
	GetPciBridges() ([]*NvidiaPCIDevice, error)
	GetDPUs() ([]*NvidiaPCIDevice, error)
}

// MemoryResources a more human readable handle.
type MemoryResources map[int]*MemoryResource

// ResourceInterface exposes some higher level functions of resources.
type ResourceInterface interface {
	GetTotalAddressableMemory(bool) (uint64, uint64)
}

type nvpci struct {
	logger         logger
	pciDevicesRoot string
	pcidbPath      string
}

var _ Interface = (*nvpci)(nil)
var _ ResourceInterface = (*MemoryResources)(nil)

// SriovInfo indicates whether device is VF/PF for SRIOV capable devices.
// Only one should be set at any given time.
type SriovInfo struct {
	PhysicalFunction *SriovPhysicalFunction
	VirtualFunction  *SriovVirtualFunction
}

// SriovPhysicalFunction stores info about SRIOV physical function.
type SriovPhysicalFunction struct {
	TotalVFs uint64
	NumVFs   uint64
}

// SriovVirtualFunction keeps data about SRIOV virtual function.
type SriovVirtualFunction struct {
	PhysicalFunction *NvidiaPCIDevice
}

func (s *SriovInfo) IsPF() bool {
	return s != nil && s.PhysicalFunction != nil
}

func (s *SriovInfo) IsVF() bool {
	return s != nil && s.VirtualFunction != nil
}

// NvidiaPCIDevice represents a PCI device for an NVIDIA product.
type NvidiaPCIDevice struct {
	Path       string
	Address    string
	Vendor     uint16
	Class      uint32
	ClassName  string
	Device     uint16
	DeviceName string
	Driver     string
	IommuGroup int
	IommuFD    string
	NumaNode   int
	Config     *ConfigSpace
	Resources  MemoryResources
	SriovInfo  SriovInfo
}

// IsVGAController if class == 0x300.
func (d *NvidiaPCIDevice) IsVGAController() bool {
	return d.Class == PCIVgaControllerClass
}

// Is3DController if class == 0x302.
func (d *NvidiaPCIDevice) Is3DController() bool {
	return d.Class == PCI3dControllerClass
}

// IsNVSwitch if class == 0x068.
func (d *NvidiaPCIDevice) IsNVSwitch() bool {
	return d.Class == PCINvSwitchClass
}

// IsGPU either VGA for older cards or 3D for newer.
func (d *NvidiaPCIDevice) IsGPU() bool {
	return d.IsVGAController() || d.Is3DController()
}

// IsResetAvailable some devices can be reset without rebooting,
// check if applicable.
func (d *NvidiaPCIDevice) IsResetAvailable() bool {
	_, err := os.Stat(path.Join(d.Path, "reset"))
	return err == nil
}

// Reset perform a reset to apply a new configuration at HW level.
func (d *NvidiaPCIDevice) Reset() error {
	err := os.WriteFile(path.Join(d.Path, "reset"), []byte("1"), 0)
	if err != nil {
		return fmt.Errorf("unable to write to reset file: %v", err)
	}
	return nil
}

// New interface that allows us to get a list of all NVIDIA PCI devices.
func New(opts ...Option) Interface {
	n := &nvpci{}
	for _, opt := range opts {
		opt(n)
	}
	if n.logger == nil {
		n.logger = &simpleLogger{}
	}
	if n.pciDevicesRoot == "" {
		n.pciDevicesRoot = PCIDevicesRoot
	}
	return n
}

// Option defines a function for passing options to the New() call.
type Option func(*nvpci)

// WithLogger provides an Option to set the logger for the library.
func WithLogger(logger logger) Option {
	return func(n *nvpci) {
		n.logger = logger
	}
}

// WithPCIDevicesRoot provides an Option to set the root path
// for PCI devices on the system.
func WithPCIDevicesRoot(root string) Option {
	return func(n *nvpci) {
		n.pciDevicesRoot = root
	}
}

// WithPCIDatabasePath provides an Option to set the path
// to the pciids database file.
func WithPCIDatabasePath(path string) Option {
	return func(n *nvpci) {
		n.pcidbPath = path
	}
}

// GetAllDevices returns all Nvidia PCI devices on the system.
func (p *nvpci) GetAllDevices() ([]*NvidiaPCIDevice, error) {
	deviceDirs, err := os.ReadDir(p.pciDevicesRoot)
	if err != nil {
		return nil, fmt.Errorf("unable to read PCI bus devices: %v", err)
	}

	var nvdevices []*NvidiaPCIDevice
	// Cache devices for each GetAllDevices invocation to speed things up.
	cache := make(map[string]*NvidiaPCIDevice)
	for _, deviceDir := range deviceDirs {
		deviceAddress := deviceDir.Name()
		nvdevice, err := p.getGPUByPciBusID(deviceAddress, cache)
		if err != nil {
			return nil, fmt.Errorf("error constructing NVIDIA PCI device %s: %v", deviceAddress, err)
		}
		if nvdevice == nil {
			continue
		}
		nvdevices = append(nvdevices, nvdevice)
	}

	addressToID := func(address string) uint64 {
		address = strings.ReplaceAll(address, ":", "")
		address = strings.ReplaceAll(address, ".", "")
		id, _ := strconv.ParseUint(address, 16, 64)
		return id
	}

	sort.Slice(nvdevices, func(i, j int) bool {
		return addressToID(nvdevices[i].Address) < addressToID(nvdevices[j].Address)
	})

	return nvdevices, nil
}

// GetGPUByPciBusID constructs an NvidiaPCIDevice for the specified address (PCI Bus ID).
func (p *nvpci) GetGPUByPciBusID(address string) (*NvidiaPCIDevice, error) {
	// Pass nil as to force reading device information from sysfs.
	return p.getGPUByPciBusID(address, nil)
}

func (p *nvpci) getGPUByPciBusID(address string, cache map[string]*NvidiaPCIDevice) (*NvidiaPCIDevice, error) {
	if cache != nil {
		if pciDevice, exists := cache[address]; exists {
			return pciDevice, nil
		}
	}
	devicePath := filepath.Join(p.pciDevicesRoot, address)

	vendor, err := os.ReadFile(path.Join(devicePath, "vendor"))
	if err != nil {
		return nil, fmt.Errorf("unable to read PCI device vendor id for %s: %v", address, err)
	}
	vendorStr := strings.TrimSpace(string(vendor))
	vendorID, err := strconv.ParseUint(vendorStr, 0, 16)
	if err != nil {
		return nil, fmt.Errorf("unable to convert vendor string to uint16: %v", vendorStr)
	}

	if uint16(vendorID) != PCINvidiaVendorID && uint16(vendorID) != PCIMellanoxVendorID {
		return nil, nil
	}

	class, err := os.ReadFile(path.Join(devicePath, "class"))
	if err != nil {
		return nil, fmt.Errorf("unable to read PCI device class for %s: %v", address, err)
	}
	classStr := strings.TrimSpace(string(class))
	classID, err := strconv.ParseUint(classStr, 0, 32)
	if err != nil {
		return nil, fmt.Errorf("unable to convert class string to uint32: %v", classStr)
	}

	device, err := os.ReadFile(path.Join(devicePath, "device"))
	if err != nil {
		return nil, fmt.Errorf("unable to read PCI device id for %s: %v", address, err)
	}
	deviceStr := strings.TrimSpace(string(device))
	deviceID, err := strconv.ParseUint(deviceStr, 0, 16)
	if err != nil {
		return nil, fmt.Errorf("unable to convert device string to uint16: %v", deviceStr)
	}

	driver, err := getDriver(devicePath)
	if err != nil {
		return nil, fmt.Errorf("unable to detect driver for %s: %w", address, err)
	}

	iommuGroup, err := getIOMMUGroup(devicePath)
	if err != nil {
		return nil, fmt.Errorf("unable to detect IOMMU group for %s: %w", address, err)
	}

	iommuFD, err := getIOMMUFD(devicePath)
	if err != nil {
		// log a warning, do not return an error as this host may not have iommufd configured/supported
		p.logger.Warningf("unable to detect IOMMU FD for %s: %v", address, err)
	}

	numa, err := os.ReadFile(path.Join(devicePath, "numa_node"))
	if err != nil {
		return nil, fmt.Errorf("unable to read PCI NUMA node for %s: %v", address, err)
	}
	numaStr := strings.TrimSpace(string(numa))
	numaNode, err := strconv.ParseInt(numaStr, 0, 64)
	if err != nil {
		return nil, fmt.Errorf("unable to convert NUMA node string to int64: %v", numaNode)
	}

	config := &ConfigSpace{
		Path: path.Join(devicePath, "config"),
	}

	resource, err := os.ReadFile(path.Join(devicePath, "resource"))
	if err != nil {
		return nil, fmt.Errorf("unable to read PCI resource file for %s: %v", address, err)
	}

	resources := make(map[int]*MemoryResource)
	for i, line := range strings.Split(strings.TrimSpace(string(resource)), "\n") {
		values := strings.Split(line, " ")
		if len(values) != 3 {
			return nil, fmt.Errorf("more than 3 entries in line '%d' of resource file", i)
		}

		start, _ := strconv.ParseUint(values[0], 0, 64)
		end, _ := strconv.ParseUint(values[1], 0, 64)
		flags, _ := strconv.ParseUint(values[2], 0, 64)

		if (end - start) != 0 {
			resources[i] = &MemoryResource{
				uintptr(start),
				uintptr(end),
				flags,
				fmt.Sprintf("%s/resource%d", devicePath, i),
			}
		}
	}

	pciDB := pciids.NewDB()

	deviceName, err := pciDB.GetDeviceName(uint16(vendorID), uint16(deviceID))
	if err != nil {
		p.logger.Warningf("unable to get device name: %v\n", err)
		deviceName = UnknownDeviceString
	}
	className, err := pciDB.GetClassName(uint32(classID))
	if err != nil {
		p.logger.Warningf("unable to get class name for device: %v\n", err)
		className = UnknownClassString
	}

	var sriovInfo SriovInfo
	// Device is a virtual function (VF) if "physfn" symlink exists.
	physFnAddress, err := filepath.EvalSymlinks(path.Join(devicePath, "physfn"))
	switch {
	case err == nil:
		physFn, err := p.getGPUByPciBusID(filepath.Base(physFnAddress), cache)
		if err != nil {
			return nil, fmt.Errorf("unable to detect physfn for %s: %v", address, err)
		}
		sriovInfo = SriovInfo{
			VirtualFunction: &SriovVirtualFunction{
				PhysicalFunction: physFn,
			},
		}
	case os.IsNotExist(err):
		sriovInfo, err = p.getSriovInfoForPhysicalFunction(devicePath)
		if err != nil {
			return nil, fmt.Errorf("unable to read SRIOV physical function details for %s: %v", devicePath, err)
		}
	default:
		return nil, fmt.Errorf("unable to read %s: %v", path.Join(devicePath, "physfn"), err)
	}

	nvdevice := &NvidiaPCIDevice{
		Path:       devicePath,
		Address:    address,
		Vendor:     uint16(vendorID),
		Class:      uint32(classID),
		Device:     uint16(deviceID),
		Driver:     driver,
		IommuGroup: int(iommuGroup),
		IommuFD:    iommuFD,
		NumaNode:   int(numaNode),
		Config:     config,
		Resources:  resources,
		DeviceName: deviceName,
		ClassName:  className,
		SriovInfo:  sriovInfo,
	}

	// Cache physical functions only as VF can't be a root device.
	if cache != nil && sriovInfo.IsPF() {
		cache[address] = nvdevice
	}

	return nvdevice, nil
}

// Get3DControllers returns all NVIDIA 3D Controller PCI devices on the system.
func (p *nvpci) Get3DControllers() ([]*NvidiaPCIDevice, error) {
	devices, err := p.GetAllDevices()
	if err != nil {
		return nil, fmt.Errorf("error getting all NVIDIA devices: %v", err)
	}

	var filtered []*NvidiaPCIDevice
	for _, d := range devices {
		if d.Is3DController() {
			filtered = append(filtered, d)
		}
	}

	return filtered, nil
}

// GetVGAControllers returns all NVIDIA VGA Controller PCI devices on the system.
func (p *nvpci) GetVGAControllers() ([]*NvidiaPCIDevice, error) {
	devices, err := p.GetAllDevices()
	if err != nil {
		return nil, fmt.Errorf("error getting all NVIDIA devices: %v", err)
	}

	var filtered []*NvidiaPCIDevice
	for _, d := range devices {
		if d.IsVGAController() {
			filtered = append(filtered, d)
		}
	}

	return filtered, nil
}

// GetNVSwitches returns all NVIDIA NVSwitch PCI devices on the system.
func (p *nvpci) GetNVSwitches() ([]*NvidiaPCIDevice, error) {
	devices, err := p.GetAllDevices()
	if err != nil {
		return nil, fmt.Errorf("error getting all NVIDIA devices: %v", err)
	}

	var filtered []*NvidiaPCIDevice
	for _, d := range devices {
		if d.IsNVSwitch() {
			filtered = append(filtered, d)
		}
	}

	return filtered, nil
}

// GetGPUs returns all NVIDIA GPU devices on the system.
func (p *nvpci) GetGPUs() ([]*NvidiaPCIDevice, error) {
	devices, err := p.GetAllDevices()
	if err != nil {
		return nil, fmt.Errorf("error getting all NVIDIA devices: %v", err)
	}

	var filtered []*NvidiaPCIDevice
	for _, d := range devices {
		if d.IsGPU() && !d.SriovInfo.IsVF() {
			filtered = append(filtered, d)
		}
	}

	return filtered, nil
}

// GetGPUByIndex returns an NVIDIA GPU device at a particular index.
func (p *nvpci) GetGPUByIndex(i int) (*NvidiaPCIDevice, error) {
	gpus, err := p.GetGPUs()
	if err != nil {
		return nil, fmt.Errorf("error getting all gpus: %v", err)
	}

	if i < 0 || i >= len(gpus) {
		return nil, fmt.Errorf("invalid index '%d'", i)
	}

	return gpus[i], nil
}

func (p *nvpci) getSriovInfoForPhysicalFunction(devicePath string) (sriovInfo SriovInfo, err error) {
	totalVfsPath := filepath.Join(devicePath, "sriov_totalvfs")
	numVfsPath := filepath.Join(devicePath, "sriov_numvfs")

	// No file for sriov_totalvfs exists? Not an SRIOV device, return nil
	_, err = os.Stat(totalVfsPath)
	if err != nil && os.IsNotExist(err) {
		return sriovInfo, nil
	}
	sriovTotalVfs, err := os.ReadFile(totalVfsPath)
	if err != nil {
		return sriovInfo, fmt.Errorf("unable to read sriov_totalvfs: %v", err)
	}
	totalVfsStr := strings.TrimSpace(string(sriovTotalVfs))
	totalVfsInt, err := strconv.ParseUint(totalVfsStr, 10, 16)
	if err != nil {
		return sriovInfo, fmt.Errorf("unable to convert sriov_totalvfs to uint64: %v", err)
	}

	sriovNumVfs, err := os.ReadFile(numVfsPath)
	if err != nil {
		return sriovInfo, fmt.Errorf("unable to read sriov_numvfs for: %v", err)
	}
	numVfsStr := strings.TrimSpace(string(sriovNumVfs))
	numVfsInt, err := strconv.ParseUint(numVfsStr, 10, 16)
	if err != nil {
		return sriovInfo, fmt.Errorf("unable to convert sriov_numvfs to uint64: %v", err)
	}

	sriovInfo = SriovInfo{
		PhysicalFunction: &SriovPhysicalFunction{
			TotalVFs: totalVfsInt,
			NumVFs:   numVfsInt,
		},
	}
	return sriovInfo, nil
}

func getDriver(devicePath string) (string, error) {
	driver, err := filepath.EvalSymlinks(path.Join(devicePath, "driver"))
	switch {
	case os.IsNotExist(err):
		return "", nil
	case err == nil:
		return filepath.Base(driver), nil
	}
	return "", err
}

func getIOMMUFD(devicePath string) (string, error) {
	content, err := os.ReadDir(path.Join(devicePath, "vfio-dev"))
	if err != nil {
		return "", err
	}
	for _, c := range content {
		if !c.IsDir() {
			continue
		}
		if strings.HasPrefix(c.Name(), "vfio") {
			return c.Name(), nil
		}
	}
	return "", fmt.Errorf("no iommufd device found")
}

func getIOMMUGroup(devicePath string) (int64, error) {
	var iommuGroup int64
	iommu, err := filepath.EvalSymlinks(path.Join(devicePath, "iommu_group"))
	switch {
	case os.IsNotExist(err):
		return -1, nil
	case err == nil:
		iommuGroupStr := strings.TrimSpace(filepath.Base(iommu))
		iommuGroup, err = strconv.ParseInt(iommuGroupStr, 0, 64)
		if err != nil {
			return 0, fmt.Errorf("unable to convert iommu_group string to int64: %v", iommuGroupStr)
		}
		return iommuGroup, nil
	}
	return 0, err
}
