package ash

import (
	"fmt"

	vk "github.com/tomas-mraz/vulkan"
)

// VulkanSyncInfo manages a fence and semaphore pair for frame synchronization.
type VulkanSyncInfo struct {
	device    vk.Device
	Fence     vk.Fence
	Semaphore vk.Semaphore
}

// NewSyncObjects creates a fence and a semaphore for frame synchronization.
func NewSyncObjects(device vk.Device) (VulkanSyncInfo, error) {
	var s VulkanSyncInfo
	s.device = device
	if err := vk.Error(vk.CreateFence(device, &vk.FenceCreateInfo{
		SType: vk.StructureTypeFenceCreateInfo,
	}, nil, &s.Fence)); err != nil {
		return s, fmt.Errorf("vk.CreateFence failed with %s", err)
	}
	if err := vk.Error(vk.CreateSemaphore(device, &vk.SemaphoreCreateInfo{
		SType: vk.StructureTypeSemaphoreCreateInfo,
	}, nil, &s.Semaphore)); err != nil {
		vk.DestroyFence(device, s.Fence, nil)
		return s, fmt.Errorf("vk.CreateSemaphore failed with %s", err)
	}
	return s, nil
}

// Destroy releases the fence and semaphore.
func (s *VulkanSyncInfo) Destroy() {
	if s == nil {
		return
	}
	vk.DestroyFence(s.device, s.Fence, nil)
	vk.DestroySemaphore(s.device, s.Semaphore, nil)
}
