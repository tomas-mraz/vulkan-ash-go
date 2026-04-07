//go:build !android

package ash

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/tomas-mraz/vulkan"
)

// NewDesktopSurface is a GLFW helper for creating a Vulkan surface
// Desktop version Linux, Windows, ...
func NewDesktopSurface(instance vulkan.Instance, window *glfw.Window) (vulkan.Surface, error) {
	surfacePointer, err := window.CreateWindowSurface(instance, nil)
	if err != nil {
		return vulkan.NullSurface, err
	}
	return vulkan.SurfaceFromPointer(surfacePointer), nil
}
