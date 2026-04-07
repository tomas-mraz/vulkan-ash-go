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
	return s.AcquireNextImageAutoRecreate(windowSize, semaphore,
		rasterizationRecreateFunc(s, rasterPass, cmdCtx, pipeline, cfg))
}

// BeginRenderPass resets and begins the frame command buffer, then starts a render pass.
// It calls BeginFrame internally and adds the render pass on top.
func (s *SwapchainContext) BeginRenderPass(
	imageIndex uint32,
	rasterPass *RasterizationPass,
	cmdCtx *CommandContext,
	clearValues []vk.ClearValue,
) (vk.CommandBuffer, error) {
	swap := s.GetSwapchain()
	if swap == nil {
		var zero vk.CommandBuffer
		return zero, fmt.Errorf("swapchain context has nil swapchain")
	}
	if int(imageIndex) >= len(swap.Framebuffers) {
		var zero vk.CommandBuffer
		return zero, fmt.Errorf("framebuffer index %d out of range %d", imageIndex, len(swap.Framebuffers))
	}

	cmd, err := s.BeginFrame(imageIndex, cmdCtx)
	if err != nil {
		return cmd, err
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
// It ends the render pass and then calls EndFrame internally.
func (s *SwapchainContext) EndRenderPass(cmd vk.CommandBuffer) error {
	vk.CmdEndRenderPass(cmd)
	return s.EndFrame(cmd)
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
	return s.PresentImageAutoRecreate(windowSize, imageIndex,
		rasterizationRecreateFunc(s, rasterPass, cmdCtx, pipeline, cfg))
}

// rasterizationRecreateFunc builds a SwapchainRecreateFunc that recreates
// rasterization-specific resources: render pass, framebuffers, command context, pipeline.
func rasterizationRecreateFunc(
	s *SwapchainContext,
	rasterPass *RasterizationPass,
	cmdCtx *CommandContext,
	pipeline *PipelineRasterization,
	cfg RasterizationRecreateConfig,
) SwapchainRecreateFunc {
	return func(swap *VulkanSwapchainInfo) error {
		log.Printf("Recreating rasterization resources: %dx%d", swap.DisplaySize.Width, swap.DisplaySize.Height)

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

		rasterPass.Destroy()
		cmdCtx.Destroy()
		pipeline.Destroy()

		*rasterPass = rp
		*cmdCtx = cc
		*pipeline = gfx
		return nil
	}
}
