package ash

import (
	"log"

	vk "github.com/tomas-mraz/vulkan"
)

// RaytracingRecreateConfig describes the raytracing resources that should
// be recreated together with the swapchain on window resize.
type RaytracingRecreateConfig struct {
	QueueFamilyIndex uint32
	UniformSize      int

	// DescriptorBindings is called with the newly created storage image and
	// uniform buffers to produce descriptor bindings for the new descriptor sets.
	// Bindings that don't change on resize (e.g. acceleration structures) should
	// be captured in the closure.
	DescriptorBindings func(storageImg *ImageResource, uniforms *VulkanUniformBuffers) []DescriptorBinding
}

// AcquireNextImageRaytracing acquires the next image and automatically recreates
// the swapchain plus raytracing resources when needed.
// If it returns ok=false, the caller should skip the current frame.
func (s *SwapchainContext) AcquireNextImageRaytracing(
	windowSize vk.Extent2D,
	cmdCtx *CommandContext,
	storageImg *ImageResource,
	uniforms *VulkanUniformBuffers,
	descriptors *VulkanDescriptorInfo,
	cfg RaytracingRecreateConfig,
	semaphore vk.Semaphore,
) (imageIndex uint32, ok bool, err error) {
	return s.AcquireNextImageAutoRecreate(windowSize, semaphore,
		raytracingRecreateFunc(s, cmdCtx, storageImg, uniforms, descriptors, cfg))
}

// PresentImageRaytracing presents the frame and automatically recreates
// the swapchain plus raytracing resources when needed.
func (s *SwapchainContext) PresentImageRaytracing(
	windowSize vk.Extent2D,
	cmdCtx *CommandContext,
	storageImg *ImageResource,
	uniforms *VulkanUniformBuffers,
	descriptors *VulkanDescriptorInfo,
	cfg RaytracingRecreateConfig,
	imageIndex uint32,
) error {
	return s.PresentImageAutoRecreate(windowSize, imageIndex,
		raytracingRecreateFunc(s, cmdCtx, storageImg, uniforms, descriptors, cfg))
}

// raytracingRecreateFunc builds a SwapchainRecreateFunc that recreates
// raytracing-specific resources: command context, storage image, uniform buffers, descriptors.
func raytracingRecreateFunc(
	s *SwapchainContext,
	cmdCtx *CommandContext,
	storageImg *ImageResource,
	uniforms *VulkanUniformBuffers,
	descriptors *VulkanDescriptorInfo,
	cfg RaytracingRecreateConfig,
) SwapchainRecreateFunc {
	return func(swap *Swapchain) error {
		dev := s.manager.Device
		gpu := s.manager.Gpu
		queue := s.manager.Queue
		w := swap.DisplaySize.Width
		h := swap.DisplaySize.Height
		swapLen := swap.DefaultSwapchainLen()

		log.Printf("Recreating raytracing resources: %dx%d", w, h)

		// 1. Command context (needed for storage image transition)
		cc, err := NewCommandContext(dev, cfg.QueueFamilyIndex, swapLen)
		if err != nil {
			return err
		}

		// 2. Storage image at new size
		img, err := NewImageStorage(dev, gpu, queue, cc.GetCmdPool(), w, h, swap.DisplayFormat)
		if err != nil {
			cc.Destroy()
			return err
		}

		// 3. Uniform buffers for new swapchain length
		ub, err := NewUniformBuffers(dev, gpu, swapLen, cfg.UniformSize)
		if err != nil {
			img.Destroy()
			cc.Destroy()
			return err
		}

		// 4. Descriptors with updated bindings
		bindings := cfg.DescriptorBindings(&img, &ub)
		desc, err := NewDescriptorSets(dev, swapLen, bindings)
		if err != nil {
			ub.Destroy()
			img.Destroy()
			cc.Destroy()
			return err
		}

		// Destroy old resources
		descriptors.Destroy()
		uniforms.Destroy()
		storageImg.Destroy()
		cmdCtx.Destroy()

		// Update pointers
		*cmdCtx = cc
		*storageImg = img
		*uniforms = ub
		*descriptors = desc

		log.Printf("Raytracing resources recreated: %dx%d", w, h)
		return nil
	}
}
