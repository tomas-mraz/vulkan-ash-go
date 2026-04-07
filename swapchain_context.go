package ash

import (
	"fmt"

	vk "github.com/tomas-mraz/vulkan"
)

// SwapchainRecreateFunc rebuilds resources that depend on the newly created swapchain.
// Typical callers recreate depth images, framebuffers, and any size-dependent pipelines.
type SwapchainRecreateFunc func(swap *VulkanSwapchainInfo) error

// SwapchainContext is a lightweight orchestration object for frame presentation.
// It centralizes Acquire/Present result handling and coordinates swapchain recreation,
// but it does not own the Manager or VulkanSwapchainInfo it references.
type SwapchainContext struct {
	manager       *Manager
	swapchain     *VulkanSwapchainInfo
	needsRecreate bool
}

// NewSwapchainContext groups the common swapchain dependencies without taking ownership.
func NewSwapchainContext(manager *Manager, swapchain *VulkanSwapchainInfo) SwapchainContext {
	return SwapchainContext{
		manager:   manager,
		swapchain: swapchain,
	}
}

// GetSwapchain returns the currently attached swapchain resource.
func (s *SwapchainContext) GetSwapchain() *VulkanSwapchainInfo {
	if s == nil {
		return nil
	}
	return s.swapchain
}

// NeedsRecreate reports whether AcquireNextImage or PresentImage observed that the
// swapchain is suboptimal/out of date, or whether recreation was requested explicitly.
func (s *SwapchainContext) NeedsRecreate() bool {
	if s == nil {
		return false
	}
	return s.needsRecreate
}

// RequestRecreate marks the swapchain as needing a recreation on the next frame boundary.
func (s *SwapchainContext) RequestRecreate() {
	if s == nil {
		return
	}
	s.needsRecreate = true
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
		s.needsRecreate = true
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
	result := vk.QueuePresent(s.manager.Queue, &presentInfo)
	presented, recreate, err := classifySwapchainResult(result)
	if recreate {
		s.needsRecreate = true
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

	swap, err := newSwapchain(s.manager.Device, s.manager.Gpu, s.manager.Surface, windowSize, oldSwap.DefaultSwapchain())
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
	s.needsRecreate = false
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
