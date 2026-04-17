package ash

import (
	"fmt"
	"sync/atomic"

	vk "github.com/tomas-mraz/vulkan"
)

// SwapchainRecreateFunc rebuilds resources that depend on the newly created swapchain.
// Typical callers recreate depth images, framebuffers, and any size-dependent pipelines.
type SwapchainRecreateFunc func(swap *Swapchain) error

// SwapchainContext is a lightweight orchestration object for frame presentation.
// It centralizes Acquire/Present result handling and coordinates swapchain recreation,
// but it does not own the Manager or Swapchain it references.
//
// needsRecreate is atomic because producers (platform event threads, e.g. Android's
// NativeWindowRedrawNeeded) may request recreation while the render goroutine observes
// the flag at frame boundaries.
type SwapchainContext struct {
	manager       *Manager
	swapchain     *Swapchain
	displayTiming *DisplayTiming
	needsRecreate atomic.Bool
}

// NewSwapchainContext groups the common swapchain dependencies without taking ownership.
func NewSwapchainContext(manager *Manager, swapchain *Swapchain) SwapchainContext {
	return SwapchainContext{
		manager:   manager,
		swapchain: swapchain,
	}
}

// SetDisplayTiming attaches a DisplayTiming to the context.
// When set, PresentImage automatically chains display timing info.
func (s *SwapchainContext) SetDisplayTiming(dt *DisplayTiming) {
	if s != nil {
		s.displayTiming = dt
	}
}

// GetSwapchain returns the currently attached swapchain resource.
func (s *SwapchainContext) GetSwapchain() *Swapchain {
	if s == nil {
		return nil
	}
	return s.swapchain
}

// NeedsRecreate reports whether AcquireNextImage or PresentImage observed that the
// swapchain is suboptimal/out of date, or whether recreation was requested explicitly.
// Safe to call from any goroutine.
func (s *SwapchainContext) NeedsRecreate() bool {
	if s == nil {
		return false
	}
	return s.needsRecreate.Load()
}

// RequestRecreate marks the swapchain as needing a recreation on the next frame boundary.
// Safe to call from any goroutine; typically invoked by the platform event loop when it
// observes a window size / orientation change (e.g. Android NativeWindowRedrawNeeded).
func (s *SwapchainContext) RequestRecreate() {
	if s == nil {
		return
	}
	s.needsRecreate.Store(true)
}

// AcquireNextImage acquires the next swapchain image and classifies WSI warnings centrally.
// When Manager returns SUBOPTIMAL, acquisition still succeeds and NeedsRecreate becomes true.
// When Manager returns OUT_OF_DATE, no image is acquired and NeedsRecreate becomes true.
func (s *SwapchainContext) AcquireNextImage(timeout uint64, semaphore vk.Semaphore, fence vk.Fence) (imageIndex uint32, acquired bool, err error) {
	if s == nil {
		return 0, false, fmt.Errorf("swapchain context is nil")
	}
	if s.swapchain == nil {
		return 0, false, fmt.Errorf("swapchain context has nil swapchain")
	}
	if len(s.swapchain.Swapchains) == 0 {
		return 0, false, fmt.Errorf("swapchain context has no swapchain handles")
	}

	result := vk.AcquireNextImage(s.manager.Device, s.swapchain.DefaultSwapchain(), timeout, semaphore, fence, &imageIndex)
	acquired, recreate, err := classifySwapchainResult(result)
	if recreate {
		s.needsRecreate.Store(true)
	}
	if err != nil {
		return 0, false, fmt.Errorf("vk.AcquireNextImage failed with %w", err)
	}
	return imageIndex, acquired, nil
}

// PresentImage presents the rendered image and classifies WSI warnings centrally.
// SUBOPTIMAL presents successfully but requests recreation; OUT_OF_DATE skips presentation
// and requests recreation.
func (s *SwapchainContext) PresentImage(imageIndex uint32, waitSemaphores []vk.Semaphore) (presented bool, err error) {
	if s == nil {
		return false, fmt.Errorf("swapchain context is nil")
	}
	if s.swapchain == nil {
		return false, fmt.Errorf("swapchain context has nil swapchain")
	}
	if len(s.swapchain.Swapchains) == 0 {
		return false, fmt.Errorf("swapchain context has no swapchain handles")
	}

	presentInfo := vk.PresentInfo{
		SType:              vk.StructureTypePresentInfo,
		WaitSemaphoreCount: uint32(len(waitSemaphores)),
		PWaitSemaphores:    waitSemaphores,
		SwapchainCount:     1,
		PSwapchains:        []vk.Swapchain{s.swapchain.DefaultSwapchain()},
		PImageIndices:      []uint32{imageIndex},
	}
	if s.displayTiming != nil {
		s.displayTiming.ChainPresentInfo(&presentInfo)
	}
	result := vk.QueuePresent(s.manager.Queue, &presentInfo)
	presented, recreate, err := classifySwapchainResult(result)
	if recreate {
		s.needsRecreate.Store(true)
	}
	if err != nil {
		return false, fmt.Errorf("vk.QueuePresent failed with %w", err)
	}
	return presented, nil
}

// Recreate rebuilds the attached swapchain and runs an optional callback for dependent resources.
// The callback is invoked only after a new swapchain has been created successfully.
func (s *SwapchainContext) Recreate(windowSize vk.Extent2D, recreateFn SwapchainRecreateFunc) error {
	if s == nil {
		return fmt.Errorf("swapchain context is nil")
	}
	if s.swapchain == nil {
		return fmt.Errorf("swapchain context has nil swapchain")
	}
	if len(s.swapchain.Swapchains) == 0 {
		return fmt.Errorf("swapchain context has no swapchain handles")
	}

	if err := vk.Error(vk.DeviceWaitIdle(s.manager.Device)); err != nil {
		return fmt.Errorf("vk.DeviceWaitIdle failed with %w", err)
	}

	oldSwap := *s.swapchain

	swap, err := newSwapchain(s.manager, windowSize, oldSwap.DefaultSwapchain())
	if err != nil {
		return err
	}
	if recreateFn != nil {
		if err := recreateFn(&swap); err != nil {
			swap.Destroy()
			return err
		}
	}

	oldSwap.Destroy()
	*s.swapchain = swap
	s.needsRecreate.Store(false)
	if s.displayTiming != nil {
		s.displayTiming.Rebind(s.swapchain.DefaultSwapchain())
	}
	return nil
}

// BeginFrame resets and begins the command buffer for the given swapchain image.
// Use this for raytracing or other non-render-pass workflows.
// For rasterization with a render pass, use BeginRenderPass instead.
func (s *SwapchainContext) BeginFrame(imageIndex uint32, cmdCtx *CommandContext) (vk.CommandBuffer, error) {
	if s == nil {
		var zero vk.CommandBuffer
		return zero, fmt.Errorf("swapchain context is nil")
	}

	cmdBuffers := cmdCtx.GetCmdBuffers()
	if int(imageIndex) >= len(cmdBuffers) {
		var zero vk.CommandBuffer
		return zero, fmt.Errorf("command buffer index %d out of range %d", imageIndex, len(cmdBuffers))
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
	return cmd, nil
}

// EndFrame finalizes command buffer recording.
// Use this for raytracing or other non-render-pass workflows.
// For rasterization with a render pass, use EndRenderPass instead.
func (s *SwapchainContext) EndFrame(cmd vk.CommandBuffer) error {
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

// AcquireNextImageAutoRecreate acquires the next image and automatically recreates
// the swapchain via the provided callback when needed.
func (s *SwapchainContext) AcquireNextImageAutoRecreate(
	windowSize vk.Extent2D,
	semaphore vk.Semaphore,
	recreateFn SwapchainRecreateFunc,
) (imageIndex uint32, ok bool, err error) {
	if s == nil {
		return 0, false, fmt.Errorf("swapchain context is nil")
	}
	if s.NeedsRecreate() {
		if err := s.Recreate(windowSize, recreateFn); err != nil {
			return 0, false, err
		}
	}

	imageIndex, acquired, err := s.AcquireNextImage(vk.MaxUint64, semaphore, vk.NullFence)
	if err != nil {
		return 0, false, fmt.Errorf("AcquireNextImage: %w", err)
	}
	if !acquired {
		if err := s.Recreate(windowSize, recreateFn); err != nil {
			return 0, false, err
		}
		return 0, false, nil
	}
	return imageIndex, true, nil
}

// PresentImageAutoRecreate presents the frame and automatically recreates
// the swapchain via the provided callback when needed.
func (s *SwapchainContext) PresentImageAutoRecreate(
	windowSize vk.Extent2D,
	imageIndex uint32,
	recreateFn SwapchainRecreateFunc,
) error {
	if s == nil {
		return fmt.Errorf("swapchain context is nil")
	}
	presented, err := s.PresentImage(imageIndex, nil)
	if err != nil {
		return fmt.Errorf("QueuePresent: %w", err)
	}
	if !presented || s.NeedsRecreate() {
		return s.Recreate(windowSize, recreateFn)
	}
	return nil
}

func classifySwapchainResult(result vk.Result) (ok bool, recreate bool, err error) {
	switch result {
	case vk.Success:
		return true, false, nil
	case vk.Suboptimal:
		return true, true, nil
	case vk.ErrorOutOfDate:
		return false, true, nil
	default:
		return false, false, vk.Error(result)
	}
}
