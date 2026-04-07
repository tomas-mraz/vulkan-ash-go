package ash

import (
	"fmt"

	vk "github.com/tomas-mraz/vulkan"
)

// SwapchainRecreateFunc rebuilds resources that depend on the newly created display.
// Typical callers recreate depth images, framebuffers, and any size-dependent pipelines.
type SwapchainRecreateFunc func(display *Display) error

// SwapchainContext is a lightweight orchestration object for frame presentation.
// It centralizes Acquire/Present result handling and coordinates swapchain recreation.
type SwapchainContext struct {
	queue         vk.Queue
	display       *Display
	needsRecreate bool
}

// NewSwapchainContext groups the common swapchain dependencies without taking ownership.
func NewSwapchainContext(queue vk.Queue, display *Display) SwapchainContext {
	return SwapchainContext{
		queue:   queue,
		display: display,
	}
}

// GetSwapchain returns the currently attached display resource.
func (s *SwapchainContext) GetSwapchain() *Display {
	if s == nil {
		return nil
	}
	return s.display
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
func (s *SwapchainContext) AcquireNextImage(timeout uint64, semaphore vk.Semaphore, fence vk.Fence) (imageIndex uint32, acquired bool, err error) {
	if s == nil || s.display == nil {
		return 0, false, fmt.Errorf("swapchain context or display is nil")
	}
	if len(s.display.Swapchains) == 0 {
		return 0, false, fmt.Errorf("display has no swapchain handles")
	}

	result := vk.AcquireNextImage(s.display.Device.Device, s.display.DefaultSwapchain(), timeout, semaphore, fence, &imageIndex)
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
func (s *SwapchainContext) PresentImage(imageIndex uint32, waitSemaphores []vk.Semaphore) (presented bool, err error) {
	if s == nil || s.display == nil {
		return false, fmt.Errorf("swapchain context or display is nil")
	}
	if len(s.display.Swapchains) == 0 {
		return false, fmt.Errorf("display has no swapchain handles")
	}

	presentInfo := vk.PresentInfo{
		SType:              vk.StructureTypePresentInfo,
		WaitSemaphoreCount: uint32(len(waitSemaphores)),
		PWaitSemaphores:    waitSemaphores,
		SwapchainCount:     1,
		PSwapchains:        []vk.Swapchain{s.display.DefaultSwapchain()},
		PImageIndices:      []uint32{imageIndex},
	}
	result := vk.QueuePresent(s.queue, &presentInfo)
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
func (s *SwapchainContext) Recreate(windowSize vk.Extent2D, recreateFn SwapchainRecreateFunc) error {
	if s == nil || s.display == nil {
		return fmt.Errorf("swapchain context or display is nil")
	}
	if len(s.display.Swapchains) == 0 {
		return fmt.Errorf("display has no swapchain handles")
	}

	if err := vk.Error(vk.DeviceWaitIdle(s.display.Device.Device)); err != nil {
		return fmt.Errorf("vk.DeviceWaitIdle failed with %w", err)
	}

	oldDisplay := *s.display

	newDisp, err := newDisplay(s.display.Device, s.display.surface, windowSize, oldDisplay.DefaultSwapchain())
	if err != nil {
		return err
	}
	if recreateFn != nil {
		if err := recreateFn(newDisp); err != nil {
			newDisp.destroySwapchain()
			return err
		}
	}

	oldDisplay.destroySwapchain()
	*s.display = *newDisp
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
