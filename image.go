package ash

import (
	"fmt"

	vk "github.com/tomas-mraz/vulkan"
)

// VulkanStorageImageInfo manages a storage image for ray tracing or compute output.
type VulkanStorageImageInfo struct {
	device vk.Device
	image  vk.Image
	mem    vk.DeviceMemory
	view   vk.ImageView
}

// NewStorageImage creates a device-local storage image and transitions it to General layout.
func NewStorageImage(device vk.Device, gpu vk.PhysicalDevice, queue vk.Queue, cmdPool vk.CommandPool, width, height uint32, format vk.Format) (VulkanStorageImageInfo, error) {
	var s VulkanStorageImageInfo
	s.device = device

	// Create image
	err := vk.Error(vk.CreateImage(device, &vk.ImageCreateInfo{
		SType:       vk.StructureTypeImageCreateInfo,
		ImageType:   vk.ImageType2d,
		Format:      format,
		Extent:      vk.Extent3D{Width: width, Height: height, Depth: 1},
		MipLevels:   1,
		ArrayLayers: 1,
		Samples:     vk.SampleCount1Bit,
		Tiling:      vk.ImageTilingOptimal,
		Usage:       vk.ImageUsageFlags(vk.ImageUsageTransferSrcBit | vk.ImageUsageStorageBit),
	}, nil, &s.image))
	if err != nil {
		return s, fmt.Errorf("vk.CreateImage (storage) failed with %s", err)
	}

	// Allocate device-local memory
	var memReqs vk.MemoryRequirements
	vk.GetImageMemoryRequirements(device, s.image, &memReqs)
	memReqs.Deref()

	memIdx, _ := vk.FindMemoryTypeIndex(gpu, memReqs.MemoryTypeBits, vk.MemoryPropertyDeviceLocalBit)
	err = vk.Error(vk.AllocateMemory(device, &vk.MemoryAllocateInfo{
		SType:           vk.StructureTypeMemoryAllocateInfo,
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memIdx,
	}, nil, &s.mem))
	if err != nil {
		vk.DestroyImage(device, s.image, nil)
		return s, fmt.Errorf("vk.AllocateMemory (storage) failed with %s", err)
	}
	vk.BindImageMemory(device, s.image, s.mem, 0)

	// Create image view
	err = vk.Error(vk.CreateImageView(device, &vk.ImageViewCreateInfo{
		SType:    vk.StructureTypeImageViewCreateInfo,
		Image:    s.image,
		ViewType: vk.ImageViewType2d,
		Format:   format,
		SubresourceRange: vk.ImageSubresourceRange{
			AspectMask: vk.ImageAspectFlags(vk.ImageAspectColorBit),
			LevelCount: 1,
			LayerCount: 1,
		},
	}, nil, &s.view))
	if err != nil {
		vk.FreeMemory(device, s.mem, nil)
		vk.DestroyImage(device, s.image, nil)
		return s, fmt.Errorf("vk.CreateImageView (storage) failed with %s", err)
	}

	// Transition to General layout
	TransitionImageLayout(device, queue, cmdPool, s.image, vk.ImageLayoutUndefined, vk.ImageLayoutGeneral)

	return s, nil
}

func (s *VulkanStorageImageInfo) GetImage() vk.Image {
	return s.image
}

func (s *VulkanStorageImageInfo) GetView() vk.ImageView {
	return s.view
}

func (s *VulkanStorageImageInfo) Destroy() {
	if s == nil {
		return
	}
	vk.DestroyImageView(s.device, s.view, nil)
	vk.FreeMemory(s.device, s.mem, nil)
	vk.DestroyImage(s.device, s.image, nil)
}
