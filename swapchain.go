package ash

import (
	"fmt"
	"log/slog"

	vk "github.com/tomas-mraz/vulkan"
)

type VulkanSwapchainInfo struct {
	Device vk.Device

	Swapchains   []vk.Swapchain
	SwapchainLen []uint32

	DisplaySize   vk.Extent2D
	DisplayFormat vk.Format

	Framebuffers []vk.Framebuffer
	DisplayViews []vk.ImageView

	swapchainImages []vk.Image // cached for CmdCopyToSwapchain
}

func NewSwapchain(device vk.Device, gpu vk.PhysicalDevice, surface vk.Surface, windowSize vk.Extent2D) (VulkanSwapchainInfo, error) {
	//gpu := v.gpuDevices[0]

	// Phase 1: vk.GetPhysicalDeviceSurfaceCapabilities
	//			vk.GetPhysicalDeviceSurfaceFormats

	var swap VulkanSwapchainInfo
	var surfaceCapabilities vk.SurfaceCapabilities
	err := vk.Error(vk.GetPhysicalDeviceSurfaceCapabilities(gpu, surface, &surfaceCapabilities))
	if err != nil {
		err = fmt.Errorf("vk.GetPhysicalDeviceSurfaceCapabilities failed with %s", err)
		return swap, err
	}
	var formatCount uint32
	vk.GetPhysicalDeviceSurfaceFormats(gpu, surface, &formatCount, nil)
	formats := make([]vk.SurfaceFormat, formatCount)
	vk.GetPhysicalDeviceSurfaceFormats(gpu, surface, &formatCount, formats)

	slog.Debug(fmt.Sprintf("got %d physical device surface formats", formatCount))

	chosenFormat := -1
	for i := 0; i < int(formatCount); i++ {
		formats[i].Deref()
		if formats[i].Format == vk.FormatB8g8r8a8Unorm || formats[i].Format == vk.FormatR8g8b8a8Unorm {
			chosenFormat = i
			break
		}
	}
	if chosenFormat < 0 {
		err := fmt.Errorf("vk.GetPhysicalDeviceSurfaceFormats not found suitable format")
		return swap, err
	}

	// Phase 2: vk.CreateSwapchain
	//			create a swapchain with supported capabilities and format

	surfaceCapabilities.Deref()
	slog.Debug(fmt.Sprintf("NewSwapchain surfaceCapabilities %v", surfaceCapabilities))
	swap.DisplayFormat = formats[chosenFormat].Format

	surfaceCapabilities.CurrentExtent.Deref()
	if surfaceCapabilities.CurrentExtent.Width == vk.MaxUint32 && surfaceCapabilities.CurrentExtent.Height == vk.MaxUint32 {
		// Wayland specific https://docs.vulkan.org/spec/latest/chapters/VK_KHR_surface/wsi.html#vkCreateAndroidSurfaceKHR
		swap.DisplaySize = windowSize
		slog.Debug("[wayland specific] surface extent size is not set, using window size")
	} else if surfaceCapabilities.CurrentExtent.Width == 0 && surfaceCapabilities.CurrentExtent.Height == 0 {
		// Android-specific not yet ready surface
		swap.DisplaySize = windowSize
		slog.Debug("[android specific] surface extent size is 0x0, using window size")
	} else {
		swap.DisplaySize = surfaceCapabilities.CurrentExtent
	}
	slog.Debug(fmt.Sprintf("final display size is %d x %d", windowSize.Width, windowSize.Height))

	swapchainCreateInfo := vk.SwapchainCreateInfo{
		SType:            vk.StructureTypeSwapchainCreateInfo,
		Surface:          surface,
		MinImageCount:    surfaceCapabilities.MinImageCount,
		ImageFormat:      formats[chosenFormat].Format,
		ImageColorSpace:  formats[chosenFormat].ColorSpace,
		ImageExtent:      swap.DisplaySize,
		ImageUsage:       vk.ImageUsageFlags(vk.ImageUsageColorAttachmentBit | vk.ImageUsageTransferDstBit),
		PreTransform:     vk.SurfaceTransformIdentityBit,
		CompositeAlpha:   vk.CompositeAlphaOpaqueBit,
		ImageArrayLayers: 1,
		ImageSharingMode: vk.SharingModeExclusive,
		PresentMode:      vk.PresentModeFifo,
		OldSwapchain:     vk.NullSwapchain,
		Clipped:          vk.False,
	}
	var swapchain vk.Swapchain
	err = vk.Error(vk.CreateSwapchain(device, &swapchainCreateInfo, nil, &swapchain))
	if err != nil {
		err = fmt.Errorf("vk.CreateSwapchain failed with %s", err)
		return swap, err
	}
	swap.Swapchains = []vk.Swapchain{swapchain}
	swap.SwapchainLen = make([]uint32, 1)

	err = vk.Error(vk.GetSwapchainImages(device, swap.DefaultSwapchain(), &(swap.SwapchainLen[0]), nil))
	if err != nil {
		err = fmt.Errorf("vk.GetSwapchainImages failed with %s", err)
		return swap, err
	}
	for i := range formats {
		formats[i].Free()
	}
	swap.Device = device
	return swap, nil
}

func (s *VulkanSwapchainInfo) DefaultSwapchain() vk.Swapchain {
	return s.Swapchains[0]
}

func (s *VulkanSwapchainInfo) DefaultSwapchainLen() uint32 {
	return s.SwapchainLen[0]
}

func (s *VulkanSwapchainInfo) CreateFramebuffers(renderPass vk.RenderPass, depthView vk.ImageView) error {
	// Phase 1: vk.GetSwapchainImages

	var swapchainImagesCount uint32
	err := vk.Error(vk.GetSwapchainImages(s.Device, s.DefaultSwapchain(), &swapchainImagesCount, nil))
	if err != nil {
		err = fmt.Errorf("vk.GetSwapchainImages failed with %s", err)
		return err
	}
	swapchainImages := make([]vk.Image, swapchainImagesCount)
	vk.GetSwapchainImages(s.Device, s.DefaultSwapchain(), &swapchainImagesCount, swapchainImages)

	// Phase 2: vk.CreateImageView
	//			create image view for each swapchain image

	s.DisplayViews = make([]vk.ImageView, len(swapchainImages))
	for i := range s.DisplayViews {
		viewCreateInfo := vk.ImageViewCreateInfo{
			SType:    vk.StructureTypeImageViewCreateInfo,
			Image:    swapchainImages[i],
			ViewType: vk.ImageViewType2d,
			Format:   s.DisplayFormat,
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
		}
		err := vk.Error(vk.CreateImageView(s.Device, &viewCreateInfo, nil, &s.DisplayViews[i]))
		if err != nil {
			err = fmt.Errorf("vk.CreateImageView failed with %s", err)
			return err // bail out
		}
	}
	swapchainImages = nil

	// Phase 3: vk.CreateFramebuffer
	//			create a framebuffer from each swapchain image

	s.Framebuffers = make([]vk.Framebuffer, s.DefaultSwapchainLen())
	for i := range s.Framebuffers {
		attachments := []vk.ImageView{
			s.DisplayViews[i], depthView,
		}
		fbCreateInfo := vk.FramebufferCreateInfo{
			SType:           vk.StructureTypeFramebufferCreateInfo,
			RenderPass:      renderPass,
			Layers:          1,
			AttachmentCount: 1, // 2 if it has depthView
			PAttachments:    attachments,
			Width:           s.DisplaySize.Width,
			Height:          s.DisplaySize.Height,
		}
		if depthView != vk.NullImageView {
			fbCreateInfo.AttachmentCount = 2
		}
		err := vk.Error(vk.CreateFramebuffer(s.Device, &fbCreateInfo, nil, &s.Framebuffers[i]))
		if err != nil {
			err = fmt.Errorf("vk.CreateFramebuffer failed with %s", err)
			return err // bail out
		}
	}
	return nil
}

func (s *VulkanSwapchainInfo) getSwapchainImages() []vk.Image {
	if s.swapchainImages == nil {
		var count uint32
		vk.GetSwapchainImages(s.Device, s.DefaultSwapchain(), &count, nil)
		s.swapchainImages = make([]vk.Image, count)
		vk.GetSwapchainImages(s.Device, s.DefaultSwapchain(), &count, s.swapchainImages)
	}
	return s.swapchainImages
}

// CmdCopyToSwapchain records commands to copy a storage image to a swapchain image.
// It transitions the swapchain image to TransferDst, the source image from General to TransferSrc,
// performs the copy, then transitions back (swapchain to PresentSrc, source to General).
func (s *VulkanSwapchainInfo) CmdCopyToSwapchain(cmd vk.CommandBuffer, srcImage vk.Image, imageIndex uint32) {
	swapImages := s.getSwapchainImages()
	dstImage := swapImages[imageIndex]
	subresourceRange := vk.ImageSubresourceRange{
		AspectMask: vk.ImageAspectFlags(vk.ImageAspectColorBit), LevelCount: 1, LayerCount: 1,
	}

	// Transition swapchain image to transfer dst
	vk.CmdPipelineBarrier(cmd,
		vk.PipelineStageFlags(vk.PipelineStageAllCommandsBit), vk.PipelineStageFlags(vk.PipelineStageAllCommandsBit),
		0, 0, nil, 0, nil, 1, []vk.ImageMemoryBarrier{{
			SType: vk.StructureTypeImageMemoryBarrier, OldLayout: vk.ImageLayoutUndefined, NewLayout: vk.ImageLayoutTransferDstOptimal,
			Image: dstImage, SubresourceRange: subresourceRange, DstAccessMask: vk.AccessFlags(vk.AccessTransferWriteBit),
			SrcQueueFamilyIndex: vk.QueueFamilyIgnored, DstQueueFamilyIndex: vk.QueueFamilyIgnored,
		}})

	// Transition source image to transfer src
	vk.CmdPipelineBarrier(cmd,
		vk.PipelineStageFlags(vk.PipelineStageAllCommandsBit), vk.PipelineStageFlags(vk.PipelineStageAllCommandsBit),
		0, 0, nil, 0, nil, 1, []vk.ImageMemoryBarrier{{
			SType: vk.StructureTypeImageMemoryBarrier, OldLayout: vk.ImageLayoutGeneral, NewLayout: vk.ImageLayoutTransferSrcOptimal,
			Image: srcImage, SubresourceRange: subresourceRange,
			SrcAccessMask: vk.AccessFlags(vk.AccessShaderWriteBit), DstAccessMask: vk.AccessFlags(vk.AccessTransferReadBit),
			SrcQueueFamilyIndex: vk.QueueFamilyIgnored, DstQueueFamilyIndex: vk.QueueFamilyIgnored,
		}})

	// Copy
	vk.CmdCopyImage(cmd, srcImage, vk.ImageLayoutTransferSrcOptimal, dstImage, vk.ImageLayoutTransferDstOptimal,
		1, []vk.ImageCopy{{
			SrcSubresource: vk.ImageSubresourceLayers{AspectMask: vk.ImageAspectFlags(vk.ImageAspectColorBit), LayerCount: 1},
			DstSubresource: vk.ImageSubresourceLayers{AspectMask: vk.ImageAspectFlags(vk.ImageAspectColorBit), LayerCount: 1},
			Extent:         vk.Extent3D{Width: s.DisplaySize.Width, Height: s.DisplaySize.Height, Depth: 1},
		}})

	// Transition swapchain image to present
	vk.CmdPipelineBarrier(cmd,
		vk.PipelineStageFlags(vk.PipelineStageAllCommandsBit), vk.PipelineStageFlags(vk.PipelineStageAllCommandsBit),
		0, 0, nil, 0, nil, 1, []vk.ImageMemoryBarrier{{
			SType: vk.StructureTypeImageMemoryBarrier, OldLayout: vk.ImageLayoutTransferDstOptimal, NewLayout: vk.ImageLayoutPresentSrc,
			Image: dstImage, SubresourceRange: subresourceRange, SrcAccessMask: vk.AccessFlags(vk.AccessTransferWriteBit),
			SrcQueueFamilyIndex: vk.QueueFamilyIgnored, DstQueueFamilyIndex: vk.QueueFamilyIgnored,
		}})

	// Transition source image back to general
	vk.CmdPipelineBarrier(cmd,
		vk.PipelineStageFlags(vk.PipelineStageAllCommandsBit), vk.PipelineStageFlags(vk.PipelineStageAllCommandsBit),
		0, 0, nil, 0, nil, 1, []vk.ImageMemoryBarrier{{
			SType: vk.StructureTypeImageMemoryBarrier, OldLayout: vk.ImageLayoutTransferSrcOptimal, NewLayout: vk.ImageLayoutGeneral,
			Image: srcImage, SubresourceRange: subresourceRange,
			SrcAccessMask: vk.AccessFlags(vk.AccessTransferReadBit), DstAccessMask: vk.AccessFlags(vk.AccessShaderWriteBit),
			SrcQueueFamilyIndex: vk.QueueFamilyIgnored, DstQueueFamilyIndex: vk.QueueFamilyIgnored,
		}})
}

func (s *VulkanSwapchainInfo) Destroy() {
	for i := uint32(0); i < s.DefaultSwapchainLen(); i++ {
		if i < uint32(len(s.Framebuffers)) {
			vk.DestroyFramebuffer(s.Device, s.Framebuffers[i], nil)
		}
		if i < uint32(len(s.DisplayViews)) {
			vk.DestroyImageView(s.Device, s.DisplayViews[i], nil)
		}
	}
	s.Framebuffers = nil
	s.DisplayViews = nil
	for i := range s.Swapchains {
		vk.DestroySwapchain(s.Device, s.Swapchains[i], nil)
	}
}
