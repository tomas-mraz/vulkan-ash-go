package asch

import (
	"fmt"
	"unsafe"

	vk "github.com/tomas-mraz/vulkan"
)

// NewBufferWithDeviceAddress creates a host-visible buffer with device address capability.
// If data is non-nil, the buffer is immediately filled with the provided bytes.
func NewBufferWithDeviceAddress(device vk.Device, gpu vk.PhysicalDevice, usage vk.BufferUsageFlags, size uint64, data unsafe.Pointer) (vk.Buffer, vk.DeviceMemory, error) {
	var buf vk.Buffer
	if err := vk.Error(vk.CreateBuffer(device, &vk.BufferCreateInfo{
		SType: vk.StructureTypeBufferCreateInfo,
		Size:  vk.DeviceSize(size),
		Usage: usage,
	}, nil, &buf)); err != nil {
		return buf, vk.NullDeviceMemory, fmt.Errorf("vk.CreateBuffer failed with %s", err)
	}

	var memReqs vk.MemoryRequirements
	vk.GetBufferMemoryRequirements(device, buf, &memReqs)
	memReqs.Deref()

	allocFlags := vk.MemoryAllocateFlagsInfo{
		SType: vk.StructureTypeMemoryAllocateFlagsInfo,
		Flags: vk.MemoryAllocateFlags(vk.MemoryAllocateDeviceAddressBit),
	}
	memIdx, _ := vk.FindMemoryTypeIndex(gpu, memReqs.MemoryTypeBits,
		vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit)

	var mem vk.DeviceMemory
	if err := vk.Error(vk.AllocateMemory(device, &vk.MemoryAllocateInfo{
		SType:           vk.StructureTypeMemoryAllocateInfo,
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memIdx,
		PNext:           unsafe.Pointer(&allocFlags),
	}, nil, &mem)); err != nil {
		vk.DestroyBuffer(device, buf, nil)
		return buf, mem, fmt.Errorf("vk.AllocateMemory failed with %s", err)
	}
	vk.BindBufferMemory(device, buf, mem, 0)

	if data != nil {
		var mapped unsafe.Pointer
		vk.MapMemory(device, mem, 0, vk.DeviceSize(size), 0, &mapped)
		vk.Memcopy(mapped, unsafe.Slice((*byte)(data), int(size)))
		vk.UnmapMemory(device, mem)
	}
	return buf, mem, nil
}

// NewDeviceLocalBuffer creates a device-local buffer with device address capability.
func NewDeviceLocalBuffer(device vk.Device, gpu vk.PhysicalDevice, usage vk.BufferUsageFlags, size uint64) (vk.Buffer, vk.DeviceMemory, error) {
	var buf vk.Buffer
	if err := vk.Error(vk.CreateBuffer(device, &vk.BufferCreateInfo{
		SType: vk.StructureTypeBufferCreateInfo,
		Size:  vk.DeviceSize(size),
		Usage: usage,
	}, nil, &buf)); err != nil {
		return buf, vk.NullDeviceMemory, fmt.Errorf("vk.CreateBuffer failed with %s", err)
	}

	var memReqs vk.MemoryRequirements
	vk.GetBufferMemoryRequirements(device, buf, &memReqs)
	memReqs.Deref()

	allocFlags := vk.MemoryAllocateFlagsInfo{
		SType: vk.StructureTypeMemoryAllocateFlagsInfo,
		Flags: vk.MemoryAllocateFlags(vk.MemoryAllocateDeviceAddressBit),
	}
	memIdx, _ := vk.FindMemoryTypeIndex(gpu, memReqs.MemoryTypeBits, vk.MemoryPropertyDeviceLocalBit)

	var mem vk.DeviceMemory
	if err := vk.Error(vk.AllocateMemory(device, &vk.MemoryAllocateInfo{
		SType:           vk.StructureTypeMemoryAllocateInfo,
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memIdx,
		PNext:           unsafe.Pointer(&allocFlags),
	}, nil, &mem)); err != nil {
		vk.DestroyBuffer(device, buf, nil)
		return buf, mem, fmt.Errorf("vk.AllocateMemory failed with %s", err)
	}
	vk.BindBufferMemory(device, buf, mem, 0)
	return buf, mem, nil
}

// GetBufferDeviceAddress returns the device address of a buffer.
func GetBufferDeviceAddress(device vk.Device, buf vk.Buffer) vk.DeviceAddress {
	return vk.GetBufferDeviceAddress(device, &vk.BufferDeviceAddressInfo{
		SType:  vk.StructureTypeBufferDeviceAddressInfo,
		Buffer: buf,
	})
}
