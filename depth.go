package asch

import (
	"fmt"

	vk "github.com/tomas-mraz/vulkan"
)

// VulkanDepthInfo manages a depth buffer image, memory, and view.
type VulkanDepthInfo struct {
	device vk.Device
	image  vk.Image
	mem    vk.DeviceMemory
	view   vk.ImageView
	format vk.Format
}

// NewDepthBuffer creates a depth buffer with the given dimensions and format.
func NewDepthBuffer(device vk.Device, gpu vk.PhysicalDevice, width, height uint32, format vk.Format) (VulkanDepthInfo, error) {
	var d VulkanDepthInfo
	d.device = device
	d.format = format

	// Create depth image
	err := vk.Error(vk.CreateImage(device, &vk.ImageCreateInfo{
		SType:       vk.StructureTypeImageCreateInfo,
		ImageType:   vk.ImageType2d,
		Format:      format,
		Extent:      vk.Extent3D{Width: width, Height: height, Depth: 1},
		MipLevels:   1,
		ArrayLayers: 1,
		Samples:     vk.SampleCount1Bit,
		Tiling:      vk.ImageTilingOptimal,
		Usage:       vk.ImageUsageFlags(vk.ImageUsageDepthStencilAttachmentBit),
	}, nil, &d.image))
	if err != nil {
		return d, fmt.Errorf("vk.CreateImage (depth) failed with %s", err)
	}

	// Allocate device-local memory
	var memReq vk.MemoryRequirements
	vk.GetImageMemoryRequirements(device, d.image, &memReq)
	memReq.Deref()

	memIdx, _ := vk.FindMemoryTypeIndex(gpu, memReq.MemoryTypeBits, vk.MemoryPropertyDeviceLocalBit)
	err = vk.Error(vk.AllocateMemory(device, &vk.MemoryAllocateInfo{
		SType:           vk.StructureTypeMemoryAllocateInfo,
		AllocationSize:  memReq.Size,
		MemoryTypeIndex: memIdx,
	}, nil, &d.mem))
	if err != nil {
		return d, fmt.Errorf("vk.AllocateMemory (depth) failed with %s", err)
	}
	err = vk.Error(vk.BindImageMemory(device, d.image, d.mem, 0))
	if err != nil {
		return d, fmt.Errorf("vk.BindImageMemory (depth) failed with %s", err)
	}

	// Create image view
	err = vk.Error(vk.CreateImageView(device, &vk.ImageViewCreateInfo{
		SType:    vk.StructureTypeImageViewCreateInfo,
		Image:    d.image,
		ViewType: vk.ImageViewType2d,
		Format:   format,
		SubresourceRange: vk.ImageSubresourceRange{
			AspectMask: vk.ImageAspectFlags(vk.ImageAspectDepthBit),
			LevelCount: 1,
			LayerCount: 1,
		},
	}, nil, &d.view))
	if err != nil {
		return d, fmt.Errorf("vk.CreateImageView (depth) failed with %s", err)
	}

	return d, nil
}

func (d *VulkanDepthInfo) GetView() vk.ImageView {
	return d.view
}

func (d *VulkanDepthInfo) GetFormat() vk.Format {
	return d.format
}

func (d *VulkanDepthInfo) Destroy() {
	if d == nil {
		return
	}
	vk.DestroyImageView(d.device, d.view, nil)
	vk.FreeMemory(d.device, d.mem, nil)
	vk.DestroyImage(d.device, d.image, nil)
}
