package ash

import (
	"fmt"
	"unsafe"

	vk "github.com/tomas-mraz/vulkan"
)

// BufferResourceOptions configures VulkanBufferResource creation.
type BufferResourceOptions struct {
	Usage               vk.BufferUsageFlags
	MemoryProperties    vk.MemoryPropertyFlagBits
	EnableDeviceAddress bool
	InitialData         unsafe.Pointer
}

// VulkanBufferResource is a generic Vulkan buffer allocation.
// It owns the VkBuffer, its backing VkDeviceMemory, and optional device address metadata.
type VulkanBufferResource struct {
	device        vk.Device
	Buffer        vk.Buffer
	Memory        vk.DeviceMemory
	Size          uint64
	Usage         vk.BufferUsageFlags
	DeviceAddress vk.DeviceAddress
}

// NewBufferResource creates a generic buffer resource with configurable usage,
// memory properties, initial data, and optional device address support.
func NewBufferResource(device vk.Device, gpu vk.PhysicalDevice, size uint64, opts BufferResourceOptions) (VulkanBufferResource, error) {
	var res VulkanBufferResource
	res.device = device
	res.Size = size
	res.Usage = opts.Usage

	if opts.EnableDeviceAddress {
		res.Usage |= vk.BufferUsageFlags(vk.BufferUsageShaderDeviceAddressBit)
	}

	err := vk.Error(vk.CreateBuffer(device, &vk.BufferCreateInfo{
		SType:       vk.StructureTypeBufferCreateInfo,
		Size:        vk.DeviceSize(size),
		Usage:       res.Usage,
		SharingMode: vk.SharingModeExclusive,
	}, nil, &res.Buffer))
	if err != nil {
		return res, fmt.Errorf("vk.CreateBuffer failed with %s", err)
	}

	var memReqs vk.MemoryRequirements
	vk.GetBufferMemoryRequirements(device, res.Buffer, &memReqs)
	memReqs.Deref()

	var allocFlags vk.MemoryAllocateFlagsInfo
	var allocPNext unsafe.Pointer
	if opts.EnableDeviceAddress {
		allocFlags = vk.MemoryAllocateFlagsInfo{
			SType: vk.StructureTypeMemoryAllocateFlagsInfo,
			Flags: vk.MemoryAllocateFlags(vk.MemoryAllocateDeviceAddressBit),
		}
		allocPNext = unsafe.Pointer(&allocFlags)
	}

	memIdx, ok := vk.FindMemoryTypeIndex(gpu, memReqs.MemoryTypeBits, opts.MemoryProperties)
	if !ok {
		vk.DestroyBuffer(device, res.Buffer, nil)
		var empty VulkanBufferResource
		return empty, fmt.Errorf("vk.FindMemoryTypeIndex failed for requested buffer memory properties")
	}

	err = vk.Error(vk.AllocateMemory(device, &vk.MemoryAllocateInfo{
		SType:           vk.StructureTypeMemoryAllocateInfo,
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memIdx,
		PNext:           allocPNext,
	}, nil, &res.Memory))
	if err != nil {
		vk.DestroyBuffer(device, res.Buffer, nil)
		return res, fmt.Errorf("vk.AllocateMemory failed with %s", err)
	}

	err = vk.Error(vk.BindBufferMemory(device, res.Buffer, res.Memory, 0))
	if err != nil {
		vk.FreeMemory(device, res.Memory, nil)
		vk.DestroyBuffer(device, res.Buffer, nil)
		return res, fmt.Errorf("vk.BindBufferMemory failed with %s", err)
	}

	if opts.InitialData != nil {
		if vk.MemoryPropertyFlags(opts.MemoryProperties)&vk.MemoryPropertyFlags(vk.MemoryPropertyHostVisibleBit) == 0 {
			res.Destroy()
			var empty VulkanBufferResource
			return empty, fmt.Errorf("initial buffer data requires host-visible memory")
		}
		var mapped unsafe.Pointer
		err = vk.Error(vk.MapMemory(device, res.Memory, 0, vk.DeviceSize(size), 0, &mapped))
		if err != nil {
			res.Destroy()
			var empty VulkanBufferResource
			return empty, fmt.Errorf("vk.MapMemory failed with %s", err)
		}
		vk.Memcopy(mapped, unsafe.Slice((*byte)(opts.InitialData), int(size)))
		vk.UnmapMemory(device, res.Memory)
	}

	if opts.EnableDeviceAddress {
		res.DeviceAddress = GetBufferDeviceAddress(device, res.Buffer)
	}

	return res, nil
}

// NewBufferResourceHostVisible creates a host-visible/coherent buffer.
func NewBufferResourceHostVisible(device vk.Device, gpu vk.PhysicalDevice, usage vk.BufferUsageFlags, size uint64, data unsafe.Pointer, enableDeviceAddress bool) (VulkanBufferResource, error) {
	return NewBufferResource(device, gpu, size, BufferResourceOptions{
		Usage:               usage,
		MemoryProperties:    vk.MemoryPropertyHostVisibleBit | vk.MemoryPropertyHostCoherentBit,
		EnableDeviceAddress: enableDeviceAddress,
		InitialData:         data,
	})
}

// NewBufferResourceDeviceLocal creates a device-local buffer.
func NewBufferResourceDeviceLocal(device vk.Device, gpu vk.PhysicalDevice, usage vk.BufferUsageFlags, size uint64, enableDeviceAddress bool) (VulkanBufferResource, error) {
	return NewBufferResource(device, gpu, size, BufferResourceOptions{
		Usage:               usage,
		MemoryProperties:    vk.MemoryPropertyDeviceLocalBit,
		EnableDeviceAddress: enableDeviceAddress,
	})
}

// Update overwrites the entire buffer contents. The resource must use host-visible memory.
func (r *VulkanBufferResource) Update(data []byte) error {
	if uint64(len(data)) > r.Size {
		return fmt.Errorf("buffer update too large: got %d bytes for buffer size %d", len(data), r.Size)
	}
	var mapped unsafe.Pointer
	if err := vk.Error(vk.MapMemory(r.device, r.Memory, 0, vk.DeviceSize(len(data)), 0, &mapped)); err != nil {
		return fmt.Errorf("vk.MapMemory failed with %s", err)
	}
	vk.Memcopy(mapped, data)
	vk.UnmapMemory(r.device, r.Memory)
	return nil
}

func (r *VulkanBufferResource) Destroy() {
	if r == nil {
		return
	}
	if r.Buffer != vk.NullBuffer {
		vk.DestroyBuffer(r.device, r.Buffer, nil)
		r.Buffer = vk.NullBuffer
	}
	if r.Memory != vk.NullDeviceMemory {
		vk.FreeMemory(r.device, r.Memory, nil)
		r.Memory = vk.NullDeviceMemory
	}
	r.DeviceAddress = 0
	r.Size = 0
}
