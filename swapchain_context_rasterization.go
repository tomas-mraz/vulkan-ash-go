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
	RecordCommands   func(swap VulkanSwapchainInfo, renderPass RasterizationPass, cmdCtx CommandContext, pipeline PipelineRasterization) error
}

// AcquireNextImageRasterization acquires the next image and automatically recreates
// the swapchain plus common rasterization resources when needed. If it returns
// ok=false, the application should skip the current frame.
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
	if s.NeedsRecreate() {
		log.Printf("AcquireNextImage returned Suboptimal or ErrorOutOfDate")
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
	if rasterPass == nil {
		var zero vk.CommandBuffer
		return zero, fmt.Errorf("rasterization pass is nil")
	}
	if cmdCtx == nil {
		var zero vk.CommandBuffer
		return zero, fmt.Errorf("command context is nil")
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

// EndRenderPass ends the render pass and finalizes command buffer recording for the frame.
func (s *SwapchainContext) EndRenderPass(cmd vk.CommandBuffer) error {
	if s == nil {
		return fmt.Errorf("swapchain context is nil")
	}
	vk.CmdEndRenderPass(cmd)
	if err := vk.Error(vk.EndCommandBuffer(cmd)); err != nil {
		return fmt.Errorf("EndCommandBuffer: %w", err)
	}
	return nil
}

// SubmitRender submits the recorded command buffer and waits for the fence to signal completion.
func (s *SwapchainContext) SubmitRender(cmd vk.CommandBuffer, fence vk.Fence, waitSemaphores []vk.Semaphore) error {
	if s == nil {
		return fmt.Errorf("swapchain context is nil")
	}
	fences := []vk.Fence{fence}
	waitStages := []vk.PipelineStageFlags(nil)
	if len(waitSemaphores) > 0 {
		waitStages = []vk.PipelineStageFlags{vk.PipelineStageFlags(vk.PipelineStageColorAttachmentOutputBit)}
	}
	vk.ResetFences(s.device, 1, fences)
	submitInfo := []vk.SubmitInfo{{
		SType:              vk.StructureTypeSubmitInfo,
		WaitSemaphoreCount: uint32(len(waitSemaphores)),
		PWaitSemaphores:    waitSemaphores,
		PWaitDstStageMask:  waitStages,
		CommandBufferCount: 1,
		PCommandBuffers:    []vk.CommandBuffer{cmd},
	}}
	if err := vk.Error(vk.QueueSubmit(s.queue, 1, submitInfo, fence)); err != nil {
		return fmt.Errorf("QueueSubmit: %w", err)
	}

	const timeoutNano = 10 * 1000 * 1000 * 1000
	if err := vk.Error(vk.WaitForFences(s.device, 1, fences, vk.True, timeoutNano)); err != nil {
		return fmt.Errorf("WaitForFences: %w", err)
	}
	return nil
}

// PresentImageRasterization presents the frame and automatically recreates the
// swapchain plus common rasterization resources after presentation when needed.
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
	if !presented {
		log.Printf("QueuePresent requested swapchain recreate")
		return s.recreateRasterization(windowSize, rasterPass, cmdCtx, pipeline, cfg)
	}
	if s.NeedsRecreate() {
		return s.recreateRasterization(windowSize, rasterPass, cmdCtx, pipeline, cfg)
	}
	return nil
}

// RecreateRasterization rebuilds the swapchain together with the common rasterization
// resources that depend on it: render pass, framebuffers, command buffers, and pipeline.
// Existing resources are kept alive until the new set is created successfully.
func (s *SwapchainContext) RecreateRasterization(
	windowSize vk.Extent2D,
	rasterPass *RasterizationPass,
	cmdCtx *CommandContext,
	pipeline *PipelineRasterization,
	cfg RasterizationRecreateConfig,
) error {
	return s.recreateRasterization(windowSize, rasterPass, cmdCtx, pipeline, cfg)
}

func (s *SwapchainContext) recreateRasterization(
	windowSize vk.Extent2D,
	rasterPass *RasterizationPass,
	cmdCtx *CommandContext,
	pipeline *PipelineRasterization,
	cfg RasterizationRecreateConfig,
) error {
	if s == nil {
		return fmt.Errorf("swapchain context is nil")
	}
	if rasterPass == nil {
		return fmt.Errorf("rasterization pass is nil")
	}
	if cmdCtx == nil {
		return fmt.Errorf("command context is nil")
	}
	if pipeline == nil {
		return fmt.Errorf("pipeline is nil")
	}

	log.Printf("Recreating swapchain: %dx%d", windowSize.Width, windowSize.Height)

	oldRasterPass := *rasterPass
	oldCmdCtx := *cmdCtx
	oldPipeline := *pipeline

	var nextRasterPass RasterizationPass
	var nextCmdCtx CommandContext
	var nextPipeline PipelineRasterization

	err := s.Recreate(windowSize, func(swap *VulkanSwapchainInfo) error {
		rp, err := NewRasterPass(swap.Device, swap.DisplayFormat)
		if err != nil {
			return err
		}
		if err := swap.CreateFramebuffers(rp.GetRenderPass(), cfg.DepthView); err != nil {
			rp.Destroy()
			return err
		}
		cc, err := NewCommandContext(swap.Device, cfg.QueueFamilyIndex, swap.DefaultSwapchainLen())
		if err != nil {
			rp.Destroy()
			return err
		}
		gfx, err := NewPipelineRasterization(swap.Device, swap.DisplaySize, rp.GetRenderPass(), cfg.PipelineOptions)
		if err != nil {
			cc.Destroy()
			rp.Destroy()
			return err
		}
		if cfg.RecordCommands != nil {
			if err := cfg.RecordCommands(*swap, rp, cc, gfx); err != nil {
				gfx.Destroy()
				cc.Destroy()
				rp.Destroy()
				return err
			}
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
