package ash

import (
	"fmt"
	"unsafe"

	vk "github.com/tomas-mraz/vulkan"
)

// VulkanDescriptorInfo manages a descriptor set layout, pool, and allocated sets.
type VulkanDescriptorInfo struct {
	device vk.Device
	layout vk.DescriptorSetLayout
	pool   vk.DescriptorPool
	sets   []vk.DescriptorSet
}

// GetLayout returns the descriptor set layout.
func (d *VulkanDescriptorInfo) GetLayout() vk.DescriptorSetLayout {
	return d.layout
}

// GetSets returns all allocated descriptor sets.
func (d *VulkanDescriptorInfo) GetSets() []vk.DescriptorSet {
	return d.sets
}

// Destroy releases the descriptor pool and layout.
func (d *VulkanDescriptorInfo) Destroy() {
	if d == nil {
		return
	}
	vk.DestroyDescriptorPool(d.device, d.pool, nil)
	vk.DestroyDescriptorSetLayout(d.device, d.layout, nil)
}

// DescriptorBinding describes a single binding in a descriptor set.
// Implement this interface with one of the concrete binding types:
// BindingAccelerationStructure, BindingStorageImage, BindingUniformBuffer,
// BindingStorageBuffer, BindingImageSampler, BindingImageSamplerArray.
type DescriptorBinding interface {
	layoutBinding(index uint32) vk.DescriptorSetLayoutBinding
	poolSize(setCount uint32) vk.DescriptorPoolSize
	writeSet(set vk.DescriptorSet, index uint32, setIndex uint32) vk.WriteDescriptorSet
}

// BindingAccelerationStructure binds a top-level acceleration structure.
type BindingAccelerationStructure struct {
	StageFlags            vk.ShaderStageFlags
	AccelerationStructure vk.AccelerationStructure
	asWriteInfo           vk.WriteDescriptorSetAccelerationStructure
}

func (b *BindingAccelerationStructure) layoutBinding(index uint32) vk.DescriptorSetLayoutBinding {
	return vk.DescriptorSetLayoutBinding{
		Binding: index, DescriptorType: vk.DescriptorTypeAccelerationStructure,
		DescriptorCount: 1, StageFlags: b.StageFlags,
	}
}

func (b *BindingAccelerationStructure) poolSize(setCount uint32) vk.DescriptorPoolSize {
	return vk.DescriptorPoolSize{Type: vk.DescriptorTypeAccelerationStructure, DescriptorCount: setCount}
}

func (b *BindingAccelerationStructure) writeSet(set vk.DescriptorSet, index uint32, _ uint32) vk.WriteDescriptorSet {
	b.asWriteInfo = vk.WriteDescriptorSetAccelerationStructure{
		SType:                      vk.StructureTypeWriteDescriptorSetAccelerationStructure,
		AccelerationStructureCount: 1,
		PAccelerationStructures:    []vk.AccelerationStructure{b.AccelerationStructure},
	}
	return vk.WriteDescriptorSet{
		SType: vk.StructureTypeWriteDescriptorSet, DstSet: set, DstBinding: index,
		DescriptorCount: 1, DescriptorType: vk.DescriptorTypeAccelerationStructure,
		PNext: unsafe.Pointer(&b.asWriteInfo),
	}
}

// BindingStorageImage binds a storage image (e.g. ray tracing output).
type BindingStorageImage struct {
	StageFlags vk.ShaderStageFlags
	Resource   *ImageResource
	ImageView  vk.ImageView
	Layout     vk.ImageLayout // 0 defaults to General
}

func (b *BindingStorageImage) layoutBinding(index uint32) vk.DescriptorSetLayoutBinding {
	return vk.DescriptorSetLayoutBinding{
		Binding: index, DescriptorType: vk.DescriptorTypeStorageImage,
		DescriptorCount: 1, StageFlags: b.StageFlags,
	}
}

func (b *BindingStorageImage) poolSize(setCount uint32) vk.DescriptorPoolSize {
	return vk.DescriptorPoolSize{Type: vk.DescriptorTypeStorageImage, DescriptorCount: setCount}
}

func (b *BindingStorageImage) writeSet(set vk.DescriptorSet, index uint32, _ uint32) vk.WriteDescriptorSet {
	info := vk.DescriptorImageInfo{
		ImageView:   b.ImageView,
		ImageLayout: b.Layout,
	}
	if b.Resource != nil {
		info = b.Resource.StorageDescriptorInfo()
	} else if info.ImageLayout == 0 {
		info.ImageLayout = vk.ImageLayoutGeneral
	}
	return vk.WriteDescriptorSet{
		SType: vk.StructureTypeWriteDescriptorSet, DstSet: set, DstBinding: index,
		DescriptorCount: 1, DescriptorType: vk.DescriptorTypeStorageImage,
		PImageInfo: []vk.DescriptorImageInfo{info},
	}
}

// BindingUniformBuffer binds per-frame uniform buffers.
// Each descriptor set gets the buffer at its swapchain index.
type BindingUniformBuffer struct {
	StageFlags vk.ShaderStageFlags
	Uniforms   *VulkanUniformBuffers
}

func (b *BindingUniformBuffer) layoutBinding(index uint32) vk.DescriptorSetLayoutBinding {
	return vk.DescriptorSetLayoutBinding{
		Binding: index, DescriptorType: vk.DescriptorTypeUniformBuffer,
		DescriptorCount: 1, StageFlags: b.StageFlags,
		PImmutableSamplers: []vk.Sampler{vk.NullSampler},
	}
}

func (b *BindingUniformBuffer) poolSize(setCount uint32) vk.DescriptorPoolSize {
	return vk.DescriptorPoolSize{Type: vk.DescriptorTypeUniformBuffer, DescriptorCount: setCount}
}

func (b *BindingUniformBuffer) writeSet(set vk.DescriptorSet, index uint32, setIndex uint32) vk.WriteDescriptorSet {
	return vk.WriteDescriptorSet{
		SType: vk.StructureTypeWriteDescriptorSet, DstSet: set, DstBinding: index,
		DescriptorCount: 1, DescriptorType: vk.DescriptorTypeUniformBuffer,
		PBufferInfo: []vk.DescriptorBufferInfo{{
			Buffer: b.Uniforms.GetBuffer(setIndex), Offset: 0,
			Range: vk.DeviceSize(b.Uniforms.GetSize()),
		}},
	}
}

// BindingStorageBuffer binds a storage buffer (e.g. geometry node SSBO).
type BindingStorageBuffer struct {
	StageFlags vk.ShaderStageFlags
	Buffer     vk.Buffer
	Range      vk.DeviceSize // 0 = WholeSize
}

func (b *BindingStorageBuffer) layoutBinding(index uint32) vk.DescriptorSetLayoutBinding {
	return vk.DescriptorSetLayoutBinding{
		Binding: index, DescriptorType: vk.DescriptorTypeStorageBuffer,
		DescriptorCount: 1, StageFlags: b.StageFlags,
	}
}

func (b *BindingStorageBuffer) poolSize(setCount uint32) vk.DescriptorPoolSize {
	return vk.DescriptorPoolSize{Type: vk.DescriptorTypeStorageBuffer, DescriptorCount: setCount}
}

func (b *BindingStorageBuffer) writeSet(set vk.DescriptorSet, index uint32, _ uint32) vk.WriteDescriptorSet {
	r := b.Range
	if r == 0 {
		r = vk.DeviceSize(vk.WholeSize)
	}
	return vk.WriteDescriptorSet{
		SType: vk.StructureTypeWriteDescriptorSet, DstSet: set, DstBinding: index,
		DescriptorCount: 1, DescriptorType: vk.DescriptorTypeStorageBuffer,
		PBufferInfo: []vk.DescriptorBufferInfo{{Buffer: b.Buffer, Offset: 0, Range: r}},
	}
}

// BindingImageSampler binds a single combined image sampler.
type BindingImageSampler struct {
	StageFlags        vk.ShaderStageFlags
	Resource          *ImageResource
	ImageView         vk.ImageView
	Sampler           vk.Sampler
	Layout            vk.ImageLayout // 0 defaults to ShaderReadOnlyOptimal
	ImmutableSamplers []vk.Sampler
}

func (b *BindingImageSampler) layoutBinding(index uint32) vk.DescriptorSetLayoutBinding {
	lb := vk.DescriptorSetLayoutBinding{
		Binding: index, DescriptorType: vk.DescriptorTypeCombinedImageSampler,
		DescriptorCount: 1, StageFlags: b.StageFlags,
	}
	if len(b.ImmutableSamplers) > 0 {
		lb.PImmutableSamplers = b.ImmutableSamplers
	}
	return lb
}

func (b *BindingImageSampler) poolSize(setCount uint32) vk.DescriptorPoolSize {
	return vk.DescriptorPoolSize{Type: vk.DescriptorTypeCombinedImageSampler, DescriptorCount: setCount}
}

func (b *BindingImageSampler) writeSet(set vk.DescriptorSet, index uint32, _ uint32) vk.WriteDescriptorSet {
	info := vk.DescriptorImageInfo{
		Sampler:     b.Sampler,
		ImageView:   b.ImageView,
		ImageLayout: b.Layout,
	}
	if b.Resource != nil {
		info = b.Resource.SampledDescriptorInfo()
	} else if info.ImageLayout == 0 {
		info.ImageLayout = vk.ImageLayoutShaderReadOnlyOptimal
	}
	return vk.WriteDescriptorSet{
		SType: vk.StructureTypeWriteDescriptorSet, DstSet: set, DstBinding: index,
		DescriptorCount: 1, DescriptorType: vk.DescriptorTypeCombinedImageSampler,
		PImageInfo: []vk.DescriptorImageInfo{info},
	}
}

// BindingImageSamplerArray binds an array of combined image samplers (e.g. texture array for RT).
type BindingImageSamplerArray struct {
	StageFlags        vk.ShaderStageFlags
	ImageInfos        []vk.DescriptorImageInfo
	ImmutableSamplers []vk.Sampler
}

// NewBindingStorageImage creates a storage-image descriptor binding from an ImageResource.
func NewBindingStorageImage(stageFlags vk.ShaderStageFlags, resource *ImageResource) *BindingStorageImage {
	return &BindingStorageImage{
		StageFlags: stageFlags,
		Resource:   resource,
	}
}

// NewBindingImageSampler creates a combined-image-sampler descriptor binding from an ImageResource.
func NewBindingImageSampler(stageFlags vk.ShaderStageFlags, resource *ImageResource, immutableSamplers []vk.Sampler) *BindingImageSampler {
	return &BindingImageSampler{
		StageFlags:        stageFlags,
		Resource:          resource,
		ImmutableSamplers: immutableSamplers,
	}
}

func (b *BindingImageSamplerArray) layoutBinding(index uint32) vk.DescriptorSetLayoutBinding {
	lb := vk.DescriptorSetLayoutBinding{
		Binding: index, DescriptorType: vk.DescriptorTypeCombinedImageSampler,
		DescriptorCount: uint32(len(b.ImageInfos)), StageFlags: b.StageFlags,
	}
	if len(b.ImmutableSamplers) > 0 {
		lb.PImmutableSamplers = b.ImmutableSamplers
	}
	return lb
}

func (b *BindingImageSamplerArray) poolSize(setCount uint32) vk.DescriptorPoolSize {
	return vk.DescriptorPoolSize{
		Type:            vk.DescriptorTypeCombinedImageSampler,
		DescriptorCount: setCount * uint32(len(b.ImageInfos)),
	}
}

func (b *BindingImageSamplerArray) writeSet(set vk.DescriptorSet, index uint32, _ uint32) vk.WriteDescriptorSet {
	return vk.WriteDescriptorSet{
		SType: vk.StructureTypeWriteDescriptorSet, DstSet: set, DstBinding: index,
		DescriptorCount: uint32(len(b.ImageInfos)), DescriptorType: vk.DescriptorTypeCombinedImageSampler,
		PImageInfo: b.ImageInfos,
	}
}

// NewDescriptorSets creates a descriptor set layout, pool, and sets from a slice of bindings.
// Binding indices are assigned sequentially (0, 1, 2, ...).
func NewDescriptorSets(device vk.Device, setCount uint32, bindings []DescriptorBinding) (VulkanDescriptorInfo, error) {
	var d VulkanDescriptorInfo
	d.device = device

	// Layout
	layoutBindings := make([]vk.DescriptorSetLayoutBinding, len(bindings))
	for i, b := range bindings {
		layoutBindings[i] = b.layoutBinding(uint32(i))
	}
	if err := vk.Error(vk.CreateDescriptorSetLayout(device, &vk.DescriptorSetLayoutCreateInfo{
		SType:        vk.StructureTypeDescriptorSetLayoutCreateInfo,
		BindingCount: uint32(len(layoutBindings)),
		PBindings:    layoutBindings,
	}, nil, &d.layout)); err != nil {
		return d, fmt.Errorf("vk.CreateDescriptorSetLayout failed with %s", err)
	}

	// Pool
	poolSizes := make([]vk.DescriptorPoolSize, len(bindings))
	for i, b := range bindings {
		poolSizes[i] = b.poolSize(setCount)
	}
	if err := vk.Error(vk.CreateDescriptorPool(device, &vk.DescriptorPoolCreateInfo{
		SType:         vk.StructureTypeDescriptorPoolCreateInfo,
		MaxSets:       setCount,
		PoolSizeCount: uint32(len(poolSizes)),
		PPoolSizes:    poolSizes,
	}, nil, &d.pool)); err != nil {
		return d, fmt.Errorf("vk.CreateDescriptorPool failed with %s", err)
	}

	// Allocate sets
	d.sets = make([]vk.DescriptorSet, setCount)
	for i := uint32(0); i < setCount; i++ {
		if err := vk.Error(vk.AllocateDescriptorSets(device, &vk.DescriptorSetAllocateInfo{
			SType: vk.StructureTypeDescriptorSetAllocateInfo, DescriptorPool: d.pool,
			DescriptorSetCount: 1, PSetLayouts: []vk.DescriptorSetLayout{d.layout},
		}, &d.sets[i])); err != nil {
			return d, fmt.Errorf("vk.AllocateDescriptorSets failed with %s", err)
		}
	}

	// Write
	for i := uint32(0); i < setCount; i++ {
		writes := make([]vk.WriteDescriptorSet, len(bindings))
		for j, b := range bindings {
			writes[j] = b.writeSet(d.sets[i], uint32(j), i)
		}
		vk.UpdateDescriptorSets(device, uint32(len(writes)), writes, 0, nil)
	}
	return d, nil
}
