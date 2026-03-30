package ash

import (
	"bytes"
	"fmt"

	vk "github.com/tomas-mraz/vulkan"
)

const (
	end     = "\x00"
	endChar = '\x00'
)

var (
	rtExtensions = [...]string{
		"VK_KHR_acceleration_structure\x00",
		"VK_KHR_ray_tracing_pipeline\x00",
		"VK_KHR_buffer_device_address\x00",
		"VK_KHR_deferred_host_operations\x00",
		"VK_EXT_descriptor_indexing\x00",
		"VK_KHR_spirv_1_4\x00",
		"VK_KHR_shader_float_controls\x00",
	}
	androidExtensions = [...]string{
		"VK_KHR_surface\x00",
		"VK_KHR_android_surface\x00"}
)

func trimCString(slice []byte) string {
	return string(bytes.TrimRight(slice, end))
}

func MakeCString(s string) string {
	if len(s) == 0 {
		return end
	}
	if s[len(s)-1] != endChar {
		return s + end
	}
	return s
}

func RaytracingExtensions() []string {
	return append([]string(nil), rtExtensions[:]...)
}

func AndroidExtensions() []string {
	return append([]string(nil), androidExtensions[:]...)
}

// Cleanup collects cleanup functions and runs them in LIFO order.
// Register cleanup right after object creation with Add, then call Destroy at shutdown to tear down everything in reverse order.
type Cleanup struct {
	fns []Destroyer
}

// Destroyer represents a type that can release its owned resources.
type Destroyer interface {
	Destroy()
}

// Add registers a cleanup function. Functions run in reverse order on Destroy.
func (d *Cleanup) Add(obj Destroyer) {
	d.fns = append(d.fns, obj)
}

// Destroy runs all registered cleanup functions in LIFO order and clears the list.
func (d *Cleanup) Destroy() {
	for i := len(d.fns) - 1; i >= 0; i-- {
		d.fns[i].Destroy()
	}
	d.fns = nil
}

// LoadShaderFromBytes creates a shader module from raw SPIR-V bytecode.
func LoadShaderFromBytes(device vk.Device, data []byte) (vk.ShaderModule, error) {
	var module vk.ShaderModule
	shaderModuleCreateInfo := vk.ShaderModuleCreateInfo{
		SType:    vk.StructureTypeShaderModuleCreateInfo,
		CodeSize: uint64(len(data)),
		PCode:    repackUint32(data),
	}
	err := vk.Error(vk.CreateShaderModule(device, &shaderModuleCreateInfo, nil, &module))
	if err != nil {
		err = fmt.Errorf("vk.CreateShaderModule failed with %s", err)
		return module, err
	}
	return module, nil
}
