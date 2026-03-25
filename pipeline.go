package ash

import (
	"fmt"

	vk "github.com/tomas-mraz/vulkan"
)

// PipelineOptions configures optional pipeline parameters.
type PipelineOptions struct {
	// VertShaderData is raw SPIR-V bytecode for the vertex shader.
	// If nil, the default embedded shader "shaders/tri-vert.spv" is used.
	VertShaderData []byte
	// FragShaderData is raw SPIR-V bytecode for the fragment shader.
	// If nil, the default embedded shader "shaders/tri-frag.spv" is used.
	FragShaderData []byte
	// PushConstantRanges defines push constant ranges for the pipeline layout.
	// If nil, no push constants are used.
	PushConstantRanges []vk.PushConstantRange
	// VertexBindings defines custom vertex input bindings.
	// If nil, default single binding (stride 12, vec3 position) is used.
	VertexBindings []vk.VertexInputBindingDescription
	// VertexAttributes defines custom vertex input attributes.
	// If nil, default single attribute (location 0, R32G32B32Sfloat) is used.
	VertexAttributes []vk.VertexInputAttributeDescription
	// DescriptorSetLayouts defines descriptor set layouts for the pipeline layout.
	// If nil, no descriptor sets are used.
	DescriptorSetLayouts []vk.DescriptorSetLayout
	// DepthTestEnable enables depth testing and writing.
	DepthTestEnable bool
}

type VulkanGfxPipelineInfo struct {
	device   vk.Device
	layout   vk.PipelineLayout
	cache    vk.PipelineCache
	pipeline vk.Pipeline
}

// NewGraphicsPipelineWithOptions creates a graphics pipeline with custom shaders and push constants.
func NewGraphicsPipelineWithOptions(device vk.Device, displaySize vk.Extent2D, renderPass vk.RenderPass, opts PipelineOptions) (VulkanGfxPipelineInfo, error) {
	var gfxPipeline VulkanGfxPipelineInfo

	// Pipeline layout
	pipelineLayoutCreateInfo := vk.PipelineLayoutCreateInfo{
		SType:                  vk.StructureTypePipelineLayoutCreateInfo,
		PushConstantRangeCount: uint32(len(opts.PushConstantRanges)),
		PPushConstantRanges:    opts.PushConstantRanges,
		SetLayoutCount:         uint32(len(opts.DescriptorSetLayouts)),
		PSetLayouts:            opts.DescriptorSetLayouts,
	}
	err := vk.Error(vk.CreatePipelineLayout(device, &pipelineLayoutCreateInfo, nil, &gfxPipeline.layout))
	if err != nil {
		err = fmt.Errorf("vk.CreatePipelineLayout failed with %s", err)
		return gfxPipeline, err
	}

	// Load shaders
	var vertexShader, fragmentShader vk.ShaderModule
	if opts.VertShaderData == nil {
		return gfxPipeline, fmt.Errorf("VertShaderData is required")
	}
	vertexShader, err = LoadShaderFromBytes(device, opts.VertShaderData)
	if err != nil {
		return gfxPipeline, err
	}
	defer vk.DestroyShaderModule(device, vertexShader, nil)

	if opts.FragShaderData == nil {
		return gfxPipeline, fmt.Errorf("FragShaderData is required")
	}
	fragmentShader, err = LoadShaderFromBytes(device, opts.FragShaderData)
	if err != nil {
		return gfxPipeline, err
	}
	defer vk.DestroyShaderModule(device, fragmentShader, nil)

	shaderStages := []vk.PipelineShaderStageCreateInfo{
		{
			SType:  vk.StructureTypePipelineShaderStageCreateInfo,
			Stage:  vk.ShaderStageVertexBit,
			Module: vertexShader,
			PName:  []byte("main\x00"),
		},
		{
			SType:  vk.StructureTypePipelineShaderStageCreateInfo,
			Stage:  vk.ShaderStageFragmentBit,
			Module: fragmentShader,
			PName:  []byte("main\x00"),
		},
	}

	// Viewport
	viewports := []vk.Viewport{{
		MinDepth: 0.0,
		MaxDepth: 1.0,
		X:        0,
		Y:        0,
		Width:    float32(displaySize.Width),
		Height:   float32(displaySize.Height),
	}}
	scissors := []vk.Rect2D{{
		Extent: displaySize,
		Offset: vk.Offset2D{
			X: 0, Y: 0,
		},
	}}
	viewportState := vk.PipelineViewportStateCreateInfo{
		SType:         vk.StructureTypePipelineViewportStateCreateInfo,
		ViewportCount: 1,
		PViewports:    viewports,
		ScissorCount:  1,
		PScissors:     scissors,
	}

	// Fixed function state
	sampleMask := []vk.SampleMask{vk.SampleMask(vk.MaxUint32)}
	multisampleState := vk.PipelineMultisampleStateCreateInfo{
		SType:                vk.StructureTypePipelineMultisampleStateCreateInfo,
		RasterizationSamples: vk.SampleCount1Bit,
		SampleShadingEnable:  vk.False,
		PSampleMask:          sampleMask,
	}
	colorBlendAttachment := []vk.PipelineColorBlendAttachmentState{{
		ColorWriteMask: vk.ColorComponentFlags(
			vk.ColorComponentRBit | vk.ColorComponentGBit |
				vk.ColorComponentBBit | vk.ColorComponentABit,
		),
		BlendEnable: vk.False,
	}}
	colorBlendState := vk.PipelineColorBlendStateCreateInfo{
		SType:           vk.StructureTypePipelineColorBlendStateCreateInfo,
		LogicOpEnable:   vk.False,
		LogicOp:         vk.LogicOpCopy,
		AttachmentCount: 1,
		PAttachments:    colorBlendAttachment,
	}
	rasterState := vk.PipelineRasterizationStateCreateInfo{
		SType:                   vk.StructureTypePipelineRasterizationStateCreateInfo,
		DepthClampEnable:        vk.False,
		RasterizerDiscardEnable: vk.False,
		PolygonMode:             vk.PolygonModeFill,
		CullMode:                vk.CullModeFlags(vk.CullModeNone),
		FrontFace:               vk.FrontFaceClockwise,
		DepthBiasEnable:         vk.False,
		LineWidth:               1,
	}
	inputAssemblyState := vk.PipelineInputAssemblyStateCreateInfo{
		SType:                  vk.StructureTypePipelineInputAssemblyStateCreateInfo,
		Topology:               vk.PrimitiveTopologyTriangleList,
		PrimitiveRestartEnable: vk.False,
	}
	vertexInputBindings := opts.VertexBindings
	if vertexInputBindings == nil {
		vertexInputBindings = []vk.VertexInputBindingDescription{{
			Binding:   0,
			Stride:    3 * 4,
			InputRate: vk.VertexInputRateVertex,
		}}
	}
	vertexInputAttributes := opts.VertexAttributes
	if vertexInputAttributes == nil {
		vertexInputAttributes = []vk.VertexInputAttributeDescription{{
			Binding:  0,
			Location: 0,
			Format:   vk.FormatR32g32b32Sfloat,
			Offset:   0,
		}}
	}
	vertexInputState := vk.PipelineVertexInputStateCreateInfo{
		SType:                           vk.StructureTypePipelineVertexInputStateCreateInfo,
		VertexBindingDescriptionCount:   uint32(len(vertexInputBindings)),
		PVertexBindingDescriptions:      vertexInputBindings,
		VertexAttributeDescriptionCount: uint32(len(vertexInputAttributes)),
		PVertexAttributeDescriptions:    vertexInputAttributes,
	}
	dynamicState := vk.PipelineDynamicStateCreateInfo{
		SType: vk.StructureTypePipelineDynamicStateCreateInfo,
	}

	// Pipeline cache and creation
	pipelineCacheInfo := vk.PipelineCacheCreateInfo{
		SType: vk.StructureTypePipelineCacheCreateInfo,
	}
	err = vk.Error(vk.CreatePipelineCache(device, &pipelineCacheInfo, nil, &gfxPipeline.cache))
	if err != nil {
		err = fmt.Errorf("vk.CreatePipelineCache failed with %s", err)
		return gfxPipeline, err
	}
	pipelineCreateInfo := vk.GraphicsPipelineCreateInfo{
		SType:               vk.StructureTypeGraphicsPipelineCreateInfo,
		StageCount:          2,
		PStages:             shaderStages,
		PVertexInputState:   &vertexInputState,
		PInputAssemblyState: &inputAssemblyState,
		PViewportState:      &viewportState,
		PRasterizationState: &rasterState,
		PMultisampleState:   &multisampleState,
		PColorBlendState:    &colorBlendState,
		PDynamicState:       &dynamicState,
		Layout:              gfxPipeline.layout,
		RenderPass:          renderPass,
	}
	if opts.DepthTestEnable {
		pipelineCreateInfo.PDepthStencilState = &vk.PipelineDepthStencilStateCreateInfo{
			SType:            vk.StructureTypePipelineDepthStencilStateCreateInfo,
			DepthTestEnable:  vk.True,
			DepthWriteEnable: vk.True,
			DepthCompareOp:   vk.CompareOpLessOrEqual,
			Back:             vk.StencilOpState{FailOp: vk.StencilOpKeep, PassOp: vk.StencilOpKeep, CompareOp: vk.CompareOpAlways},
			Front:            vk.StencilOpState{FailOp: vk.StencilOpKeep, PassOp: vk.StencilOpKeep, CompareOp: vk.CompareOpAlways},
		}
	}
	pipelineCreateInfos := []vk.GraphicsPipelineCreateInfo{pipelineCreateInfo}
	pipelines := make([]vk.Pipeline, 1)
	err = vk.Error(vk.CreateGraphicsPipelines(device,
		gfxPipeline.cache, 1, pipelineCreateInfos, nil, pipelines))
	if err != nil {
		err = fmt.Errorf("vk.CreateGraphicsPipelines failed with %s", err)
		return gfxPipeline, err
	}
	gfxPipeline.pipeline = pipelines[0]
	gfxPipeline.device = device
	return gfxPipeline, nil
}

func (gfx *VulkanGfxPipelineInfo) GetLayout() vk.PipelineLayout {
	return gfx.layout
}

func (gfx *VulkanGfxPipelineInfo) GetPipeline() vk.Pipeline {
	return gfx.pipeline
}

func (gfx *VulkanGfxPipelineInfo) Destroy() {
	if gfx == nil {
		return
	}
	vk.DestroyPipeline(gfx.device, gfx.pipeline, nil)
	vk.DestroyPipelineCache(gfx.device, gfx.cache, nil)
	vk.DestroyPipelineLayout(gfx.device, gfx.layout, nil)
}
