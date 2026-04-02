package ash

import (
	"fmt"
	"unsafe"

	vk "github.com/tomas-mraz/vulkan"
)

// VulkanImageResource is a generic Device image allocation.
// It owns the VkImage, its backing VkDeviceMemory, VkImageView, and optional VkSampler.
type VulkanImageResource struct {
	device  vk.Device
	Image   vk.Image
	Memory  vk.DeviceMemory
	View    vk.ImageView
	Sampler vk.Sampler
	Format  vk.Format
}

func (r *VulkanImageResource) GetImage() vk.Image     { return r.Image }
func (r *VulkanImageResource) GetView() vk.ImageView  { return r.View }
func (r *VulkanImageResource) GetSampler() vk.Sampler { return r.Sampler }
func (r *VulkanImageResource) GetFormat() vk.Format   { return r.Format }

// NewImageResourceFromHandles wraps pre-existing Device handles into a VulkanImageResource.
// Use this when image creation is done manually (e.g. staging buffer upload with optimal tiling).
func NewImageResourceFromHandles(device vk.Device, image vk.Image, memory vk.DeviceMemory, view vk.ImageView, sampler vk.Sampler, format vk.Format) VulkanImageResource {
	return VulkanImageResource{
		device:  device,
		Image:   image,
		Memory:  memory,
		View:    view,
		Sampler: sampler,
		Format:  format,
	}
}

func (r *VulkanImageResource) Destroy() {
	if r == nil {
		return
	}
	if r.Sampler != vk.NullSampler {
		vk.DestroySampler(r.device, r.Sampler, nil)
		r.Sampler = vk.NullSampler
	}
	if r.View != vk.NullImageView {
		vk.DestroyImageView(r.device, r.View, nil)
		r.View = vk.NullImageView
	}
	if r.Memory != vk.NullDeviceMemory {
		vk.FreeMemory(r.device, r.Memory, nil)
		r.Memory = vk.NullDeviceMemory
	}
	if r.Image != vk.NullImage {
		vk.DestroyImage(r.device, r.Image, nil)
		r.Image = vk.NullImage
	}
}

// NewImageTexture creates a texture from RGBA pixel data with a nearest-filter sampler.
// After creation, call TransitionImageLayout to transition from PreInitialized to ShaderReadOnlyOptimal.
func NewImageTexture(device vk.Device, gpu vk.PhysicalDevice, width, height uint32, rgbaPixels []byte) (VulkanImageResource, error) {
	var r VulkanImageResource
	r.device = device
	r.Format = vk.FormatR8g8b8a8Unorm

	err := vk.Error(vk.CreateImage(device, &vk.ImageCreateInfo{
		SType:         vk.StructureTypeImageCreateInfo,
		ImageType:     vk.ImageType2d,
		Format:        r.Format,
		Extent:        vk.Extent3D{Width: width, Height: height, Depth: 1},
		MipLevels:     1,
		ArrayLayers:   1,
		Samples:       vk.SampleCount1Bit,
		Tiling:        vk.ImageTilingLinear,
		Usage:         vk.ImageUsageFlags(vk.ImageUsageSampledBit),
		InitialLayout: vk.ImageLayoutPreinitialized,
	}, nil, &r.Image))
	if err != nil {
		return r, fmt.Errorf("vk.CreateImage (texture) failed with %s", err)
	}

	var memReq vk.MemoryRequirements
	vk.GetImageMemoryRequirements(device, r.Image, &memReq)
	memReq.Deref()

	memIdx, _ := vk.FindMemoryTypeIndex(gpu, memReq.MemoryTypeBits,
		vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit)
	err = vk.Error(vk.AllocateMemory(device, &vk.MemoryAllocateInfo{
		SType:           vk.StructureTypeMemoryAllocateInfo,
		AllocationSize:  memReq.Size,
		MemoryTypeIndex: memIdx,
	}, nil, &r.Memory))
	if err != nil {
		return r, fmt.Errorf("vk.AllocateMemory (texture) failed with %s", err)
	}
	vk.BindImageMemory(device, r.Image, r.Memory, 0)

	var pData unsafe.Pointer
	vk.MapMemory(device, r.Memory, 0, vk.DeviceSize(len(rgbaPixels)), 0, &pData)
	vk.Memcopy(pData, rgbaPixels)
	vk.UnmapMemory(device, r.Memory)

	err = vk.Error(vk.CreateSampler(device, &vk.SamplerCreateInfo{
		SType:         vk.StructureTypeSamplerCreateInfo,
		MagFilter:     vk.FilterNearest,
		MinFilter:     vk.FilterNearest,
		MipmapMode:    vk.SamplerMipmapModeNearest,
		AddressModeU:  vk.SamplerAddressModeClampToEdge,
		AddressModeV:  vk.SamplerAddressModeClampToEdge,
		AddressModeW:  vk.SamplerAddressModeClampToEdge,
		MaxAnisotropy: 1,
		CompareOp:     vk.CompareOpNever,
		BorderColor:   vk.BorderColorFloatOpaqueWhite,
	}, nil, &r.Sampler))
	if err != nil {
		return r, fmt.Errorf("vk.CreateSampler failed with %s", err)
	}

	err = vk.Error(vk.CreateImageView(device, &vk.ImageViewCreateInfo{
		SType:    vk.StructureTypeImageViewCreateInfo,
		Image:    r.Image,
		ViewType: vk.ImageViewType2d,
		Format:   r.Format,
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
	}, nil, &r.View))
	if err != nil {
		return r, fmt.Errorf("vk.CreateImageView (texture) failed with %s", err)
	}

	return r, nil
}

// NewImageTextureWithSampler uploads RGBA pixels through a staging buffer into an
// optimal-tiled sampled image and creates a sampler from samplerInfo.
func NewImageTextureWithSampler(device vk.Device, gpu vk.PhysicalDevice, queue vk.Queue, cmdCtx *CommandContext, width, height uint32, rgbaPixels []byte, samplerInfo vk.SamplerCreateInfo) (VulkanImageResource, error) {
	var r VulkanImageResource
	r.device = device
	r.Format = vk.FormatR8g8b8a8Unorm

	stagingBuf, err := NewBufferHostVisible(device, gpu, rgbaPixels, true, vk.BufferUsageFlags(vk.BufferUsageTransferSrcBit))
	if err != nil {
		return r, fmt.Errorf("create staging buffer: %w", err)
	}
	defer stagingBuf.Destroy()

	err = vk.Error(vk.CreateImage(device, &vk.ImageCreateInfo{
		SType:       vk.StructureTypeImageCreateInfo,
		ImageType:   vk.ImageType2d,
		Format:      r.Format,
		Extent:      vk.Extent3D{Width: width, Height: height, Depth: 1},
		MipLevels:   1,
		ArrayLayers: 1,
		Samples:     vk.SampleCount1Bit,
		Tiling:      vk.ImageTilingOptimal,
		Usage:       vk.ImageUsageFlags(vk.ImageUsageTransferDstBit | vk.ImageUsageSampledBit),
	}, nil, &r.Image))
	if err != nil {
		return r, fmt.Errorf("vk.CreateImage (texture) failed with %s", err)
	}

	var memReqs vk.MemoryRequirements
	vk.GetImageMemoryRequirements(device, r.Image, &memReqs)
	memReqs.Deref()

	memIdx, _ := vk.FindMemoryTypeIndex(gpu, memReqs.MemoryTypeBits, vk.MemoryPropertyDeviceLocalBit)
	err = vk.Error(vk.AllocateMemory(device, &vk.MemoryAllocateInfo{
		SType:           vk.StructureTypeMemoryAllocateInfo,
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memIdx,
	}, nil, &r.Memory))
	if err != nil {
		r.Destroy()
		return r, fmt.Errorf("vk.AllocateMemory (texture) failed with %s", err)
	}

	err = vk.Error(vk.BindImageMemory(device, r.Image, r.Memory, 0))
	if err != nil {
		r.Destroy()
		return r, fmt.Errorf("vk.BindImageMemory (texture) failed with %s", err)
	}

	cmd, err := cmdCtx.BeginOneTime()
	if err != nil {
		r.Destroy()
		return r, fmt.Errorf("BeginOneTime: %w", err)
	}

	rangeColor := vk.ImageSubresourceRange{AspectMask: vk.ImageAspectFlags(vk.ImageAspectColorBit), LevelCount: 1, LayerCount: 1}
	vk.CmdPipelineBarrier(cmd,
		vk.PipelineStageFlags(vk.PipelineStageTopOfPipeBit), vk.PipelineStageFlags(vk.PipelineStageTransferBit),
		0, 0, nil, 0, nil, 1, []vk.ImageMemoryBarrier{{
			SType:     vk.StructureTypeImageMemoryBarrier,
			OldLayout: vk.ImageLayoutUndefined, NewLayout: vk.ImageLayoutTransferDstOptimal,
			Image: r.Image, SubresourceRange: rangeColor,
			DstAccessMask:       vk.AccessFlags(vk.AccessTransferWriteBit),
			SrcQueueFamilyIndex: vk.QueueFamilyIgnored, DstQueueFamilyIndex: vk.QueueFamilyIgnored,
		}})
	vk.CmdCopyBufferToImage(cmd, stagingBuf.Buffer, r.Image, vk.ImageLayoutTransferDstOptimal, 1, []vk.BufferImageCopy{{
		ImageSubresource: vk.ImageSubresourceLayers{AspectMask: vk.ImageAspectFlags(vk.ImageAspectColorBit), LayerCount: 1},
		ImageExtent:      vk.Extent3D{Width: width, Height: height, Depth: 1},
	}})
	vk.CmdPipelineBarrier(cmd,
		vk.PipelineStageFlags(vk.PipelineStageTransferBit), vk.PipelineStageFlags(vk.PipelineStageFragmentShaderBit|vk.PipelineStageRayTracingShaderBit),
		0, 0, nil, 0, nil, 1, []vk.ImageMemoryBarrier{{
			SType:     vk.StructureTypeImageMemoryBarrier,
			OldLayout: vk.ImageLayoutTransferDstOptimal, NewLayout: vk.ImageLayoutShaderReadOnlyOptimal,
			Image: r.Image, SubresourceRange: rangeColor,
			SrcAccessMask: vk.AccessFlags(vk.AccessTransferWriteBit), DstAccessMask: vk.AccessFlags(vk.AccessShaderReadBit),
			SrcQueueFamilyIndex: vk.QueueFamilyIgnored, DstQueueFamilyIndex: vk.QueueFamilyIgnored,
		}})
	if err := cmdCtx.EndOneTime(queue, cmd); err != nil {
		r.Destroy()
		return r, fmt.Errorf("EndOneTime: %w", err)
	}

	err = vk.Error(vk.CreateImageView(device, &vk.ImageViewCreateInfo{
		SType:            vk.StructureTypeImageViewCreateInfo,
		Image:            r.Image,
		ViewType:         vk.ImageViewType2d,
		Format:           r.Format,
		SubresourceRange: rangeColor,
	}, nil, &r.View))
	if err != nil {
		r.Destroy()
		return r, fmt.Errorf("vk.CreateImageView (texture) failed with %s", err)
	}

	err = vk.Error(vk.CreateSampler(device, &samplerInfo, nil, &r.Sampler))
	if err != nil {
		r.Destroy()
		return r, fmt.Errorf("vk.CreateSampler failed with %s", err)
	}

	return r, nil
}

// NewImageStorage creates a device-local storage image and transitions it to General layout.
func NewImageStorage(device vk.Device, gpu vk.PhysicalDevice, queue vk.Queue, cmdPool vk.CommandPool, width, height uint32, format vk.Format) (VulkanImageResource, error) {
	var r VulkanImageResource
	r.device = device
	r.Format = format

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
	}, nil, &r.Image))
	if err != nil {
		return r, fmt.Errorf("vk.CreateImage (storage) failed with %s", err)
	}

	var memReqs vk.MemoryRequirements
	vk.GetImageMemoryRequirements(device, r.Image, &memReqs)
	memReqs.Deref()

	memIdx, _ := vk.FindMemoryTypeIndex(gpu, memReqs.MemoryTypeBits, vk.MemoryPropertyDeviceLocalBit)
	err = vk.Error(vk.AllocateMemory(device, &vk.MemoryAllocateInfo{
		SType:           vk.StructureTypeMemoryAllocateInfo,
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memIdx,
	}, nil, &r.Memory))
	if err != nil {
		vk.DestroyImage(device, r.Image, nil)
		return r, fmt.Errorf("vk.AllocateMemory (storage) failed with %s", err)
	}
	vk.BindImageMemory(device, r.Image, r.Memory, 0)

	err = vk.Error(vk.CreateImageView(device, &vk.ImageViewCreateInfo{
		SType:    vk.StructureTypeImageViewCreateInfo,
		Image:    r.Image,
		ViewType: vk.ImageViewType2d,
		Format:   format,
		SubresourceRange: vk.ImageSubresourceRange{
			AspectMask: vk.ImageAspectFlags(vk.ImageAspectColorBit),
			LevelCount: 1,
			LayerCount: 1,
		},
	}, nil, &r.View))
	if err != nil {
		vk.FreeMemory(device, r.Memory, nil)
		vk.DestroyImage(device, r.Image, nil)
		return r, fmt.Errorf("vk.CreateImageView (storage) failed with %s", err)
	}

	TransitionImageLayout(device, queue, cmdPool, r.Image, vk.ImageLayoutUndefined, vk.ImageLayoutGeneral)

	return r, nil
}

// NewImageDepth creates a depth buffer with the given dimensions and format.
func NewImageDepth(device vk.Device, gpu vk.PhysicalDevice, width, height uint32, format vk.Format) (VulkanImageResource, error) {
	var r VulkanImageResource
	r.device = device
	r.Format = format

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
	}, nil, &r.Image))
	if err != nil {
		return r, fmt.Errorf("vk.CreateImage (depth) failed with %s", err)
	}

	var memReq vk.MemoryRequirements
	vk.GetImageMemoryRequirements(device, r.Image, &memReq)
	memReq.Deref()

	memIdx, _ := vk.FindMemoryTypeIndex(gpu, memReq.MemoryTypeBits, vk.MemoryPropertyDeviceLocalBit)
	err = vk.Error(vk.AllocateMemory(device, &vk.MemoryAllocateInfo{
		SType:           vk.StructureTypeMemoryAllocateInfo,
		AllocationSize:  memReq.Size,
		MemoryTypeIndex: memIdx,
	}, nil, &r.Memory))
	if err != nil {
		return r, fmt.Errorf("vk.AllocateMemory (depth) failed with %s", err)
	}
	err = vk.Error(vk.BindImageMemory(device, r.Image, r.Memory, 0))
	if err != nil {
		return r, fmt.Errorf("vk.BindImageMemory (depth) failed with %s", err)
	}

	err = vk.Error(vk.CreateImageView(device, &vk.ImageViewCreateInfo{
		SType:    vk.StructureTypeImageViewCreateInfo,
		Image:    r.Image,
		ViewType: vk.ImageViewType2d,
		Format:   format,
		SubresourceRange: vk.ImageSubresourceRange{
			AspectMask: vk.ImageAspectFlags(vk.ImageAspectDepthBit),
			LevelCount: 1,
			LayerCount: 1,
		},
	}, nil, &r.View))
	if err != nil {
		return r, fmt.Errorf("vk.CreateImageView (depth) failed with %s", err)
	}

	return r, nil
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
	case oldLayout == vk.ImageLayoutUndefined && newLayout == vk.ImageLayoutGeneral:
		dstAccess = vk.AccessFlags(vk.AccessShaderWriteBit)
		srcStage = vk.PipelineStageFlags(vk.PipelineStageTopOfPipeBit)
		dstStage = vk.PipelineStageFlags(vk.PipelineStageAllCommandsBit)
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
