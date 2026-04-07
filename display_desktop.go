//go:build !android

package ash

import vk "github.com/tomas-mraz/vulkan"

// NewDisplay creates and initializes a Display object with a Vulkan surface and swapchain for rendering.
// Desktop (Linux, Windows, ...) version with a surface pointer.
func NewDisplay(device *Device, windowSize vk.Extent2D, surfacePointer uintptr) (*Display, error) {
	surface := vk.SurfaceFromPointer(surfacePointer)
	return newDisplay(device, surface, windowSize, vk.NullSwapchain)
}
