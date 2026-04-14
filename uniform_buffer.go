package ash

import (
	"fmt"
	"unsafe"

	vk "github.com/tomas-mraz/vulkan"
)

// UniformBuffers manages a set of uniform buffers (typically one per swapchain image).
type UniformBuffers struct {
	device   vk.Device
	buffers  []vk.Buffer
	memories []vk.DeviceMemory
	size     int
}

// NewUniformBuffers creates count uniform buffers of the given byte size.
// Typically count matches the swapchain image count.
func NewUniformBuffers(device vk.Device, gpu vk.PhysicalDevice, count uint32, dataSize int) (UniformBuffers, error) {
	u := UniformBuffers{
		device:   device,
		buffers:  make([]vk.Buffer, count),
		memories: make([]vk.DeviceMemory, count),
		size:     dataSize,
	}

	for i := uint32(0); i < count; i++ {
		err := vk.Error(vk.CreateBuffer(device, &vk.BufferCreateInfo{
			SType:       vk.StructureTypeBufferCreateInfo,
			Size:        vk.DeviceSize(dataSize),
			Usage:       vk.BufferUsageFlags(vk.BufferUsageUniformBufferBit),
			SharingMode: vk.SharingModeExclusive,
		}, nil, &u.buffers[i]))
		if err != nil {
			return u, fmt.Errorf("vk.CreateBuffer (uniform %d) failed with %s", i, err)
		}

		var memReq vk.MemoryRequirements
		vk.GetBufferMemoryRequirements(device, u.buffers[i], &memReq)
		memReq.Deref()

		memIdx, _ := FindMemoryTypeIndex(gpu, memReq.MemoryTypeBits, vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit)
		err = vk.Error(vk.AllocateMemory(device, &vk.MemoryAllocateInfo{
			SType:           vk.StructureTypeMemoryAllocateInfo,
			AllocationSize:  memReq.Size,
			MemoryTypeIndex: memIdx,
		}, nil, &u.memories[i]))
		if err != nil {
			return u, fmt.Errorf("vk.AllocateMemory (uniform %d) failed with %s", i, err)
		}
		vk.BindBufferMemory(device, u.buffers[i], u.memories[i], 0)
	}
	return u, nil
}

// Update writes data into the uniform buffer at the given index.
func (u *UniformBuffers) Update(index uint32, data []byte) {
	var pData unsafe.Pointer
	vk.MapMemory(u.device, u.memories[index], 0, vk.DeviceSize(len(data)), 0, &pData)
	vk.Memcopy(pData, data)
	vk.UnmapMemory(u.device, u.memories[index])
}

// GetBuffer returns the buffer at the given index.
func (u *UniformBuffers) GetBuffer(index uint32) vk.Buffer {
	return u.buffers[index]
}

// GetBuffers returns all buffers.
func (u *UniformBuffers) GetBuffers() []vk.Buffer {
	return u.buffers
}

// GetSize returns the byte size of each uniform buffer.
func (u *UniformBuffers) GetSize() int {
	return u.size
}

// Destroy frees all uniform buffers and their memory.
func (u *UniformBuffers) Destroy() {
	if u == nil {
		return
	}
	for i := range u.buffers {
		vk.FreeMemory(u.device, u.memories[i], nil)
		vk.DestroyBuffer(u.device, u.buffers[i], nil)
	}
}
