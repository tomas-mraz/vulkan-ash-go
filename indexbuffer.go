package asch

import (
	"fmt"
	"unsafe"

	vk "github.com/tomas-mraz/vulkan"
)

// VulkanIndexBufferInfo manages an index buffer for indexed drawing.
type VulkanIndexBufferInfo struct {
	device vk.Device
	buffer vk.Buffer
	mem    vk.DeviceMemory
	count  uint32
}

// NewIndexBuffer creates an index buffer from uint16 index data.
func NewIndexBuffer(device vk.Device, gpu vk.PhysicalDevice, indices []uint16) (VulkanIndexBufferInfo, error) {
	dataSize := len(indices) * 2
	data := unsafe.Slice((*byte)(unsafe.Pointer(&indices[0])), dataSize)
	return newIndexBufferFromBytes(device, gpu, data, uint32(len(indices)))
}

// NewIndexBuffer32 creates an index buffer from uint32 index data.
func NewIndexBuffer32(device vk.Device, gpu vk.PhysicalDevice, indices []uint32) (VulkanIndexBufferInfo, error) {
	dataSize := len(indices) * 4
	data := unsafe.Slice((*byte)(unsafe.Pointer(&indices[0])), dataSize)
	return newIndexBufferFromBytes(device, gpu, data, uint32(len(indices)))
}

func newIndexBufferFromBytes(device vk.Device, gpu vk.PhysicalDevice, data []byte, count uint32) (VulkanIndexBufferInfo, error) {
	var ib VulkanIndexBufferInfo
	ib.device = device
	ib.count = count

	err := vk.Error(vk.CreateBuffer(device, &vk.BufferCreateInfo{
		SType:                 vk.StructureTypeBufferCreateInfo,
		Size:                  vk.DeviceSize(len(data)),
		Usage:                 vk.BufferUsageFlags(vk.BufferUsageIndexBufferBit),
		SharingMode:           vk.SharingModeExclusive,
		QueueFamilyIndexCount: 1,
		PQueueFamilyIndices:   []uint32{0},
	}, nil, &ib.buffer))
	if err != nil {
		return ib, fmt.Errorf("vk.CreateBuffer (index) failed with %s", err)
	}

	var memReq vk.MemoryRequirements
	vk.GetBufferMemoryRequirements(device, ib.buffer, &memReq)
	memReq.Deref()

	memIdx, _ := vk.FindMemoryTypeIndex(gpu, memReq.MemoryTypeBits, vk.MemoryPropertyHostVisibleBit)
	err = vk.Error(vk.AllocateMemory(device, &vk.MemoryAllocateInfo{
		SType:           vk.StructureTypeMemoryAllocateInfo,
		AllocationSize:  memReq.Size,
		MemoryTypeIndex: memIdx,
	}, nil, &ib.mem))
	if err != nil {
		return ib, fmt.Errorf("vk.AllocateMemory (index) failed with %s", err)
	}

	var ptr unsafe.Pointer
	vk.MapMemory(device, ib.mem, 0, vk.DeviceSize(len(data)), 0, &ptr)
	vk.Memcopy(ptr, data)
	vk.UnmapMemory(device, ib.mem)

	err = vk.Error(vk.BindBufferMemory(device, ib.buffer, ib.mem, 0))
	if err != nil {
		return ib, fmt.Errorf("vk.BindBufferMemory (index) failed with %s", err)
	}
	return ib, nil
}

func (ib *VulkanIndexBufferInfo) GetBuffer() vk.Buffer {
	return ib.buffer
}

// GetCount returns the number of indices.
func (ib *VulkanIndexBufferInfo) GetCount() uint32 {
	return ib.count
}

func (ib *VulkanIndexBufferInfo) Destroy() {
	if ib == nil {
		return
	}
	vk.FreeMemory(ib.device, ib.mem, nil)
	vk.DestroyBuffer(ib.device, ib.buffer, nil)
}
