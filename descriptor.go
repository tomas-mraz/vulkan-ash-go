package ash

import (
	"fmt"

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

// NewDescriptorUBO creates descriptors with a single UBO binding at binding 0 (vertex stage).
func NewDescriptorUBO(device vk.Device, uniforms *VulkanUniformBuffers, count uint32) (VulkanDescriptorInfo, error) {
	var d VulkanDescriptorInfo
	d.device = device

	if err := vk.Error(vk.CreateDescriptorSetLayout(device, &vk.DescriptorSetLayoutCreateInfo{
		SType: vk.StructureTypeDescriptorSetLayoutCreateInfo, BindingCount: 1,
		PBindings: []vk.DescriptorSetLayoutBinding{
			{Binding: 0, DescriptorType: vk.DescriptorTypeUniformBuffer, DescriptorCount: 1,
				StageFlags: vk.ShaderStageFlags(vk.ShaderStageVertexBit), PImmutableSamplers: []vk.Sampler{vk.NullSampler}},
		},
	}, nil, &d.layout)); err != nil {
		return d, fmt.Errorf("vk.CreateDescriptorSetLayout failed with %s", err)
	}

	if err := vk.Error(vk.CreateDescriptorPool(device, &vk.DescriptorPoolCreateInfo{
		SType: vk.StructureTypeDescriptorPoolCreateInfo, MaxSets: count, PoolSizeCount: 1,
		PPoolSizes: []vk.DescriptorPoolSize{
			{Type: vk.DescriptorTypeUniformBuffer, DescriptorCount: count},
		},
	}, nil, &d.pool)); err != nil {
		return d, fmt.Errorf("vk.CreateDescriptorPool failed with %s", err)
	}

	d.sets = make([]vk.DescriptorSet, count)
	for i := uint32(0); i < count; i++ {
		if err := vk.Error(vk.AllocateDescriptorSets(device, &vk.DescriptorSetAllocateInfo{
			SType: vk.StructureTypeDescriptorSetAllocateInfo, DescriptorPool: d.pool,
			DescriptorSetCount: 1, PSetLayouts: []vk.DescriptorSetLayout{d.layout},
		}, &d.sets[i])); err != nil {
			return d, fmt.Errorf("vk.AllocateDescriptorSets failed with %s", err)
		}
	}

	for i := uint32(0); i < count; i++ {
		vk.UpdateDescriptorSets(device, 1, []vk.WriteDescriptorSet{
			{SType: vk.StructureTypeWriteDescriptorSet, DstSet: d.sets[i], DstBinding: 0, DescriptorCount: 1,
				DescriptorType: vk.DescriptorTypeUniformBuffer,
				PBufferInfo:    []vk.DescriptorBufferInfo{{Buffer: uniforms.GetBuffer(i), Offset: 0, Range: vk.DeviceSize(uniforms.GetSize())}}},
		}, 0, nil)
	}
	return d, nil
}

// NewDescriptorUBOTexture creates descriptors with UBO at binding 0 (vertex stage)
// and a combined image sampler at binding 1 (fragment stage).
func NewDescriptorUBOTexture(device vk.Device, uniforms *VulkanUniformBuffers, texture *VulkanImageResource, count uint32) (VulkanDescriptorInfo, error) {
	var d VulkanDescriptorInfo
	d.device = device

	if err := vk.Error(vk.CreateDescriptorSetLayout(device, &vk.DescriptorSetLayoutCreateInfo{
		SType: vk.StructureTypeDescriptorSetLayoutCreateInfo, BindingCount: 2,
		PBindings: []vk.DescriptorSetLayoutBinding{
			{Binding: 0, DescriptorType: vk.DescriptorTypeUniformBuffer, DescriptorCount: 1,
				StageFlags: vk.ShaderStageFlags(vk.ShaderStageVertexBit), PImmutableSamplers: []vk.Sampler{vk.NullSampler}},
			{Binding: 1, DescriptorType: vk.DescriptorTypeCombinedImageSampler, DescriptorCount: 1,
				StageFlags: vk.ShaderStageFlags(vk.ShaderStageFragmentBit), PImmutableSamplers: []vk.Sampler{texture.GetSampler()}},
		},
	}, nil, &d.layout)); err != nil {
		return d, fmt.Errorf("vk.CreateDescriptorSetLayout failed with %s", err)
	}

	if err := vk.Error(vk.CreateDescriptorPool(device, &vk.DescriptorPoolCreateInfo{
		SType: vk.StructureTypeDescriptorPoolCreateInfo, MaxSets: count, PoolSizeCount: 2,
		PPoolSizes: []vk.DescriptorPoolSize{
			{Type: vk.DescriptorTypeUniformBuffer, DescriptorCount: count},
			{Type: vk.DescriptorTypeCombinedImageSampler, DescriptorCount: count},
		},
	}, nil, &d.pool)); err != nil {
		return d, fmt.Errorf("vk.CreateDescriptorPool failed with %s", err)
	}

	d.sets = make([]vk.DescriptorSet, count)
	for i := uint32(0); i < count; i++ {
		if err := vk.Error(vk.AllocateDescriptorSets(device, &vk.DescriptorSetAllocateInfo{
			SType: vk.StructureTypeDescriptorSetAllocateInfo, DescriptorPool: d.pool,
			DescriptorSetCount: 1, PSetLayouts: []vk.DescriptorSetLayout{d.layout},
		}, &d.sets[i])); err != nil {
			return d, fmt.Errorf("vk.AllocateDescriptorSets failed with %s", err)
		}
	}

	for i := uint32(0); i < count; i++ {
		vk.UpdateDescriptorSets(device, 2, []vk.WriteDescriptorSet{
			{SType: vk.StructureTypeWriteDescriptorSet, DstSet: d.sets[i], DstBinding: 0, DescriptorCount: 1,
				DescriptorType: vk.DescriptorTypeUniformBuffer,
				PBufferInfo:    []vk.DescriptorBufferInfo{{Buffer: uniforms.GetBuffer(i), Offset: 0, Range: vk.DeviceSize(uniforms.GetSize())}}},
			{SType: vk.StructureTypeWriteDescriptorSet, DstSet: d.sets[i], DstBinding: 1, DescriptorCount: 1,
				DescriptorType: vk.DescriptorTypeCombinedImageSampler,
				PImageInfo:     []vk.DescriptorImageInfo{{Sampler: texture.GetSampler(), ImageView: texture.GetView(), ImageLayout: vk.ImageLayoutShaderReadOnlyOptimal}}},
		}, 0, nil)
	}
	return d, nil
}
