package ash

import (
	"fmt"
	"unsafe"

	vk "github.com/tomas-mraz/vulkan"
)

// ShaderBindingTable holds a shader binding table buffer and the strided device address
// regions for each shader group type (raygen, miss, hit, callable).
type ShaderBindingTable struct {
	device   vk.Device
	Buffer   BufferResource
	Raygen   vk.StridedDeviceAddressRegion
	Miss     vk.StridedDeviceAddressRegion
	Hit      vk.StridedDeviceAddressRegion
	Callable vk.StridedDeviceAddressRegion
}

// NewShaderBindingTable creates a shader binding table from a ray tracing pipeline.
// Groups are laid out sequentially: raygen, miss, hit, callable.
// handleSize and handleAlignment come from VkPhysicalDeviceRayTracingPipelinePropertiesKHR.
func NewShaderBindingTable(device vk.Device, gpu vk.PhysicalDevice, pipeline vk.Pipeline,
	handleSize, handleAlignment uint32,
	raygenCount, missCount, hitCount, callableCount uint32,
) (ShaderBindingTable, error) {
	var s ShaderBindingTable
	s.device = device

	groupCount := raygenCount + missCount + hitCount + callableCount
	handleSizeAligned := AlignUp(handleSize, handleAlignment)
	sbtSize := groupCount * handleSizeAligned

	handleStorage := make([]byte, sbtSize)
	if err := vk.Error(vk.GetRayTracingShaderGroupHandles(device, pipeline, 0, groupCount, uint64(sbtSize), unsafe.Pointer(&handleStorage[0]))); err != nil {
		return s, fmt.Errorf("vk.GetRayTracingShaderGroupHandles failed with %s", err)
	}

	buf, err := NewBufferHostVisible(device, gpu, handleStorage, true,
		vk.BufferUsageFlags(vk.BufferUsageShaderBindingTableBit))
	if err != nil {
		return s, fmt.Errorf("create SBT buffer: %w", err)
	}
	s.Buffer = buf

	addr := buf.DeviceAddress
	stride := vk.DeviceSize(handleSizeAligned)
	offset := vk.DeviceAddress(0)

	s.Raygen = vk.StridedDeviceAddressRegion{
		DeviceAddress: addr + offset,
		Stride:        stride,
		Size:          vk.DeviceSize(raygenCount) * stride,
	}
	offset += vk.DeviceAddress(raygenCount) * vk.DeviceAddress(handleSizeAligned)

	s.Miss = vk.StridedDeviceAddressRegion{
		DeviceAddress: addr + offset,
		Stride:        stride,
		Size:          vk.DeviceSize(missCount) * stride,
	}
	offset += vk.DeviceAddress(missCount) * vk.DeviceAddress(handleSizeAligned)

	s.Hit = vk.StridedDeviceAddressRegion{
		DeviceAddress: addr + offset,
		Stride:        stride,
		Size:          vk.DeviceSize(hitCount) * stride,
	}
	offset += vk.DeviceAddress(hitCount) * vk.DeviceAddress(handleSizeAligned)

	if callableCount > 0 {
		s.Callable = vk.StridedDeviceAddressRegion{
			DeviceAddress: addr + offset,
			Stride:        stride,
			Size:          vk.DeviceSize(callableCount) * stride,
		}
	}

	return s, nil
}

func (s *ShaderBindingTable) Destroy() {
	if s == nil {
		return
	}
	s.Buffer.Destroy()
}
