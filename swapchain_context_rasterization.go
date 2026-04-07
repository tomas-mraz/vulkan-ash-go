package ash

import (
	"fmt"
	"log"

	vk "github.com/tomas-mraz/vulkan"
)

// RasterizationRecreateConfig describes the rasterization resources that should
// be recreated together with the swapchain.
type RasterizationRecreateConfig struct {
	QueueFamilyIndex uint32
	DepthView        vk.ImageView
	PipelineOptions  PipelineOptions
}

// AcquireNextImageRasterization acquires the next image and automatically recreates
// the swapchain plus rasterization resources when needed.
// If it returns ok=false, the caller should skip the current frame.
func (s *SwapchainContext) AcquireNextImageRasterization(
	windowSize vk.Extent2D,
	rasterPass *RasterizationPass,
	cmdCtx *CommandContext,
	pipeline *PipelineRasterization,
	cfg RasterizationRecreateConfig,
	semaphore vk.Semaphore,
) (imageIndex uint32, ok bool, err error) {
	if s == nil {
		return 0, false, fmt.Errorf("swapchain context is nil")
	}
	if s.NeedsRecreate() {
		if err := s.recreateRasterization(windowSize, rasterPass, cmdCtx, pipeline, cfg); err != nil {
			return 0, false, err
		}
	}

	imageIndex, acquired, err := s.AcquireNextImage(vk.MaxUint64, semaphore, vk.NullFence)
	if err != nil {
		return 0, false, fmt.Errorf("AcquireNextImage: %w", err)
	}
	if !acquired {
		log.Printf("AcquireNextImage requested swapchain recreate")
		if err := s.recreateRasterization(windowSize, rasterPass, cmdCtx, pipeline, cfg); err != nil {
			return 0, false, err
		}
		return 0, false, nil
	}
	return imageIndex, true, nil
}

// BeginRenderPass resets and begins the frame command buffer for the given swapchain image.
func (s *SwapchainContext) BeginRenderPass(
	imageIndex uint32,
	rasterPass *RasterizationPass,
	cmdCtx *CommandContext,
	clearValues []vk.ClearValue,
) (vk.CommandBuffer, error) {
	if s == nil {
		var zero vk.CommandBuffer
		return zero, fmt.Errorf("swapchain context is nil")
	}

	swap := s.GetSwapchain()
	if swap == nil {
		var zero vk.CommandBuffer
		return zero, fmt.Errorf("swapchain context has nil swapchain")
	}

	cmdBuffers := cmdCtx.GetCmdBuffers()
	if int(imageIndex) >= len(cmdBuffers) {
		var zero vk.CommandBuffer
		return zero, fmt.Errorf("command buffer index %d out of range %d", imageIndex, len(cmdBuffers))
	}
	if int(imageIndex) >= len(swap.Framebuffers) {
		var zero vk.CommandBuffer
		return zero, fmt.Errorf("framebuffer index %d out of range %d", imageIndex, len(swap.Framebuffers))
	}

	cmd := cmdBuffers[imageIndex]
	if err := vk.Error(vk.ResetCommandBuffer(cmd, 0)); err != nil {
		var zero vk.CommandBuffer
		return zero, fmt.Errorf("ResetCommandBuffer: %w", err)
	}
	if err := vk.Error(vk.BeginCommandBuffer(cmd, &vk.CommandBufferBeginInfo{
		SType: vk.StructureTypeCommandBufferBeginInfo,
	})); err != nil {
		var zero vk.CommandBuffer
		return zero, fmt.Errorf("BeginCommandBuffer: %w", err)
	}
	vk.CmdBeginRenderPass(cmd, &vk.RenderPassBeginInfo{
		SType:           vk.StructureTypeRenderPassBeginInfo,
		RenderPass:      rasterPass.GetRenderPass(),
		Framebuffer:     swap.Framebuffers[imageIndex],
		RenderArea:      vk.Rect2D{Extent: swap.DisplaySize},
		ClearValueCount: uint32(len(clearValues)),
		PClearValues:    clearValues,
	}, vk.SubpassContentsInline)
	return cmd, nil
}

// EndRenderPass ends the render pass and finalizes command buffer recording.
func (s *SwapchainContext) EndRenderPass(cmd vk.CommandBuffer) error {
	vk.CmdEndRenderPass(cmd)
	if err := vk.Error(vk.EndCommandBuffer(cmd)); err != nil {
		return fmt.Errorf("EndCommandBuffer: %w", err)
	}
	return nil
}

// SubmitRender submits the recorded command buffer and waits for the fence.
func (s *SwapchainContext) SubmitRender(cmd vk.CommandBuffer, fence vk.Fence, waitSemaphores []vk.Semaphore) error {
	if s == nil {
		return fmt.Errorf("swapchain context is nil")
	}
	fences := []vk.Fence{fence}
	var waitStages []vk.PipelineStageFlags
	if len(waitSemaphores) > 0 {
		waitStages = []vk.PipelineStageFlags{vk.PipelineStageFlags(vk.PipelineStageColorAttachmentOutputBit)}
	}
	vk.ResetFences(s.manager.Device, 1, fences)
	if err := vk.Error(vk.QueueSubmit(s.manager.Queue, 1, []vk.SubmitInfo{{
		SType:              vk.StructureTypeSubmitInfo,
		WaitSemaphoreCount: uint32(len(waitSemaphores)),
		PWaitSemaphores:    waitSemaphores,
		PWaitDstStageMask:  waitStages,
		CommandBufferCount: 1,
		PCommandBuffers:    []vk.CommandBuffer{cmd},
	}}, fence)); err != nil {
		return fmt.Errorf("QueueSubmit: %w", err)
	}

	const timeoutNano = 10 * 1000 * 1000 * 1000
	if err := vk.Error(vk.WaitForFences(s.manager.Device, 1, fences, vk.True, timeoutNano)); err != nil {
		return fmt.Errorf("WaitForFences: %w", err)
	}
	return nil
}

// PresentImageRasterization presents the frame and automatically recreates
// the swapchain plus rasterization resources when needed.
func (s *SwapchainContext) PresentImageRasterization(
	windowSize vk.Extent2D,
	rasterPass *RasterizationPass,
	cmdCtx *CommandContext,
	pipeline *PipelineRasterization,
	cfg RasterizationRecreateConfig,
	imageIndex uint32,
) error {
	if s == nil {
		return fmt.Errorf("swapchain context is nil")
	}
	needsRecreateBeforePresent := s.NeedsRecreate()
	presented, err := s.PresentImage(imageIndex, nil)
	if !needsRecreateBeforePresent && s.NeedsRecreate() {
		log.Printf("QueuePresent returned Suboptimal or ErrorOutOfDate")
	}
	if err != nil {
		return fmt.Errorf("QueuePresent: %w", err)
	}
	if !presented || s.NeedsRecreate() {
		return s.recreateRasterization(windowSize, rasterPass, cmdCtx, pipeline, cfg)
	}
	return nil
}

func (s *SwapchainContext) recreateRasterization(
	windowSize vk.Extent2D,
	rasterPass *RasterizationPass,
	cmdCtx *CommandContext,
	pipeline *PipelineRasterization,
	cfg RasterizationRecreateConfig,
) error {
	log.Printf("Recreating swapchain: %dx%d", windowSize.Width, windowSize.Height)

	oldRasterPass := *rasterPass
	oldCmdCtx := *cmdCtx
	oldPipeline := *pipeline

	var nextRasterPass RasterizationPass
	var nextCmdCtx CommandContext
	var nextPipeline PipelineRasterization

	err := s.Recreate(windowSize, func(swap *VulkanSwapchainInfo) error {
		rp, err := NewRasterPass(s.manager.Device, swap.DisplayFormat)
		if err != nil {
			return err
		}
		if err := swap.CreateFramebuffers(rp.GetRenderPass(), cfg.DepthView); err != nil {
			rp.Destroy()
			return err
		}
		cc, err := NewCommandContext(s.manager.Device, cfg.QueueFamilyIndex, swap.DefaultSwapchainLen())
		if err != nil {
			rp.Destroy()
			return err
		}
		gfx, err := NewPipelineRasterization(s.manager.Device, swap.DisplaySize, rp.GetRenderPass(), cfg.PipelineOptions)
		if err != nil {
			cc.Destroy()
			rp.Destroy()
			return err
		}

		nextRasterPass = rp
		nextCmdCtx = cc
		nextPipeline = gfx
		return nil
	})
	if err != nil {
		return err
	}

	oldPipeline.Destroy()
	oldCmdCtx.Destroy()
	oldRasterPass.Destroy()

	*rasterPass = nextRasterPass
	*cmdCtx = nextCmdCtx
	*pipeline = nextPipeline

	log.Printf("Swapchain recreated: %dx%d", windowSize.Width, windowSize.Height)
	return nil
}
