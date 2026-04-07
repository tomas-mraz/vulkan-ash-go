package ash

import (
	"fmt"
	"log/slog"

	vk "github.com/tomas-mraz/vulkan"
)

// Display owns a VkSurfaceKHR and its associated swapchain, image views, and framebuffers.
type Display struct {
	Device *Device

	Swapchains   []vk.Swapchain
	SwapchainLen []uint32

	surface vk.Surface

	DisplaySize   vk.Extent2D
	DisplayFormat vk.Format

	Framebuffers []vk.Framebuffer
	DisplayViews []vk.ImageView

	swapchainImages []vk.Image // cached for CmdCopyToSwapchain
}

func newDisplay(device *Device, surface vk.Surface, windowSize vk.Extent2D, oldSwapchain vk.Swapchain) (*Display, error) {

	d := &Display{
		Device:  device,
		surface: surface,
	}

	// Query surface capabilities
	var surfaceCapabilities vk.SurfaceCapabilities
	err := vk.Error(vk.GetPhysicalDeviceSurfaceCapabilities(device.Gpu, surface, &surfaceCapabilities))
	if err != nil {
		return nil, fmt.Errorf("vk.GetPhysicalDeviceSurfaceCapabilities failed with %s", err)
	}

	// Choose surface format
	var formatCount uint32
	vk.GetPhysicalDeviceSurfaceFormats(device.Gpu, surface, &formatCount, nil)
	formats := make([]vk.SurfaceFormat, formatCount)
	vk.GetPhysicalDeviceSurfaceFormats(device.Gpu, surface, &formatCount, formats)
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
		return nil, fmt.Errorf("vk.GetPhysicalDeviceSurfaceFormats not found suitable format")
	}

	// Resolve display extent
	surfaceCapabilities.Deref()
	slog.Debug(fmt.Sprintf("newDisplay surfaceCapabilities %v", surfaceCapabilities))
	d.DisplayFormat = formats[chosenFormat].Format

	surfaceCapabilities.CurrentExtent.Deref()
	switch {
	case surfaceCapabilities.CurrentExtent.Width == vk.MaxUint32 && surfaceCapabilities.CurrentExtent.Height == vk.MaxUint32:
		// Wayland: https://docs.vulkan.org/spec/latest/chapters/VK_KHR_surface/wsi.html
		d.DisplaySize = windowSize
		slog.Debug("[wayland specific] surface extent size is not set, using window size")
	case surfaceCapabilities.CurrentExtent.Width == 0 && surfaceCapabilities.CurrentExtent.Height == 0:
		// Android: surface not yet ready
		d.DisplaySize = windowSize
		slog.Debug("[android specific] surface extent size is 0x0, using window size")
	default:
		d.DisplaySize = surfaceCapabilities.CurrentExtent
	}
	slog.Debug(fmt.Sprintf("final display size is %d x %d", d.DisplaySize.Width, d.DisplaySize.Height))

	// Create swapchain
	swapchainCreateInfo := vk.SwapchainCreateInfo{
		SType:            vk.StructureTypeSwapchainCreateInfo,
		Surface:          surface,
		MinImageCount:    surfaceCapabilities.MinImageCount,
		ImageFormat:      formats[chosenFormat].Format,
		ImageColorSpace:  formats[chosenFormat].ColorSpace,
		ImageExtent:      d.DisplaySize,
		ImageUsage:       vk.ImageUsageFlags(vk.ImageUsageColorAttachmentBit | vk.ImageUsageTransferDstBit),
		PreTransform:     vk.SurfaceTransformIdentityBit,
		CompositeAlpha:   vk.CompositeAlphaOpaqueBit,
		ImageArrayLayers: 1,
		ImageSharingMode: vk.SharingModeExclusive,
		PresentMode:      vk.PresentModeFifo,
		OldSwapchain:     oldSwapchain,
		Clipped:          vk.False,
	}
	var swapchain vk.Swapchain
	err = vk.Error(vk.CreateSwapchain(device.Device, &swapchainCreateInfo, nil, &swapchain))
	if err != nil {
		return nil, fmt.Errorf("vk.CreateSwapchain failed with %s", err)
	}
	d.Swapchains = []vk.Swapchain{swapchain}
	d.SwapchainLen = make([]uint32, 1)

	err = vk.Error(vk.GetSwapchainImages(device.Device, d.DefaultSwapchain(), &(d.SwapchainLen[0]), nil))
	if err != nil {
		return nil, fmt.Errorf("vk.GetSwapchainImages failed with %s", err)
	}

	for i := range formats {
		formats[i].Free()
	}

	return d, nil
}

// Surface returns the owned VkSurfaceKHR handle.
func (d *Display) Surface() vk.Surface {
	return d.surface
}

func (d *Display) DefaultSwapchain() vk.Swapchain {
	return d.Swapchains[0]
}

func (d *Display) DefaultSwapchainLen() uint32 {
	return d.SwapchainLen[0]
}

func (d *Display) CreateFramebuffers(renderPass vk.RenderPass, depthView vk.ImageView) error {
	// Phase 1: vk.GetSwapchainImages

	var swapchainImagesCount uint32
	err := vk.Error(vk.GetSwapchainImages(d.Device.Device, d.DefaultSwapchain(), &swapchainImagesCount, nil))
	if err != nil {
		return fmt.Errorf("vk.GetSwapchainImages failed with %s", err)
	}
	swapchainImages := make([]vk.Image, swapchainImagesCount)
	vk.GetSwapchainImages(d.Device.Device, d.DefaultSwapchain(), &swapchainImagesCount, swapchainImages)

	// Phase 2: vk.CreateImageView
	//          create image view for each swapchain image

	d.DisplayViews = make([]vk.ImageView, len(swapchainImages))
	for i := range d.DisplayViews {
		viewCreateInfo := vk.ImageViewCreateInfo{
			SType:    vk.StructureTypeImageViewCreateInfo,
			Image:    swapchainImages[i],
			ViewType: vk.ImageViewType2d,
			Format:   d.DisplayFormat,
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
		err := vk.Error(vk.CreateImageView(d.Device.Device, &viewCreateInfo, nil, &d.DisplayViews[i]))
		if err != nil {
			return fmt.Errorf("vk.CreateImageView failed with %s", err)
		}
	}
	swapchainImages = nil

	// Phase 3: vk.CreateFramebuffer
	//          create a framebuffer from each swapchain image

	d.Framebuffers = make([]vk.Framebuffer, d.DefaultSwapchainLen())
	for i := range d.Framebuffers {
		attachments := []vk.ImageView{
			d.DisplayViews[i], depthView,
		}
		fbCreateInfo := vk.FramebufferCreateInfo{
			SType:           vk.StructureTypeFramebufferCreateInfo,
			RenderPass:      renderPass,
			Layers:          1,
			AttachmentCount: 1, // 2 if it has depthView
			PAttachments:    attachments,
			Width:           d.DisplaySize.Width,
			Height:          d.DisplaySize.Height,
		}
		if depthView != vk.NullImageView {
			fbCreateInfo.AttachmentCount = 2
		}
		err := vk.Error(vk.CreateFramebuffer(d.Device.Device, &fbCreateInfo, nil, &d.Framebuffers[i]))
		if err != nil {
			return fmt.Errorf("vk.CreateFramebuffer failed with %s", err)
		}
	}
	return nil
}

func (d *Display) getSwapchainImages() []vk.Image {
	if d.swapchainImages == nil {
		var count uint32
		vk.GetSwapchainImages(d.Device.Device, d.DefaultSwapchain(), &count, nil)
		d.swapchainImages = make([]vk.Image, count)
		vk.GetSwapchainImages(d.Device.Device, d.DefaultSwapchain(), &count, d.swapchainImages)
	}
	return d.swapchainImages
}

// CmdCopyToSwapchain records commands to copy a storage image to a swapchain image.
// It transitions the swapchain image to TransferDst, the source image from General to TransferSrc,
// performs the copy, then transitions back (swapchain to PresentSrc, source to General).
func (d *Display) CmdCopyToSwapchain(cmd vk.CommandBuffer, srcImage vk.Image, imageIndex uint32) {
	swapImages := d.getSwapchainImages()
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
			Extent:         vk.Extent3D{Width: d.DisplaySize.Width, Height: d.DisplaySize.Height, Depth: 1},
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

// destroySwapchain releases swapchain-related resources without destroying the surface.
func (d *Display) destroySwapchain() {
	for i := uint32(0); i < d.DefaultSwapchainLen(); i++ {
		if i < uint32(len(d.Framebuffers)) {
			vk.DestroyFramebuffer(d.Device.Device, d.Framebuffers[i], nil)
		}
		if i < uint32(len(d.DisplayViews)) {
			vk.DestroyImageView(d.Device.Device, d.DisplayViews[i], nil)
		}
	}
	d.Framebuffers = nil
	d.DisplayViews = nil
	for i := range d.Swapchains {
		vk.DestroySwapchain(d.Device.Device, d.Swapchains[i], nil)
	}
}

// Destroy releases all resources including the surface.
func (d *Display) Destroy() {
	if d == nil || d.Device == nil {
		return
	}
	d.destroySwapchain()
	if d.surface != vk.NullSurface {
		vk.DestroySurface(d.Device.Instance, d.surface, nil)
		d.surface = vk.NullSurface
	}
}
