//go:build android

package ash

import "github.com/tomas-mraz/vulkan"

// NewAndroidSurface is an Android helper to get Vulkan surface
func NewAndroidSurface(instance vulkan.Instance, windowPointer uintptr) (vulkan.Surface, error) {
	var surface vulkan.Surface
	err := vulkan.Error(vulkan.CreateWindowSurface(instance, windowPointer, nil, &surface))
	if err != nil {
		return vulkan.NullSurface, err
	}
	return surface, nil
}
