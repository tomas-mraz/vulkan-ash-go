//go:build android

package ash

import vk "github.com/tomas-mraz/vulkan"

// NewDisplay creates and initializes a Display object with a Vulkan surface and swapchain for rendering.
// Android version with a window pointer.
func NewDisplay(device *Device, windowSize vk.Extent2D, windowPointer uintptr) (*Display, error) {
	var surface vk.Surface
	err := vk.Error(vk.CreateWindowSurface(device.Instance, windowPointer, nil, &surface))
	if err != nil {
		return nil, err
	}
	return newDisplay(device, surface, windowSize, vk.NullSwapchain)
}
