# Vulkan ash

Abstract layer for Vulkan API.

# API

## Device capability checks

### CheckDeviceExtensions

```go
func CheckDeviceExtensions(gpu vk.PhysicalDevice, required []string) (ok bool, missing []string)
```

Returns whether the physical device supports all required Vulkan extensions. Any unsupported extensions are returned in `missing`.

```go
ok, missing := asch.CheckDeviceExtensions(gpu, []string{
    "VK_KHR_acceleration_structure\x00",
    "VK_KHR_ray_tracing_pipeline\x00",
})
if !ok {
    log.Fatalf("Missing extensions: %v", missing)
}
```

### CheckDeviceApiVersion

```go
func CheckDeviceApiVersion(gpu vk.PhysicalDevice, minVersion uint32) (ok bool, deviceVersion uint32)
```

Returns whether the physical device supports at least the given Vulkan API version.

```go
ok, ver := asch.CheckDeviceApiVersion(gpu, vk.MakeVersion(1, 2, 0))
if !ok {
    log.Fatalf("GPU supports Vulkan %s, need 1.2.0", vk.Version(ver))
}
```

# Links

- Vulkan bindings (original xlab) - https://github.com/vulkan-go/vulkan
- Vulkan bindings (updated 1.3) - https://github.com/goki/vulkan
- Vulkan framework - https://github.com/goki/vgpu
- GLFW bindings - https://github.com/go-gl/glfw

