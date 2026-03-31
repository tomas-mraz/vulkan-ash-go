# API

Public API of the `ash` package. The overview below is split into exported structs and exported functions/methods; `Destroy` functions are intentionally omitted from the second table.

# Structures

| Structure                                                 | Layer         | Description                                                      |
|-----------------------------------------------------------|---------------|------------------------------------------------------------------|
| [`Vulkan`](#struct-vulkan)                                | common        | Main Vulkan context with instance, device, surface, and queue.   |
| [`DeviceOptions`](#struct-deviceoptions)                  | common        | Options for advanced Vulkan device creation.                     |
| [`Cleanup`](#struct-cleanup)                              | common        | Simple LIFO registry for cleanup steps.                          |
| [`CommandContext`](#commandcontext)                       | common        | Command pool and a set of reusable command buffers.              |
| [`VulkanSwapchainInfo`](#struct-vulkanswapchaininfo)      | common        | Metadata and handles for the swapchain and framebuffers.         |
| [`VulkanBufferInfo`](#struct-vulkanbufferinfo)            | common        | Simple vertex buffer helper.                                     |
| [`BufferResourceOptions`](#struct-bufferresourceoptions)  | common        | Configuration for a generic buffer resource.                     |
| [`VulkanBufferResource`](#struct-vulkanbufferresource)    | common        | Generic representation of a Vulkan buffer and its memory.        |
| [`VulkanIndexBufferInfo`](#struct-vulkanindexbufferinfo)  | common        | Helper for an index buffer and its index count.                  |
| [`VulkanImageResource`](#struct-vulkanimageresource)      | common        | Generic image resource with image, view, sampler, and memory.    |
| [`VulkanUniformBuffers`](#struct-vulkanuniformbuffers)    | common        | Set of uniform buffers, typically one per frame.                 |
| [`VulkanDescriptorInfo`](#struct-vulkandescriptorinfo)    | common        | Descriptor layout, pool, and allocated sets.                     |
| [`VulkanSyncInfo`](#struct-vulkansyncinfo)                | common        | Fence and semaphore for synchronization.                         |
| [`Model`](#struct-model)                                  | common        | Untextured glTF model in an interleaved vertex layout.           |
| [`TexturedModel`](#struct-texturedmodel)                  | common        | Textured model including an RGBA base color texture.             |
| [`Vec2`](#struct-vec2)                                    | common        | 2D vector type for math utilities.                               |
| [`Vec3`](#struct-vec3)                                    | common        | 3D vector type for math utilities.                               |
| [`Vec4`](#struct-vec4)                                    | common        | 4D vector type for math utilities.                               |
| [`Quat`](#struct-quat)                                    | common        | Quaternion type for rotations.                                   |
| [`Mat4x4`](#struct-mat4x4)                                | common        | 4x4 matrix for transforms and projections.                       |
| [`ArrayFloat32`](#struct-arrayfloat32)                    | common        | Helper alias for a byte view over `[]float32`.                   |
| [`PipelineOptions`](#struct-pipelineoptions)              | rasterization | Shader, layout, and state configuration for a graphics pipeline. |
| [`PipelineRasterizationInfo`](#pipelinerasterizationinfo) | rasterization | Handles for the graphics pipeline, layout, and cache.            |
| [`VulkanRasterPassInfo`](#struct-vulkanrasterpassinfo)    | rasterization | Wrapper over a rasterization render pass.                        |
| [`AccelerationStructure`](#struct-accelerationstructure)  | raytracing    | Acceleration structure including its backing buffer.             |
| [`GLTFPrimitive`](#struct-gltfprimitive)                  | raytracing    | GPU data for one glTF primitive used in RT.                      |
| [`GLTFModel`](#struct-gltfmodel)                          | raytracing    | Complete RT model with primitives, textures, and BLAS.           |
| [`ShaderBindingTable`](#shaderbindingtable)               | raytracing    | Shader binding table and its address regions.                    |

# Structures detailed description

<a id="commandcontext"></a>
## CommandContext{}

### `NewCommandContext()`
Creates a resettable command pool and optionally preallocates a set of primary command buffers. Typically used both for frame command buffers and transient upload operations.  
`func NewCommandContext(device vk.Device, queueFamilyIndex, commandBufferCount uint32) (CommandContext, error)`

### `(*CommandContext).GetCmdPool`
Returns the command pool handle managed by the context. Useful when another API expects a raw `vk.CommandPool`.  
`func (c *CommandContext) GetCmdPool() vk.CommandPool`


### `(*CommandContext).GetCmdBuffers`
Returns the preallocated command buffers. The library does not record or rotate them automatically; that remains the caller's responsibility.  
`func (c *CommandContext) GetCmdBuffers() []vk.CommandBuffer`


### `(*CommandContext).BeginOneTime`
Allocates a transient primary command buffer and immediately begins it with `OneTimeSubmit`. Suitable for short uploads and layout transitions.  
`func (c *CommandContext) BeginOneTime() (vk.CommandBuffer, error)`


### `(*CommandContext).EndOneTime`
Ends, submits, and synchronously completes a one-time command buffer. After completion it frees the buffer back to the command pool.  
`func (c *CommandContext) EndOneTime(queue vk.Queue, cmd vk.CommandBuffer) error`

### `(*CommandContext).Destroy`
Frees all preallocated command buffers and then destroys the command pool. It safely handles partially initialized states as well.  


<a id="pipelinerasterizationinfo"></a>
## PipelineRasterizationInfo{}

Wrapper over the created graphics pipeline, pipeline layout, and pipeline cache.

<a id="shaderbindingtable"></a>
## ShaderBindingTable{}
Shader binding table buffer and the computed `StridedDeviceAddressRegion` values for raygen, miss, hit, and callable groups.

### `NewSBT()`
Creates a shader binding table from an RT pipeline and computes its `StridedDeviceAddressRegion` values for raygen, miss, hit, and callable groups. Alignment is controlled by `handleAlignment` from Vulkan ray tracing properties.  
`func NewSBT(device vk.Device, gpu vk.PhysicalDevice, pipeline vk.Pipeline, handleSize, handleAlignment uint32, raygenCount, missCount, hitCount, callableCount uint32) (ShaderBindingTable, error)`


### `(*ShaderBindingTable).Destroy()`
Frees the buffer that stores the shader binding table.



## Functions

| Function                                                            | Layer         | Description                                                    |
|---------------------------------------------------------------------|---------------|----------------------------------------------------------------|
| [`SetDebug`](#setdebug)                                             | common        | Enables or disables internal debug mode.                       |
| [`NewExtentSize`](#newextentsize)                                   | common        | Creates a `vk.Extent2D` from `int` dimensions.                 |
| [`GetDeviceExtensions`](#getdeviceextensions)                       | common        | Returns device extension names supported by the GPU.           |
| [`CheckDeviceExtensions`](#checkdeviceextensions)                   | common        | Verifies that the GPU supports required extensions.            |
| [`CheckDeviceApiVersion`](#checkdeviceapiversion)                   | common        | Verifies the minimum Vulkan API version of the GPU.            |
| [`NewDevice`](#newdevice)                                           | common        | Creates the basic Vulkan instance/device/surface context.      |
| [`NewDeviceWithOptions`](#newdevicewithoptions)                     | common        | Creates a Vulkan context with advanced options.                |
| [`NewAndroidSurface`](#newandroidsurface)                           | Android       | Creates an Android surface from a native window.               |
| [`AndroidExtensions`](#androidextensions)                           | Android       | Returns required Android instance extensions.                  |
| [`MakeCString`](#makecstring)                                       | common        | Appends a null terminator to a Go string.                      |
| [`LoadShaderFromBytes`](#loadshaderfrombytes)                       | common        | Creates a shader module from raw SPIR-V data.                  |
| [`NewCommandContext`](#newcommandcontext)                           | common        | Creates a command pool and optional command buffers.           |
| [`NewSwapchain`](#newswapchain)                                     | common        | Creates a swapchain and selects a surface format.              |
| [`NewBuffer`](#newbuffer)                                           | common        | Creates the default triangle vertex buffer.                    |
| [`NewBufferWithData`](#newbufferwithdata)                           | common        | Creates a vertex buffer from `[]float32`.                      |
| [`NewBufferResource`](#newbufferresource)                           | common        | Creates a generic Vulkan buffer resource.                      |
| [`NewBufferHostVisible`](#newbufferhostvisible)                     | common        | Creates a host-visible buffer from a typed slice.              |
| [`NewBufferDeviceLocal`](#newbufferdevicelocal)                     | common        | Creates a device-local buffer resource.                        |
| [`NewIndexBuffer`](#newindexbuffer)                                 | common        | Creates an index buffer from `[]uint16`.                       |
| [`NewIndexBuffer32`](#newindexbuffer32)                             | common        | Creates an index buffer from `[]uint32`.                       |
| [`NewImageResourceFromHandles`](#newimageresourcefromhandles)       | common        | Wraps existing image handles into a resource type.             |
| [`NewImageTexture`](#newimagetexture)                               | common        | Creates a 2D texture from RGBA pixels.                         |
| [`NewImageTextureWithSampler`](#newimagetexturewithsampler)         | common        | Uploads a texture through staging and creates a sampler.       |
| [`NewImageStorage`](#newimagestorage)                               | common        | Creates a storage image and transitions it to `General`.       |
| [`NewImageDepth`](#newimagedepth)                                   | common        | Creates a depth image with a view.                             |
| [`TransitionImageLayout`](#transitionimagelayout)                   | common        | Performs a one-time image layout transition.                   |
| [`NewUniformBuffers`](#newuniformbuffers)                           | common        | Creates a set of uniform buffers.                              |
| [`NewDescriptorUBO`](#newdescriptorubo)                             | common        | Prepares descriptor sets for a UBO only.                       |
| [`NewDescriptorUBOTexture`](#newdescriptorubotexture)               | common        | Prepares descriptor sets for a UBO and a texture.              |
| [`NewSyncObjects`](#newsyncobjects)                                 | common        | Creates a fence and semaphore for synchronization.             |
| [`LoadModel`](#loadmodel)                                           | common        | Loads a glTF/GLB model without textures.                       |
| [`LoadGLBModel`](#loadglbmodel)                                     | common        | Loads a textured glTF/GLB model.                               |
| [`DegreesToRadians`](#degreestoradians)                             | common        | Converts degrees to radians.                                   |
| [`RadiansToDegrees`](#radianstodegrees)                             | common        | Converts radians to degrees.                                   |
| [`AlignUp`](#alignup)                                               | common        | Rounds a size up to a multiple of the alignment.               |
| [`Vec2MultInner`](#vec2multinner)                                   | common        | Computes the dot product of two `Vec2` values.                 |
| [`Vec3MultInner`](#vec3multinner)                                   | common        | Computes the dot product of two `Vec3` values.                 |
| [`Vec4MultInner`](#vec4multinner)                                   | common        | Computes the dot product of two `Vec4` values.                 |
| [`Vec4MultInner3`](#vec4multinner3)                                 | common        | Computes the dot product of the first three `Vec4` components. |
| [`QuatMultInner3`](#quatmultinner3)                                 | common        | Computes the inner product of the first three components.      |
| [`QuatInnerProduct`](#quatinnerproduct)                             | common        | Computes the full dot product of two quaternions.              |
| [`InvertMatrix`](#invertmatrix)                                     | common        | Returns a matrix inverse with identity fallback.               |
| [`DumpMatrix`](#dumpmatrix)                                         | common        | Writes a matrix into a debug string.                           |
| [`NewRasterPass`](#newrasterpass)                                   | rasterization | Creates a render pass with one color attachment.               |
| [`NewRasterPassWithDepth`](#newrasterpasswithdepth)                 | rasterization | Creates a render pass with color and depth attachments.        |
| [`NewGraphicsPipelineWithOptions`](#newgraphicspipelinewithoptions) | rasterization | Creates a graphics pipeline from the provided options.         |
| [`RaytracingExtensions`](#raytracingextensions)                     | raytracing    | Returns the list of required RT device extensions.             |
| [`DecodeGLTFTexture`](#decodegltftexture)                           | raytracing    | Decodes a glTF image into RGBA pixels.                         |
| [`LoadGLTFTextures`](#loadgltftextures)                             | raytracing    | Uploads glTF textures into Vulkan image resources.             |
| [`NewGLTFModel`](#newgltfmodel)                                     | raytracing    | Builds a `GLTFModel` from ready-made GPU resources.            |
| [`LoadGLTFModel`](#loadgltfmodel)                                   | raytracing    | Loads a glTF scene and prepares RT GPU data.                   |
| [`NewBufferWithDeviceAddress`](#newbufferwithdeviceaddress)         | raytracing    | Creates a host-visible buffer with a device address.           |
| [`NewDeviceLocalBuffer`](#newdevicelocalbuffer)                     | raytracing    | Creates a device-local buffer with a device address.           |
| [`GetBufferDeviceAddress`](#getbufferdeviceaddress)                 | raytracing    | Returns the device address of a given buffer.                  |
| [`NewSBT`](#newsbt)                                                 | raytracing    | Creates a shader binding table for an RT pipeline.             |

## Structures In Detail

<a id="struct-vulkan"></a>
### `Vulkan`

Main container for the library's basic Vulkan handles: `Instance`, `Device`, `Surface`, the physical GPU, and the queue.

<a id="struct-deviceoptions"></a>
### `DeviceOptions`

Options for `NewDeviceWithOptions`, especially device extensions, `pNext` chains, features, and the target API version.

<a id="struct-cleanup"></a>
### `Cleanup`

Simple LIFO registry of cleanup objects implementing `Destroy()`.

<a id="struct-commandcontext"></a>
### `CommandContext`

Manages a command pool and a set of reusable command buffers plus helpers for one-time command buffers.

<a id="struct-vulkanswapchaininfo"></a>
### `VulkanSwapchainInfo`

Carries swapchain handles, image counts, chosen format, dimensions, and created framebuffers.

<a id="struct-vulkanbufferinfo"></a>
### `VulkanBufferInfo`

Simpler helper around a vertex buffer and its device memory, used mainly in basic examples.

<a id="struct-bufferresourceoptions"></a>
### `BufferResourceOptions`

Configuration structure for a generic buffer resource: usage flags, memory properties, initial data, and device address support.

<a id="struct-vulkanbufferresource"></a>
### `VulkanBufferResource`

General representation of a Vulkan buffer that owns `vk.Buffer`, `vk.DeviceMemory`, its size, and an optional device address.

<a id="struct-vulkanindexbufferinfo"></a>
### `VulkanIndexBufferInfo`

Wrapper over an index buffer, its memory, and its index count.

<a id="struct-vulkanimageresource"></a>
### `VulkanImageResource`

General image resource including `vk.Image`, `vk.ImageView`, `vk.Sampler`, memory, and format.

<a id="struct-vulkanuniformbuffers"></a>
### `VulkanUniformBuffers`

Manages a set of uniform buffers of equal size, usually one per frame or per swapchain image.

<a id="struct-vulkandescriptorinfo"></a>
### `VulkanDescriptorInfo`

Holds a descriptor set layout, descriptor pool, and already allocated descriptor sets.

<a id="struct-vulkansyncinfo"></a>
### `VulkanSyncInfo`

Minimal synchronization bundle with one fence and one semaphore.

<a id="struct-model"></a>
### `Model`

Untextured glTF model with an interleaved `position + normal` layout and CPU-side index data.

<a id="struct-texturedmodel"></a>
### `TexturedModel`

Textured model with an interleaved `position + normal + uv` layout and an RGBA-decoded base color texture.

<a id="struct-vec2"></a>
### `Vec2`

Basic 2D vector type used by the math helper functions.

<a id="struct-vec3"></a>
### `Vec3`

Basic 3D vector type for transforms, normals, and spatial calculations.

<a id="struct-vec4"></a>
### `Vec4`

4D vector type suitable for homogeneous coordinates and matrix operations.

<a id="struct-quat"></a>
### `Quat`

Quaternion type for representing and composing rotations.

<a id="struct-mat4x4"></a>
### `Mat4x4`

4x4 matrix used for transforms, view matrices, and projection operations.

<a id="struct-arrayfloat32"></a>
### `ArrayFloat32`

Alias over `[]float32` with helpers for obtaining the size and a byte view without copying.

<a id="struct-pipelineoptions"></a>
### `PipelineOptions`

Configures the rasterization graphics pipeline: shaders, push constants, descriptor layouts, vertex layout, and depth testing.

<a id="struct-vulkanrasterpassinfo"></a>
### `VulkanRasterPassInfo`

Simple owner of a `vk.RenderPass` handle for rasterization.

<a id="struct-accelerationstructure"></a>
### `AccelerationStructure`

Owner of a Vulkan acceleration structure handle, its backing buffer, and an optional device address.

<a id="struct-gltfprimitive"></a>
### `GLTFPrimitive`

GPU representation of a single glTF primitive for ray tracing, including vertex/index buffers and texture indices.

<a id="struct-gltfmodel"></a>
### `GLTFModel`

Complete ray tracing model made of primitives, a geometry buffer, textures, and one BLAS.


## Common

<a id="setdebug"></a>
### `SetDebug`
`func SetDebug(state bool)`

Toggles the library's internal debug mode. When enabled, instance creation adds the debug report extension and callback.

<a id="vulkan-getdebugcallback"></a>
### `(*Vulkan).GetDebugCallback`
`func (v *Vulkan) GetDebugCallback() vk.DebugReportCallback`

Returns the debug callback handle stored in the main Vulkan context. Useful when integrating with custom debug tooling.

<a id="vulkan-destroy"></a>
### `(*Vulkan).Destroy`
`func (v *Vulkan) Destroy()`

Waits for `vk.DeviceWaitIdle` and then destroys the device, debug callback, surface, and instance in a safe order. This is the main cleanup entry point for the whole initialization path.

<a id="newextentsize"></a>
### `NewExtentSize`
`func NewExtentSize(width, height int) vk.Extent2D`

Converts Go `int` dimensions to `vk.Extent2D`. Mainly useful when dimensions come from a windowing library.

<a id="getdeviceextensions"></a>
### `GetDeviceExtensions`
`func GetDeviceExtensions(gpu vk.PhysicalDevice) []string`

Reads the supported device extensions of a physical GPU. The returned names do not contain C-style null terminators.

<a id="checkdeviceextensions"></a>
### `CheckDeviceExtensions`
`func CheckDeviceExtensions(gpu vk.PhysicalDevice, required []string) (ok bool, missing []string)`

Compares a required extension list with the actually available extensions and returns the missing ones. It also accepts input names that end with `\x00`.

<a id="checkdeviceapiversion"></a>
### `CheckDeviceApiVersion`
`func CheckDeviceApiVersion(gpu vk.PhysicalDevice, minVersion uint32) (ok bool, deviceVersion uint32)`

Checks whether the physical GPU supports at least the requested Vulkan API version. It also returns the actual device version for diagnostics.

<a id="newdevice"></a>
### `NewDevice`
`func NewDevice(appName string, instanceExtensions []string, createSurfaceFunc func(instance vk.Instance, window uintptr) (vk.Surface, error), window uintptr) (Vulkan, error)`

Creates the main Vulkan context including the instance, surface, physical device, logical device, and queue. This is the simplified entry point without advanced options.

<a id="newdevicewithoptions"></a>
### `NewDeviceWithOptions`
`func NewDeviceWithOptions(appName string, instanceExtensions []string, createSurfaceFunc func(instance vk.Instance, window uintptr) (vk.Surface, error), window uintptr, opts *DeviceOptions) (Vulkan, error)`

Extended version of `NewDevice` that allows adding device extensions, a custom `pNext` chain, features, and a target API version. It is especially useful for ray tracing and other advanced feature sets.

<a id="newandroidsurface"></a>
### `NewAndroidSurface`
`func NewAndroidSurface(instance vk.Instance, windowPtr uintptr) (vk.Surface, error)`

Android-only helper for creating a `vk.Surface` from a native window pointer. The function is available only when built with the `android` build tag.

<a id="makecstring"></a>
### `MakeCString`
`func MakeCString(s string) string`

Appends a null terminator to the end of a string if it is not already present. Useful when passing extension and layer names to Vulkan.

<a id="androidextensions"></a>
### `AndroidExtensions`
`func AndroidExtensions() []string`

Returns a fresh slice containing the Android instance extensions used by the library. The result is a copy and can be safely modified.

<a id="destroyerfunc-destroy"></a>
### `(DestroyerFunc).Destroy`
`func (f DestroyerFunc) Destroy()`

Runs the wrapped function and exposes it through the common `Destroyer` interface. Useful for lightweight cleanup callbacks without defining a dedicated type.

<a id="cleanup-add"></a>
### `(*Cleanup).Add`
`func (d *Cleanup) Add(obj Destroyer)`

Adds an object implementing `Destroy()` to the internal cleanup list. The intended pattern is to call it immediately after successful resource creation.

<a id="cleanup-destroy"></a>
### `(*Cleanup).Destroy`
`func (d *Cleanup) Destroy()`

Runs all registered destroyers in reverse order. That preserves the typical Vulkan teardown order.

<a id="loadshaderfrombytes"></a>
### `LoadShaderFromBytes`
`func LoadShaderFromBytes(device vk.Device, data []byte) (vk.ShaderModule, error)`

Creates a `vk.ShaderModule` directly from raw SPIR-V bytes. The caller remains responsible for the module lifetime.


<a id="newswapchain"></a>

## NewSwapchainInfo{}

### `NewSwapchain()`
`func NewSwapchain(device vk.Device, gpu vk.PhysicalDevice, surface vk.Surface, windowSize vk.Extent2D) (VulkanSwapchainInfo, error)`

Queries surface capabilities, selects a suitable format, and creates the swapchain. It also sets the resulting `DisplayFormat` and `DisplaySize`.

<a id="swapchain-defaultswapchain"></a>
### `(*VulkanSwapchainInfo).DefaultSwapchain`
`func (s *VulkanSwapchainInfo) DefaultSwapchain() vk.Swapchain`

Returns the first, and effectively the main, swapchain handle. The rest of the library works with this one by default.

<a id="swapchain-defaultswapchainlen"></a>
### `(*VulkanSwapchainInfo).DefaultSwapchainLen`
`func (s *VulkanSwapchainInfo) DefaultSwapchainLen() uint32`

Returns the number of images in the primary swapchain. Typically used for framebuffer counts or per-frame resources.

<a id="swapchain-createframebuffers"></a>
### `(*VulkanSwapchainInfo).CreateFramebuffers`
`func (s *VulkanSwapchainInfo) CreateFramebuffers(renderPass vk.RenderPass, depthView vk.ImageView) error`

Creates an image view for each swapchain image and then framebuffers for the given render pass. If `depthView` is non-null, it is attached as the second attachment.

<a id="swapchain-destroy"></a>
### `(*VulkanSwapchainInfo).Destroy`
`func (s *VulkanSwapchainInfo) Destroy()`

Cleans up framebuffers, image views, and all swapchain handles stored in the struct. It does not destroy the `vk.Surface` itself.

<a id="newbuffer"></a>
### `NewBuffer`
`func NewBuffer(device vk.Device, gpu vk.PhysicalDevice) (VulkanBufferInfo, error)`

Creates a simple vertex buffer with default triangle data. This is mainly a quick path for basic rasterization examples.

<a id="newbufferwithdata"></a>
### `NewBufferWithData`
`func NewBufferWithData(device vk.Device, gpu vk.PhysicalDevice, vertices []float32) (VulkanBufferInfo, error)`

Creates a vertex buffer and fills it from a `float32` slice. It assumes the slice is not empty.

<a id="vulkanbufferinfo-destroy"></a>
### `(*VulkanBufferInfo).Destroy`
`func (buf *VulkanBufferInfo) Destroy()`

Destroys all buffers in the object and frees their device memory. It performs no queue synchronization, so that must already be handled.

<a id="vulkanbufferinfo-defaultvertexbuffer"></a>
### `(*VulkanBufferInfo).DefaultVertexBuffer`
`func (buf *VulkanBufferInfo) DefaultVertexBuffer() vk.Buffer`

Returns the first vertex buffer stored in the struct. Internally the helper uses a single buffer, so this is a direct shortcut.

<a id="vulkanbufferinfo-getdevicememory"></a>
### `(*VulkanBufferInfo).GetDeviceMemory`
`func (buf *VulkanBufferInfo) GetDeviceMemory() vk.DeviceMemory`

Returns the device memory associated with the vertex buffer. This exposes a lower-level detail for cases that need direct access to the allocation.

<a id="newbufferresource"></a>
### `NewBufferResource`
`func NewBufferResource(device vk.Device, gpu vk.PhysicalDevice, size uint64, opts BufferResourceOptions) (VulkanBufferResource, error)`

Generic buffer constructor that lets you configure usage flags, memory properties, initial data, and device address support. It is the base building block for more advanced buffer resources.

<a id="newbufferhostvisible"></a>
### `NewBufferHostVisible`
`func NewBufferHostVisible[T any](device vk.Device, gpu vk.PhysicalDevice, data []T, enableDeviceAddress bool, usage vk.BufferUsageFlags) (VulkanBufferResource, error)`

Creates a host-visible and host-coherent buffer and computes its size from the provided slice. It can also enable shader device addresses.

<a id="newbufferdevicelocal"></a>
### `NewBufferDeviceLocal`
`func NewBufferDeviceLocal(device vk.Device, gpu vk.PhysicalDevice, size uint64, enableDeviceAddress bool, usage vk.BufferUsageFlags) (VulkanBufferResource, error)`

Creates a device-local buffer without CPU mapping. Suitable for resources that are fully managed from the GPU side.

<a id="vulkanbufferresource-update"></a>
### `(*VulkanBufferResource).Update`
`func (r *VulkanBufferResource) Update(data []byte) error`

Overwrites the buffer contents from offset 0. It works only for buffers backed by host-visible memory and validates that the new data does not exceed the buffer size.

<a id="vulkanbufferresource-destroy"></a>
### `(*VulkanBufferResource).Destroy`
`func (r *VulkanBufferResource) Destroy()`

Destroys the `vk.Buffer`, frees `vk.DeviceMemory`, and clears metadata including the device address. It is safe to call repeatedly on an already cleaned-up object.

<a id="newindexbuffer"></a>
### `NewIndexBuffer`
`func NewIndexBuffer(device vk.Device, gpu vk.PhysicalDevice, indices []uint16) (VulkanIndexBufferInfo, error)`

Creates an index buffer from 16-bit indices. The data is written directly into host-visible memory during construction.

<a id="newindexbuffer32"></a>
### `NewIndexBuffer32`
`func NewIndexBuffer32(device vk.Device, gpu vk.PhysicalDevice, indices []uint32) (VulkanIndexBufferInfo, error)`

Creates an index buffer from 32-bit indices. The usage pattern is the same as `NewIndexBuffer`, just for a wider index format.

<a id="vulkanindexbufferinfo-getbuffer"></a>
### `(*VulkanIndexBufferInfo).GetBuffer`
`func (ib *VulkanIndexBufferInfo) GetBuffer() vk.Buffer`

Returns the index buffer handle. Typically used for binding with `vk.CmdBindIndexBuffer`.

<a id="vulkanindexbufferinfo-getcount"></a>
### `(*VulkanIndexBufferInfo).GetCount`
`func (ib *VulkanIndexBufferInfo) GetCount() uint32`

Returns the number of indices stored in the buffer. This value can be used directly for indexed draw calls.

<a id="vulkanindexbufferinfo-destroy"></a>
### `(*VulkanIndexBufferInfo).Destroy`
`func (ib *VulkanIndexBufferInfo) Destroy()`

Frees the device memory and destroys the index buffer. It does not perform GPU synchronization.

<a id="vulkanimageresource-getimage"></a>
### `(*VulkanImageResource).GetImage`
`func (r *VulkanImageResource) GetImage() vk.Image`

Returns the low-level `vk.Image` handle. Useful when passing it directly to Vulkan structures and commands.

<a id="vulkanimageresource-getview"></a>
### `(*VulkanImageResource).GetView`
`func (r *VulkanImageResource) GetView() vk.ImageView`

Returns the `vk.ImageView` handle associated with the resource. Commonly used in framebuffers and descriptor sets.

<a id="vulkanimageresource-getsampler"></a>
### `(*VulkanImageResource).GetSampler`
`func (r *VulkanImageResource) GetSampler() vk.Sampler`

Returns the resource's `vk.Sampler` handle. If no sampler exists, the method returns a null handle.

<a id="vulkanimageresource-getformat"></a>
### `(*VulkanImageResource).GetFormat`
`func (r *VulkanImageResource) GetFormat() vk.Format`

Returns the Vulkan image format. This is a quick way to access metadata without reading the struct field directly.

<a id="newimageresourcefromhandles"></a>
### `NewImageResourceFromHandles`
`func NewImageResourceFromHandles(device vk.Device, image vk.Image, memory vk.DeviceMemory, view vk.ImageView, sampler vk.Sampler, format vk.Format) VulkanImageResource`

Wraps already created Vulkan handles into `VulkanImageResource`. Intended for cases where the image is created outside of the library helpers.

<a id="vulkanimageresource-destroy"></a>
### `(*VulkanImageResource).Destroy`
`func (r *VulkanImageResource) Destroy()`

Destroys the sampler, image view, device memory, and image in the correct order. It also works safely on a partially initialized struct.

<a id="newimagetexture"></a>
### `NewImageTexture`
`func NewImageTexture(device vk.Device, gpu vk.PhysicalDevice, width, height uint32, rgbaPixels []byte) (VulkanImageResource, error)`

Creates a linearly tiled 2D texture from RGBA pixels and adds a simple nearest sampler. After creation, the image may still need to be transitioned to a shader-read layout.

<a id="newimagetexturewithsampler"></a>
### `NewImageTextureWithSampler`
`func NewImageTextureWithSampler(device vk.Device, gpu vk.PhysicalDevice, queue vk.Queue, cmdCtx *CommandContext, width, height uint32, rgbaPixels []byte, samplerInfo vk.SamplerCreateInfo) (VulkanImageResource, error)`

Uploads a texture through a staging buffer into an optimally tiled sampled image and creates a sampler from the provided `samplerInfo`. The function performs the required layout transitions during upload.

<a id="newimagestorage"></a>
### `NewImageStorage`
`func NewImageStorage(device vk.Device, gpu vk.PhysicalDevice, queue vk.Queue, cmdPool vk.CommandPool, width, height uint32, format vk.Format) (VulkanImageResource, error)`

Creates a storage image in device-local memory and performs a one-time transition to the `General` layout. Suitable for compute outputs or ray generation targets.

<a id="newimagedepth"></a>
### `NewImageDepth`
`func NewImageDepth(device vk.Device, gpu vk.PhysicalDevice, width, height uint32, format vk.Format) (VulkanImageResource, error)`

Creates a depth attachment image in device-local memory together with its image view. It does not perform a layout transition because that is typically handled by the render pass.

<a id="transitionimagelayout"></a>
### `TransitionImageLayout`
`func TransitionImageLayout(device vk.Device, queue vk.Queue, cmdPool vk.CommandPool, image vk.Image, oldLayout, newLayout vk.ImageLayout)`

Performs several common layout transition scenarios using a one-time command buffer. This is a low-level helper without an error return, so it assumes a valid layout combination.

<a id="newuniformbuffers"></a>
### `NewUniformBuffers`
`func NewUniformBuffers(device vk.Device, gpu vk.PhysicalDevice, count uint32, dataSize int) (VulkanUniformBuffers, error)`

Creates `count` uniform buffers of the same size, typically one per swapchain image. All of them use host-visible and host-coherent memory.

<a id="vulkanuniformbuffers-update"></a>
### `(*VulkanUniformBuffers).Update`
`func (u *VulkanUniformBuffers) Update(index uint32, data []byte)`

Writes new data into the uniform buffer at the given index. The function assumes a valid index and data size and does not return an error.

<a id="vulkanuniformbuffers-getbuffer"></a>
### `(*VulkanUniformBuffers).GetBuffer`
`func (u *VulkanUniformBuffers) GetBuffer(index uint32) vk.Buffer`

Returns a specific uniform buffer by index. Mainly used when filling descriptor sets.

<a id="vulkanuniformbuffers-getbuffers"></a>
### `(*VulkanUniformBuffers).GetBuffers`
`func (u *VulkanUniformBuffers) GetBuffers() []vk.Buffer`

Returns all created uniform buffers at once. Convenient for batch descriptor initialization.

<a id="vulkanuniformbuffers-getsize"></a>
### `(*VulkanUniformBuffers).GetSize`
`func (u *VulkanUniformBuffers) GetSize() int`

Returns the size of one uniform buffer in bytes. It does not return the combined size of all allocations.

<a id="vulkanuniformbuffers-destroy"></a>
### `(*VulkanUniformBuffers).Destroy`
`func (u *VulkanUniformBuffers) Destroy()`

Frees all managed buffers and their device memory. This is the full teardown path for the whole resource set.

<a id="vulkandescriptorinfo-getlayout"></a>
### `(*VulkanDescriptorInfo).GetLayout`
`func (d *VulkanDescriptorInfo) GetLayout() vk.DescriptorSetLayout`

Returns the descriptor set layout created by the library helpers. Typically used while building the pipeline layout.

<a id="vulkandescriptorinfo-getsets"></a>
### `(*VulkanDescriptorInfo).GetSets`
`func (d *VulkanDescriptorInfo) GetSets() []vk.DescriptorSet`

Returns the allocated descriptor sets. The library helpers already populate them based on the selected constructor.

<a id="vulkandescriptorinfo-destroy"></a>
### `(*VulkanDescriptorInfo).Destroy`
`func (d *VulkanDescriptorInfo) Destroy()`

Destroys the descriptor pool and descriptor set layout. The sets themselves are implicitly released together with the pool.

<a id="newdescriptorubo"></a>
### `NewDescriptorUBO`
`func NewDescriptorUBO(device vk.Device, uniforms *VulkanUniformBuffers, count uint32) (VulkanDescriptorInfo, error)`

Creates a descriptor layout, pool, and `count` descriptor sets with a single UBO binding at slot 0. The binding is prepared for the vertex shader stage.

<a id="newdescriptorubotexture"></a>
### `NewDescriptorUBOTexture`
`func NewDescriptorUBOTexture(device vk.Device, uniforms *VulkanUniformBuffers, texture *VulkanImageResource, count uint32) (VulkanDescriptorInfo, error)`

Like `NewDescriptorUBO`, but also adds a combined image sampler binding at slot 1. The texture binding is intended for the fragment shader.

<a id="newsyncobjects"></a>
### `NewSyncObjects`
`func NewSyncObjects(device vk.Device) (VulkanSyncInfo, error)`

Creates a simple pair of `Fence` and `Semaphore` objects for frame synchronization. This is a minimal helper for a basic render loop.

<a id="vulkansyncinfo-destroy"></a>
### `(*VulkanSyncInfo).Destroy`
`func (s *VulkanSyncInfo) Destroy()`

Releases the fence and semaphore stored in `VulkanSyncInfo`. The object cannot be reused after destruction.

<a id="model-vertexcount"></a>
### `(*Model).VertexCount`
`func (m *Model) VertexCount() int`

Returns the number of vertices in the interleaved model data. The value is derived from `FloatsPerVertex`.

<a id="model-indexcount"></a>
### `(*Model).IndexCount`
`func (m *Model) IndexCount() int`

Returns the number of indices in the model. This is simply the length of the `Indices` slice.

<a id="texturedmodel-vertexcount"></a>
### `(*TexturedModel).VertexCount`
`func (m *TexturedModel) VertexCount() int`

Returns the number of vertices in the textured model. Just like `Model`, it is computed from the interleaved vertex layout.

<a id="texturedmodel-indexcount"></a>
### `(*TexturedModel).IndexCount`
`func (m *TexturedModel) IndexCount() int`

Returns the number of indices in the textured model. It is a direct helper over the underlying slice.

<a id="loadmodel"></a>
### `LoadModel`
`func LoadModel(path string) (Model, error)`

Loads a glTF or GLB file and returns an untextured model with the interleaved layout `position(3) + normal(3)`. Suitable for simpler rasterization use cases.

<a id="loadglbmodel"></a>
### `LoadGLBModel`
`func LoadGLBModel(path string) (TexturedModel, error)`

Loads a glTF or GLB file and returns a textured model with the layout `position(3) + normal(3) + uv(2)`, including the base color texture decoded into RGBA.

<a id="degreestoradians"></a>
### `DegreesToRadians`
`func DegreesToRadians(angleDegrees float32) float32`

Converts an angle in degrees to radians. Basic math helper for working with rotations.

<a id="radianstodegrees"></a>
### `RadiansToDegrees`
`func RadiansToDegrees(angleRadians float32) float32`

Converts an angle in radians to degrees. This is the inverse of `DegreesToRadians`.

<a id="alignup"></a>
### `AlignUp`
`func AlignUp(size, alignment uint32) uint32`

Rounds a size up to the nearest multiple of `alignment`. Used for example when laying out SBTs or other binary blocks.

<a id="vec2-add"></a>
### `(*Vec2).Add`
`func (r *Vec2) Add(a, b *Vec2)`

Adds two `Vec2` values and stores the result in the receiver.

<a id="vec2-sub"></a>
### `(*Vec2).Sub`
`func (r *Vec2) Sub(a, b *Vec2)`

Subtracts `b` from `a` and stores the result in the receiver.

<a id="vec2-scale"></a>
### `(*Vec2).Scale`
`func (r *Vec2) Scale(v *Vec2, s float32)`

Multiplies vector `v` by scalar `s` and stores the result in the receiver.

<a id="vec2-len"></a>
### `(*Vec2).Len`
`func (v *Vec2) Len() float32`

Returns the Euclidean length of a 2D vector.

<a id="vec2-norm"></a>
### `(*Vec2).Norm`
`func (r *Vec2) Norm(v *Vec2)`

Normalizes the vector to unit length and stores it in the receiver.

<a id="vec2-min"></a>
### `(*Vec2).Min`
`func (r *Vec2) Min(a, b *Vec2)`

Selects the smaller component-wise value from two 2D vectors.

<a id="vec2-max"></a>
### `(*Vec2).Max`
`func (r *Vec2) Max(a, b *Vec2)`

Selects the larger component-wise value from two 2D vectors.

<a id="vec2multinner"></a>
### `Vec2MultInner`
`func Vec2MultInner(a, b *Vec2) float32`

Computes the dot product of two `Vec2` values.

<a id="vec3-add"></a>
### `(*Vec3).Add`
`func (r *Vec3) Add(a, b *Vec3)`

Adds two `Vec3` values and stores the result in the receiver.

<a id="vec3-sub"></a>
### `(*Vec3).Sub`
`func (r *Vec3) Sub(a, b *Vec3)`

Subtracts two 3D vectors component-wise.

<a id="vec3-scale"></a>
### `(*Vec3).Scale`
`func (r *Vec3) Scale(v *Vec3, s float32)`

Multiplies a 3D vector by a scalar.

<a id="vec3-scalevec4"></a>
### `(*Vec3).ScaleVec4`
`func (r *Vec3) ScaleVec4(v *Vec4, s float32)`

Takes the first three components of `Vec4`, scales them, and stores them in `Vec3`.

<a id="vec3-scalequat"></a>
### `(*Vec3).ScaleQuat`
`func (r *Vec3) ScaleQuat(q *Quat, s float32)`

Takes the vector part of a quaternion, scales it, and stores it in `Vec3`.

<a id="vec3-len"></a>
### `(*Vec3).Len`
`func (v *Vec3) Len() float32`

Returns the length of a 3D vector.

<a id="vec3-norm"></a>
### `(*Vec3).Norm`
`func (r *Vec3) Norm(v *Vec3)`

Normalizes a 3D vector to unit length.

<a id="vec3-min"></a>
### `(*Vec3).Min`
`func (r *Vec3) Min(a, b *Vec3)`

Selects the component-wise minimum of two `Vec3` values.

<a id="vec3-max"></a>
### `(*Vec3).Max`
`func (r *Vec3) Max(a, b *Vec3)`

Selects the component-wise maximum of two `Vec3` values.

<a id="vec3multinner"></a>
### `Vec3MultInner`
`func Vec3MultInner(a, b *Vec3) float32`

Computes the dot product of two `Vec3` values.

<a id="vec3-multcross"></a>
### `(*Vec3).MultCross`
`func (r *Vec3) MultCross(a, b *Vec3)`

Computes the vector cross product `a x b`.

<a id="vec3-reflect"></a>
### `(*Vec3).Reflect`
`func (r *Vec3) Reflect(v, n *Vec3)`

Reflects vector `v` around normal `n` and stores the result in the receiver.

<a id="vec3-quatmultvec3"></a>
### `(*Vec3).QuatMultVec3`
`func (r *Vec3) QuatMultVec3(q *Quat, v *Vec3)`

Applies the rotation represented by a quaternion to a vector.

<a id="vec4-add"></a>
### `(*Vec4).Add`
`func (r *Vec4) Add(a, b *Vec4)`

Adds two `Vec4` values.

<a id="vec4-sub"></a>
### `(*Vec4).Sub`
`func (r *Vec4) Sub(a, b *Vec4)`

Subtracts two `Vec4` values.

<a id="vec4-subvec3"></a>
### `(*Vec4).SubVec3`
`func (r *Vec4) SubVec3(a *Vec4, b *Vec3)`

Subtracts a `Vec3` from the first three components of `Vec4`; the fourth component remains based on the receiver state.

<a id="vec4-scale"></a>
### `(*Vec4).Scale`
`func (r *Vec4) Scale(v *Vec4, s float32)`

Multiplies a 4D vector by a scalar.

<a id="vec4-len"></a>
### `(*Vec4).Len`
`func (v *Vec4) Len() float32`

Returns the length of a 4D vector.

<a id="vec4-norm"></a>
### `(*Vec4).Norm`
`func (r *Vec4) Norm(v *Vec4)`

Normalizes a 4D vector.

<a id="vec4-min"></a>
### `(*Vec4).Min`
`func (r *Vec4) Min(a, b *Vec4)`

Selects the component-wise minimum of two `Vec4` values.

<a id="vec4-max"></a>
### `(*Vec4).Max`
`func (r *Vec4) Max(a, b *Vec4)`

Selects the component-wise maximum of two `Vec4` values.

<a id="vec4multinner"></a>
### `Vec4MultInner`
`func Vec4MultInner(a, b *Vec4) float32`

Computes the full dot product of two `Vec4` values.

<a id="vec4multinner3"></a>
### `Vec4MultInner3`
`func Vec4MultInner3(a, b *Vec4) float32`

Computes the dot product of only the first three `Vec4` components.

<a id="vec4-multcross"></a>
### `(*Vec4).MultCross`
`func (r *Vec4) MultCross(a, b *Vec4)`

Computes the cross product of the first three components and sets the fourth component to `1`.

<a id="vec4-reflect"></a>
### `(*Vec4).Reflect`
`func (r *Vec4) Reflect(v, n *Vec4)`

Reflects a 4D vector around the given normal.

<a id="vec4-mat4x4row"></a>
### `(*Vec4).Mat4x4Row`
`func (r *Vec4) Mat4x4Row(m *Mat4x4, i int)`

Loads the `i`-th matrix row into the receiver.

<a id="vec4-mat4x4col"></a>
### `(*Vec4).Mat4x4Col`
`func (r *Vec4) Mat4x4Col(m *Mat4x4, i int)`

Loads the `i`-th matrix column into the receiver.

<a id="vec4-mat4x4multvec4"></a>
### `(*Vec4).Mat4x4MultVec4`
`func (r *Vec4) Mat4x4MultVec4(m *Mat4x4, v Vec4)`

Multiplies a 4x4 matrix by vector `v` and stores the result in the receiver.

<a id="vec4-quatmultvec4"></a>
### `(*Vec4).QuatMultVec4`
`func (r *Vec4) QuatMultVec4(q *Quat, v *Vec4)`

Applies a quaternion rotation to the first three components of `Vec4`.

<a id="quat-identity"></a>
### `(*Quat).Identity`
`func (q *Quat) Identity()`

Sets the receiver to the identity quaternion.

<a id="quat-add"></a>
### `(*Quat).Add`
`func (r *Quat) Add(a, b *Quat)`

Adds two quaternions component-wise.

<a id="quat-addvec3"></a>
### `(*Quat).AddVec3`
`func (r *Quat) AddVec3(a *Quat, v *Vec3)`

Adds a `Vec3` to the vector part of a quaternion.

<a id="quat-sub"></a>
### `(*Quat).Sub`
`func (r *Quat) Sub(a, b *Quat)`

Subtracts two quaternions component-wise.

<a id="quat-multcross3"></a>
### `(*Quat).MultCross3`
`func (r *Quat) MultCross3(a, b *Quat)`

Computes the cross product from the first three components of two quaternions.

<a id="quatmultinner3"></a>
### `QuatMultInner3`
`func QuatMultInner3(a, b *Quat) float32`

Returns the dot product of the vector parts of two quaternions.

<a id="quat-mult"></a>
### `(*Quat).Mult`
`func (r *Quat) Mult(p, q *Quat)`

Multiplies two quaternions and stores the result in the receiver.

<a id="quat-scale"></a>
### `(*Quat).Scale`
`func (r *Quat) Scale(q *Quat, s float32)`

Multiplies a quaternion by a scalar.

<a id="quatinnerproduct"></a>
### `QuatInnerProduct`
`func QuatInnerProduct(a, b *Quat) float32`

Returns the full dot product of two quaternions.

<a id="quat-conj"></a>
### `(*Quat).Conj`
`func (r *Quat) Conj(q *Quat)`

Creates the conjugate quaternion by negating the vector part.

<a id="quat-len"></a>
### `(*Quat).Len`
`func (q *Quat) Len() float32`

Returns the quaternion length.

<a id="quat-norm"></a>
### `(*Quat).Norm`
`func (r *Quat) Norm(q *Quat)`

Normalizes a quaternion to unit length.

<a id="quat-frommat4x4"></a>
### `(*Quat).FromMat4x4`
`func (q *Quat) FromMat4x4(m *Mat4x4)`

Derives a quaternion from the rotation part of a 4x4 matrix. If the matrix is degenerate, the method returns a fallback representation.

<a id="mat4x4-fill"></a>
### `(*Mat4x4).Fill`
`func (m *Mat4x4) Fill(d float32)`

Fills all 16 matrix elements with the same value.

<a id="mat4x4-identity"></a>
### `(*Mat4x4).Identity`
`func (m *Mat4x4) Identity()`

Sets the receiver to the identity matrix.

<a id="mat4x4-dup"></a>
### `(*Mat4x4).Dup`
`func (m *Mat4x4) Dup(n *Mat4x4)`

Copies the contents of another matrix into the receiver.

<a id="mat4x4-transpose"></a>
### `(*Mat4x4).Transpose`
`func (m *Mat4x4) Transpose(n *Mat4x4)`

Transposes the input matrix into the receiver.

<a id="mat4x4-add"></a>
### `(*Mat4x4).Add`
`func (m *Mat4x4) Add(a, b *Mat4x4)`

Adds two matrices component-wise.

<a id="mat4x4-sub"></a>
### `(*Mat4x4).Sub`
`func (m *Mat4x4) Sub(a, b *Mat4x4)`

Subtracts two matrices component-wise.

<a id="mat4x4-scale"></a>
### `(*Mat4x4).Scale`
`func (m *Mat4x4) Scale(a *Mat4x4, k float32)`

Multiplies a matrix by a scalar.

<a id="mat4x4-scaleaniso"></a>
### `(*Mat4x4).ScaleAniso`
`func (m *Mat4x4) ScaleAniso(a *Mat4x4, x, y, z float32)`

Applies non-uniform scaling along the X, Y, and Z axes.

<a id="mat4x4-mult"></a>
### `(*Mat4x4).Mult`
`func (m *Mat4x4) Mult(a, b *Mat4x4)`

Multiplies two 4x4 matrices in the library's internal layout convention.

<a id="mat4x4-translate"></a>
### `(*Mat4x4).Translate`
`func (m *Mat4x4) Translate(x, y, z float32)`

Sets the receiver to a pure translation matrix.

<a id="mat4x4-translateinplace"></a>
### `(*Mat4x4).TranslateInPlace`
`func (m *Mat4x4) TranslateInPlace(x, y, z float32)`

Adds a translation to an existing matrix without rebuilding it from scratch.

<a id="mat4x4-fromvec3multouter"></a>
### `(*Mat4x4).FromVec3MultOuter`
`func (m *Mat4x4) FromVec3MultOuter(a, b *Vec3)`

Creates a matrix from the outer product of two 3D vectors in the upper 3x3 block.

<a id="mat4x4-rotate"></a>
### `(*Mat4x4).Rotate`
`func (r *Mat4x4) Rotate(m *Mat4x4, x, y, z, angle float32)`

Applies a rotation around an arbitrary axis and stores the result in the receiver. The axis is normalized before the calculation.

<a id="mat4x4-rotatex"></a>
### `(*Mat4x4).RotateX`
`func (q *Mat4x4) RotateX(m *Mat4x4, angle float32)`

Applies a rotation around the X axis.

<a id="mat4x4-rotatey"></a>
### `(*Mat4x4).RotateY`
`func (q *Mat4x4) RotateY(m *Mat4x4, angle float32)`

Applies a rotation around the Y axis.

<a id="mat4x4-rotatez"></a>
### `(*Mat4x4).RotateZ`
`func (q *Mat4x4) RotateZ(m *Mat4x4, angle float32)`

Applies a rotation around the Z axis.

<a id="mat4x4-invert"></a>
### `(*Mat4x4).Invert`
`func (t *Mat4x4) Invert(m *Mat4x4)`

Computes the inverse of a 4x4 matrix and stores it in the receiver. It does not check for singular inputs, so it assumes the matrix is invertible.

<a id="mat4x4-orthonormalize"></a>
### `(*Mat4x4).OrthoNormalize`
`func (r *Mat4x4) OrthoNormalize(m *Mat4x4)`

Orthogonalizes and normalizes the upper 3x3 basis of the matrix. Useful for repairing numerically drifting transform matrices.

<a id="mat4x4-frustum"></a>
### `(*Mat4x4).Frustum`
`func (m *Mat4x4) Frustum(l, r, b, t, n, f float32)`

Sets the receiver to a perspective projection matrix defined by an explicit frustum.

<a id="mat4x4-ortho"></a>
### `(*Mat4x4).Ortho`
`func (m *Mat4x4) Ortho(l, r, b, t, n, f float32)`

Sets the receiver to an orthographic projection matrix.

<a id="mat4x4-perspective"></a>
### `(*Mat4x4).Perspective`
`func (m *Mat4x4) Perspective(y_fov, aspect, n, f float32)`

Sets the receiver to a perspective projection matrix from vertical FOV and aspect ratio.

<a id="mat4x4-lookat"></a>
### `(*Mat4x4).LookAt`
`func (m *Mat4x4) LookAt(eye, center, up *Vec3)`

Builds a view matrix from camera position, target point, and up vector.

<a id="mat4x4-fromquat"></a>
### `(*Mat4x4).FromQuat`
`func (m *Mat4x4) FromQuat(q *Quat)`

Converts a quaternion into a 4x4 rotation matrix.

<a id="mat4x4-multquat"></a>
### `(*Mat4x4).MultQuat`
`func (r *Mat4x4) MultQuat(m *Mat4x4, q *Quat)`

Applies a quaternion rotation to the orientation part of a matrix.

<a id="mat4x4-data"></a>
### `(*Mat4x4).Data`
`func (m *Mat4x4) Data() []byte`

Returns a byte view over the matrix without copying. Useful for direct upload into a uniform buffer.

<a id="invertmatrix"></a>
### `InvertMatrix`
`func InvertMatrix(m *Mat4x4) Mat4x4`

Returns the inverse of a matrix as a new value. If the input is singular, it returns the identity matrix instead of an undefined result.

<a id="dumpmatrix"></a>
### `DumpMatrix`
`func DumpMatrix(m *Mat4x4, note string) string`

Creates a readable text dump of a matrix. The `note` parameter is included as a label.

<a id="arrayfloat32-sizeof"></a>
### `(ArrayFloat32).Sizeof`
`func (a ArrayFloat32) Sizeof() int`

Returns the size of the `float32` array in bytes.

<a id="arrayfloat32-data"></a>
### `(ArrayFloat32).Data`
`func (a ArrayFloat32) Data() []byte`

Returns a byte view over the `ArrayFloat32` data without copying.

## Rasterization

<a id="newrasterpass"></a>
### `NewRasterPass`
`func NewRasterPass(device vk.Device, displayFormat vk.Format) (VulkanRasterPassInfo, error)`

Creates a simple render pass with a single color attachment and presentation to the swapchain. This is the basic helper for rasterization pipelines.

<a id="newrasterpasswithdepth"></a>
### `NewRasterPassWithDepth`
`func NewRasterPassWithDepth(device vk.Device, displayFormat vk.Format, depthFormat vk.Format) (VulkanRasterPassInfo, error)`

Extends `NewRasterPass` with a depth attachment and depth/stencil layout setup. Suitable for common 3D rendering.

<a id="vulkanrasterpassinfo-getrenderpass"></a>
### `(*VulkanRasterPassInfo).GetRenderPass`
`func (r *VulkanRasterPassInfo) GetRenderPass() vk.RenderPass`

Returns the `vk.RenderPass` handle wrapped by the helper type.

<a id="vulkanrasterpassinfo-destroy"></a>
### `(*VulkanRasterPassInfo).Destroy`
`func (r *VulkanRasterPassInfo) Destroy()`

Destroys the render pass if it is valid. It is safe to call even on a null or already cleaned-up state.

<a id="newgraphicspipelinewithoptions"></a>
### `NewGraphicsPipelineWithOptions`
`func NewGraphicsPipelineWithOptions(device vk.Device, displaySize vk.Extent2D, renderPass vk.RenderPass, opts PipelineOptions) (PipelineRasterizationInfo, error)`

Creates a full graphics pipeline including the layout and pipeline cache. Behavior is controlled by `PipelineOptions`, which configure shaders, descriptor set layouts, vertex layout, and depth testing.

<a id="pipelinerasterizationinfo-getlayout"></a>
### `(*PipelineRasterizationInfo).GetLayout`
`func (gfx *PipelineRasterizationInfo) GetLayout() vk.PipelineLayout`

Returns the `vk.PipelineLayout` created together with the pipeline.

<a id="pipelinerasterizationinfo-getpipeline"></a>
### `(*PipelineRasterizationInfo).GetPipeline`
`func (gfx *PipelineRasterizationInfo) GetPipeline() vk.Pipeline`

Returns the graphics pipeline handle itself.

<a id="pipelinerasterizationinfo-destroy"></a>
### `(*PipelineRasterizationInfo).Destroy`
`func (gfx *PipelineRasterizationInfo) Destroy()`

Destroys the pipeline, cache, and layout. The caller must ensure the GPU is no longer using them.

## Raytracing

<a id="raytracingextensions"></a>
### `RaytracingExtensions`
`func RaytracingExtensions() []string`

Returns a copy of the device extensions required by the ray tracing helpers in the library. The resulting slice can be further extended or filtered.

<a id="decodegltftexture"></a>
### `DecodeGLTFTexture`
`func DecodeGLTFTexture(doc *gltf.Document, baseDir string, imageIndex int) ([]byte, uint32, uint32, error)`

Decodes an image from a glTF document into tightly packed RGBA pixels and also returns its width and height. It supports embedded and external files, but not bufferView-backed image data.

<a id="loadgltftextures"></a>
### `LoadGLTFTextures`
`func LoadGLTFTextures(dev vk.Device, gpu vk.PhysicalDevice, queue vk.Queue, cmdCtx *CommandContext, doc *gltf.Document, baseDir string) ([]VulkanImageResource, error)`

Uploads all textures from a glTF document into Vulkan image resources. Index `0` always contains a 1x1 white fallback texture for simpler shader indexing.

<a id="newgltfmodel"></a>
### `NewGLTFModel`
`func NewGLTFModel(device vk.Device, primitives []GLTFPrimitive, geometryBuffer VulkanBufferResource, blas AccelerationStructure, textures []VulkanImageResource) GLTFModel`

Builds a `GLTFModel` from already created GPU resources without doing additional work. This is a pure value constructor.

<a id="gltfmodel-destroy"></a>
### `(*GLTFModel).Destroy`
`func (m *GLTFModel) Destroy()`

Frees all primitive vertex/index buffers, textures, the geometry buffer, and the BLAS. This is the complete teardown path for a ray tracing model.

<a id="loadgltfmodel"></a>
### `LoadGLTFModel`
`func LoadGLTFModel(dev vk.Device, gpu vk.PhysicalDevice, queue vk.Queue, cmdCtx *CommandContext, path string) (GLTFModel, error)`

Loads a glTF scene, creates vertex/index buffers for each primitive, uploads textures, builds a geometry SSBO, and constructs one BLAS for the whole model. This is the main high-level loader for RT examples.

<a id="accelerationstructure-getdeviceaddress"></a>
### `(*AccelerationStructure).GetDeviceAddress`
`func (a *AccelerationStructure) GetDeviceAddress() vk.DeviceAddress`

Returns the acceleration structure device address and lazily fetches it from Vulkan on the first call. The result is then cached.

<a id="accelerationstructure-destroy"></a>
### `(*AccelerationStructure).Destroy`
`func (a *AccelerationStructure) Destroy()`

First destroys the acceleration structure handle and only then the backing buffer. This order matters for valid Vulkan teardown.

<a id="newbufferwithdeviceaddress"></a>
### `NewBufferWithDeviceAddress`
`func NewBufferWithDeviceAddress(device vk.Device, gpu vk.PhysicalDevice, usage vk.BufferUsageFlags, size uint64, data unsafe.Pointer) (vk.Buffer, vk.DeviceMemory, error)`

Creates a host-visible buffer with shader device address support and optionally copies initial data into it. This is a lower-level helper for RT buffers outside `VulkanBufferResource`.

<a id="newdevicelocalbuffer"></a>
### `NewDeviceLocalBuffer`
`func NewDeviceLocalBuffer(device vk.Device, gpu vk.PhysicalDevice, usage vk.BufferUsageFlags, size uint64) (vk.Buffer, vk.DeviceMemory, error)`

Creates a device-local buffer with shader device address support. Unlike `NewBufferWithDeviceAddress`, it does not perform a CPU-side upload.

<a id="getbufferdeviceaddress"></a>
### `GetBufferDeviceAddress`
`func GetBufferDeviceAddress(device vk.Device, buf vk.Buffer) vk.DeviceAddress`

Returns the device address of the given Vulkan buffer. Mainly used when building acceleration structures and shader binding tables.
