package asch

import (
	"fmt"
	"unsafe"

	vk "github.com/tomas-mraz/vulkan"
)

// VulkanTextureInfo manages a texture image, memory, view, and sampler.
type VulkanTextureInfo struct {
	device  vk.Device
	image   vk.Image
	mem     vk.DeviceMemory
	view    vk.ImageView
	sampler vk.Sampler
}

// NewTexture creates a texture from RGBA pixel data with a nearest-filter sampler.
// After creation, call TransitionImageLayout to transition from PreInitialized to ShaderReadOnlyOptimal.
func NewTexture(device vk.Device, gpu vk.PhysicalDevice, width, height uint32, rgbaPixels []byte) (VulkanTextureInfo, error) {
	var t VulkanTextureInfo
	t.device = device

	// Create image
	err := vk.Error(vk.CreateImage(device, &vk.ImageCreateInfo{
		SType:         vk.StructureTypeImageCreateInfo,
		ImageType:     vk.ImageType2d,
		Format:        vk.FormatR8g8b8a8Unorm,
		Extent:        vk.Extent3D{Width: width, Height: height, Depth: 1},
		MipLevels:     1,
		ArrayLayers:   1,
		Samples:       vk.SampleCount1Bit,
		Tiling:        vk.ImageTilingLinear,
		Usage:         vk.ImageUsageFlags(vk.ImageUsageSampledBit),
		InitialLayout: vk.ImageLayoutPreinitialized,
	}, nil, &t.image))
	if err != nil {
		return t, fmt.Errorf("vk.CreateImage (texture) failed with %s", err)
	}

	// Allocate host-visible memory
	var memReq vk.MemoryRequirements
	vk.GetImageMemoryRequirements(device, t.image, &memReq)
	memReq.Deref()

	memIdx, _ := vk.FindMemoryTypeIndex(gpu, memReq.MemoryTypeBits,
		vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit)
	err = vk.Error(vk.AllocateMemory(device, &vk.MemoryAllocateInfo{
		SType:           vk.StructureTypeMemoryAllocateInfo,
		AllocationSize:  memReq.Size,
		MemoryTypeIndex: memIdx,
	}, nil, &t.mem))
	if err != nil {
		return t, fmt.Errorf("vk.AllocateMemory (texture) failed with %s", err)
	}
	vk.BindImageMemory(device, t.image, t.mem, 0)

	// Upload pixel data
	var pData unsafe.Pointer
	vk.MapMemory(device, t.mem, 0, vk.DeviceSize(len(rgbaPixels)), 0, &pData)
	vk.Memcopy(pData, rgbaPixels)
	vk.UnmapMemory(device, t.mem)

	// Create sampler
	err = vk.Error(vk.CreateSampler(device, &vk.SamplerCreateInfo{
		SType:        vk.StructureTypeSamplerCreateInfo,
		MagFilter:    vk.FilterNearest,
		MinFilter:    vk.FilterNearest,
		MipmapMode:   vk.SamplerMipmapModeNearest,
		AddressModeU: vk.SamplerAddressModeClampToEdge,
		AddressModeV: vk.SamplerAddressModeClampToEdge,
		AddressModeW: vk.SamplerAddressModeClampToEdge,
		MaxAnisotropy: 1,
		CompareOp:    vk.CompareOpNever,
		BorderColor:  vk.BorderColorFloatOpaqueWhite,
	}, nil, &t.sampler))
	if err != nil {
		return t, fmt.Errorf("vk.CreateSampler failed with %s", err)
	}

	// Create image view
	err = vk.Error(vk.CreateImageView(device, &vk.ImageViewCreateInfo{
		SType:    vk.StructureTypeImageViewCreateInfo,
		Image:    t.image,
		ViewType: vk.ImageViewType2d,
		Format:   vk.FormatR8g8b8a8Unorm,
		Components: vk.ComponentMapping{
			R: vk.ComponentSwizzleR,
			G: vk.ComponentSwizzleG,
			B: vk.ComponentSwizzleB,
			A: vk.ComponentSwizzleA,
		},
		SubresourceRange: vk.ImageSubresourceRange{
			AspectMask: vk.ImageAspectFlags(vk.ImageAspectColorBit),
			LevelCount: 1,
			LayerCount: 1,
		},
	}, nil, &t.view))
	if err != nil {
		return t, fmt.Errorf("vk.CreateImageView (texture) failed with %s", err)
	}

	return t, nil
}

func (t *VulkanTextureInfo) GetView() vk.ImageView {
	return t.view
}

func (t *VulkanTextureInfo) GetSampler() vk.Sampler {
	return t.sampler
}

func (t *VulkanTextureInfo) GetImage() vk.Image {
	return t.image
}

func (t *VulkanTextureInfo) Destroy() {
	if t == nil {
		return
	}
	vk.DestroySampler(t.device, t.sampler, nil)
	vk.DestroyImageView(t.device, t.view, nil)
	vk.FreeMemory(t.device, t.mem, nil)
	vk.DestroyImage(t.device, t.image, nil)
}

// TransitionImageLayout performs a one-time image layout transition using a temporary command buffer.
func TransitionImageLayout(device vk.Device, queue vk.Queue, cmdPool vk.CommandPool,
	image vk.Image, oldLayout, newLayout vk.ImageLayout) {

	cmds := make([]vk.CommandBuffer, 1)
	vk.AllocateCommandBuffers(device, &vk.CommandBufferAllocateInfo{
		SType:              vk.StructureTypeCommandBufferAllocateInfo,
		CommandPool:        cmdPool,
		Level:              vk.CommandBufferLevelPrimary,
		CommandBufferCount: 1,
	}, cmds)
	cmd := cmds[0]

	vk.BeginCommandBuffer(cmd, &vk.CommandBufferBeginInfo{
		SType: vk.StructureTypeCommandBufferBeginInfo,
		Flags: vk.CommandBufferUsageFlags(vk.CommandBufferUsageOneTimeSubmitBit),
	})

	var srcAccess, dstAccess vk.AccessFlags
	var srcStage, dstStage vk.PipelineStageFlags

	switch {
	case oldLayout == vk.ImageLayoutPreinitialized && newLayout == vk.ImageLayoutShaderReadOnlyOptimal:
		srcAccess = vk.AccessFlags(vk.AccessHostWriteBit)
		dstAccess = vk.AccessFlags(vk.AccessShaderReadBit)
		srcStage = vk.PipelineStageFlags(vk.PipelineStageHostBit)
		dstStage = vk.PipelineStageFlags(vk.PipelineStageFragmentShaderBit)
	case oldLayout == vk.ImageLayoutUndefined && newLayout == vk.ImageLayoutTransferDstOptimal:
		dstAccess = vk.AccessFlags(vk.AccessTransferWriteBit)
		srcStage = vk.PipelineStageFlags(vk.PipelineStageTopOfPipeBit)
		dstStage = vk.PipelineStageFlags(vk.PipelineStageTransferBit)
	case oldLayout == vk.ImageLayoutTransferDstOptimal && newLayout == vk.ImageLayoutShaderReadOnlyOptimal:
		srcAccess = vk.AccessFlags(vk.AccessTransferWriteBit)
		dstAccess = vk.AccessFlags(vk.AccessShaderReadBit)
		srcStage = vk.PipelineStageFlags(vk.PipelineStageTransferBit)
		dstStage = vk.PipelineStageFlags(vk.PipelineStageFragmentShaderBit)
	}

	vk.CmdPipelineBarrier(cmd, srcStage, dstStage, 0, 0, nil, 0, nil, 1, []vk.ImageMemoryBarrier{{
		SType:         vk.StructureTypeImageMemoryBarrier,
		SrcAccessMask: srcAccess,
		DstAccessMask: dstAccess,
		OldLayout:     oldLayout,
		NewLayout:     newLayout,
		Image:         image,
		SubresourceRange: vk.ImageSubresourceRange{
			AspectMask: vk.ImageAspectFlags(vk.ImageAspectColorBit),
			LevelCount: 1,
			LayerCount: 1,
		},
	}})

	vk.EndCommandBuffer(cmd)

	vk.QueueSubmit(queue, 1, []vk.SubmitInfo{{
		SType:              vk.StructureTypeSubmitInfo,
		CommandBufferCount: 1,
		PCommandBuffers:    []vk.CommandBuffer{cmd},
	}}, vk.NullFence)
	vk.QueueWaitIdle(queue)
	vk.FreeCommandBuffers(device, cmdPool, 1, []vk.CommandBuffer{cmd})
}
