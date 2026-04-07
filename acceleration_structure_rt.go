package ash

import vk "github.com/tomas-mraz/vulkan"

// AccelerationStructure owns a Manager acceleration structure handle,
// its backing buffer, and optional device address.
// Works for both BLAS (bottom-level) and TLAS (top-level).
type AccelerationStructure struct {
	device                vk.Device
	AccelerationStructure vk.AccelerationStructure
	Buffer                VulkanBufferResource
	Type                  vk.AccelerationStructureType
	DeviceAddress         vk.DeviceAddress
}

// GetDeviceAddress returns the cached device address, querying it on first call.
func (a *AccelerationStructure) GetDeviceAddress() vk.DeviceAddress {
	if a.DeviceAddress == 0 && a.AccelerationStructure != vk.AccelerationStructure(vk.NullHandle) {
		a.DeviceAddress = vk.GetAccelerationStructureDeviceAddress(a.device, &vk.AccelerationStructureDeviceAddressInfo{
			SType:                 vk.StructureTypeAccelerationStructureDeviceAddressInfo,
			AccelerationStructure: a.AccelerationStructure,
		})
	}
	return a.DeviceAddress
}

// Destroy releases the acceleration structure handle first, then the backing buffer.
// Order matters: the AS handle must be destroyed before the buffer it references.
func (a *AccelerationStructure) Destroy() {
	if a == nil {
		return
	}
	if a.AccelerationStructure != vk.AccelerationStructure(vk.NullHandle) {
		vk.DestroyAccelerationStructure(a.device, a.AccelerationStructure, nil)
		a.AccelerationStructure = vk.AccelerationStructure(vk.NullHandle)
	}
	a.Buffer.Destroy()
	a.DeviceAddress = 0
}
