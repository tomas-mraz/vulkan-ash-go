package ash

import vk "github.com/tomas-mraz/vulkan"

func AndroidDeviceExtensions() []string {
	return []string{
		MakeCString(vk.GoogleDisplayTimingExtensionName),
	}
}

func AndroidInstanceExtensions() []string {
	return []string{
		MakeCString(vk.KhrSurfaceExtensionName),
		MakeCString(vk.KhrAndroidSurfaceExtensionName),
	}
}
