package ash

import (
	"fmt"
	"log"
	"unsafe"

	vk "github.com/tomas-mraz/vulkan"
)

type VulkanBufferInfo struct {
	device        vk.Device
	vertexBuffers []vk.Buffer
	deviceMemory  vk.DeviceMemory
}

// NewBuffer creates a vertex buffer with default triangle data.
func NewBuffer(device vk.Device, gpu vk.PhysicalDevice) (VulkanBufferInfo, error) {

	// Phase 1: vk.CreateBuffer
	//			create the triangle vertex buffer

	vertexData := ArrayFloat32([]float32{
		-1, -1, 0,
		1, -1, 0,
		0, 1, 0,
	})
	return newBufferFromData(device, gpu, vertexData.Data(), vertexData.Sizeof())
}

// NewBufferWithData creates a vertex buffer with custom float32 vertex data.
func NewBufferWithData(device vk.Device, gpu vk.PhysicalDevice, vertices []float32) (VulkanBufferInfo, error) {
	dataBytes := unsafe.Slice((*byte)(unsafe.Pointer(&vertices[0])), len(vertices)*4)
	return newBufferFromData(device, gpu, dataBytes, len(vertices)*4)
}

func newBufferFromData(device vk.Device, gpu vk.PhysicalDevice, data []byte, dataSize int) (VulkanBufferInfo, error) {
	queueFamilyIdx := []uint32{0}
	bufferCreateInfo := vk.BufferCreateInfo{
		SType:                 vk.StructureTypeBufferCreateInfo,
		Size:                  vk.DeviceSize(dataSize),
		Usage:                 vk.BufferUsageFlags(vk.BufferUsageVertexBufferBit),
		SharingMode:           vk.SharingModeExclusive,
		QueueFamilyIndexCount: 1,
		PQueueFamilyIndices:   queueFamilyIdx,
	}
	buffer := VulkanBufferInfo{
		vertexBuffers: make([]vk.Buffer, 1),
	}
	err := vk.Error(vk.CreateBuffer(device, &bufferCreateInfo, nil, &buffer.vertexBuffers[0]))
	if err != nil {
		err = fmt.Errorf("vk.CreateBuffer failed with %s", err)
		return buffer, err
	}

	// Phase 2: vk.GetBufferMemoryRequirements
	//			vk.FindMemoryTypeIndex
	// 			assign a proper memory type for that buffer

	var memReq vk.MemoryRequirements
	vk.GetBufferMemoryRequirements(device, buffer.DefaultVertexBuffer(), &memReq)
	memReq.Deref()
	allocInfo := vk.MemoryAllocateInfo{
		SType:           vk.StructureTypeMemoryAllocateInfo,
		AllocationSize:  memReq.Size,
		MemoryTypeIndex: 0, // see below
	}
	allocInfo.MemoryTypeIndex, _ = vk.FindMemoryTypeIndex(gpu, memReq.MemoryTypeBits, vk.MemoryPropertyHostVisibleBit) //FIXME

	// Phase 3: vk.AllocateMemory
	//			vk.MapMemory
	//			vk.MemCopyFloat32
	//			vk.UnmapMemory
	// 			allocate and map memory for that buffer

	var deviceMemory vk.DeviceMemory
	err = vk.Error(vk.AllocateMemory(device, &allocInfo, nil, &deviceMemory))
	if err != nil {
		err = fmt.Errorf("vk.AllocateMemory failed with %s", err)
		return buffer, err
	}
	var ptr unsafe.Pointer
	vk.MapMemory(device, deviceMemory, 0, vk.DeviceSize(dataSize), 0, &ptr)
	n := vk.Memcopy(ptr, data)
	if n != dataSize {
		log.Println("[WARN] failed to copy vertex buffer data")
	}
	vk.UnmapMemory(device, deviceMemory)
	buffer.deviceMemory = deviceMemory

	// Phase 4: vk.BindBufferMemory
	//			copy vertex data and bind buffer

	err = vk.Error(vk.BindBufferMemory(device, buffer.DefaultVertexBuffer(), deviceMemory, 0))
	if err != nil {
		err = fmt.Errorf("vk.BindBufferMemory failed with %s", err)
		return buffer, err
	}
	buffer.device = device
	return buffer, err
}

func (buf *VulkanBufferInfo) Destroy() {
	for i := range buf.vertexBuffers {
		vk.DestroyBuffer(buf.device, buf.vertexBuffers[i], nil)
	}
}

func (buf *VulkanBufferInfo) DefaultVertexBuffer() vk.Buffer {
	return buf.vertexBuffers[0]
}

func (buf *VulkanBufferInfo) GetDeviceMemory() vk.DeviceMemory {
	return buf.deviceMemory
}
