package asch

import (
	"fmt"

	vk "github.com/tomas-mraz/vulkan"
)

// NewSyncObjects creates a fence and a semaphore for frame synchronization.
func NewSyncObjects(device vk.Device) (vk.Fence, vk.Semaphore, error) {
	var fence vk.Fence
	var sem vk.Semaphore
	if err := vk.Error(vk.CreateFence(device, &vk.FenceCreateInfo{
		SType: vk.StructureTypeFenceCreateInfo,
	}, nil, &fence)); err != nil {
		return fence, sem, fmt.Errorf("vk.CreateFence failed with %s", err)
	}
	if err := vk.Error(vk.CreateSemaphore(device, &vk.SemaphoreCreateInfo{
		SType: vk.StructureTypeSemaphoreCreateInfo,
	}, nil, &sem)); err != nil {
		vk.DestroyFence(device, fence, nil)
		return fence, sem, fmt.Errorf("vk.CreateSemaphore failed with %s", err)
	}
	return fence, sem, nil
}
