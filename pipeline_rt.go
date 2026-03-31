package ash

import (
	"fmt"

	vk "github.com/tomas-mraz/vulkan"
)

// RTShaderGroup defines one shader group for a ray tracing pipeline.
// The group type is inferred from which shader fields are set.
// Set exactly one of RaygenShader/MissShader/CallableShader for General groups,
// or ClosestHitShader (with optional AnyHitShader) for triangle hit groups,
// or IntersectionShader (with optional ClosestHitShader/AnyHitShader) for procedural hit groups.
type RTShaderGroup struct {
	RaygenShader       []byte // SPIR-V — implies General group, ShaderStageRaygenBit
	MissShader         []byte // SPIR-V — implies General group, ShaderStageMissBit
	CallableShader     []byte // SPIR-V — implies General group, ShaderStageCallableBit
	ClosestHitShader   []byte // SPIR-V — ShaderStageClosestHitBit
	AnyHitShader       []byte // SPIR-V — ShaderStageAnyHitBit
	IntersectionShader []byte // SPIR-V — implies Procedural hit group, ShaderStageIntersectionBit
}

// RTPipelineOptions configures a ray tracing pipeline.
type RTPipelineOptions struct {
	Groups                       []RTShaderGroup
	DescriptorSetLayouts         []vk.DescriptorSetLayout
	PushConstantRanges           []vk.PushConstantRange
	MaxPipelineRayRecursionDepth uint32 // 0 defaults to 1
}

// PipelineRtInfo holds a ray tracing pipeline and its layout.
type PipelineRtInfo struct {
	device   vk.Device
	layout   vk.PipelineLayout
	pipeline vk.Pipeline
}

func (p *PipelineRtInfo) GetLayout() vk.PipelineLayout { return p.layout }
func (p *PipelineRtInfo) GetPipeline() vk.Pipeline     { return p.pipeline }

func (p *PipelineRtInfo) Destroy() {
	if p == nil {
		return
	}
	vk.DestroyPipeline(p.device, p.pipeline, nil)
	vk.DestroyPipelineLayout(p.device, p.layout, nil)
}

// NewRTPipeline creates a ray tracing pipeline from shader group definitions.
// Shader modules are created internally and destroyed after pipeline creation.
func NewRTPipeline(device vk.Device, opts RTPipelineOptions) (PipelineRtInfo, error) {
	var p PipelineRtInfo
	p.device = device

	if len(opts.Groups) == 0 {
		return p, fmt.Errorf("RTPipelineOptions.Groups must not be empty")
	}

	maxRecursion := opts.MaxPipelineRayRecursionDepth
	if maxRecursion == 0 {
		maxRecursion = 1
	}

	// Pipeline layout
	if err := vk.Error(vk.CreatePipelineLayout(device, &vk.PipelineLayoutCreateInfo{
		SType:                  vk.StructureTypePipelineLayoutCreateInfo,
		SetLayoutCount:         uint32(len(opts.DescriptorSetLayouts)),
		PSetLayouts:            opts.DescriptorSetLayouts,
		PushConstantRangeCount: uint32(len(opts.PushConstantRanges)),
		PPushConstantRanges:    opts.PushConstantRanges,
	}, nil, &p.layout)); err != nil {
		return p, fmt.Errorf("vk.CreatePipelineLayout failed with %s", err)
	}

	var stages []vk.PipelineShaderStageCreateInfo
	var groups []vk.RayTracingShaderGroupCreateInfo
	var modules []vk.ShaderModule

	addStage := func(data []byte, stageBit vk.ShaderStageFlagBits) (uint32, error) {
		module, err := LoadShaderFromBytes(device, data)
		if err != nil {
			return 0, err
		}
		modules = append(modules, module)
		idx := uint32(len(stages))
		stages = append(stages, vk.PipelineShaderStageCreateInfo{
			SType:  vk.StructureTypePipelineShaderStageCreateInfo,
			Stage:  stageBit,
			Module: module,
			PName:  []byte("main\x00"),
		})
		return idx, nil
	}

	cleanup := func() {
		for _, m := range modules {
			vk.DestroyShaderModule(device, m, nil)
		}
		vk.DestroyPipelineLayout(device, p.layout, nil)
	}

	for i, g := range opts.Groups {
		group := vk.RayTracingShaderGroupCreateInfo{
			SType:              vk.StructureTypeRayTracingShaderGroupCreateInfo,
			GeneralShader:      vk.ShaderUnused,
			ClosestHitShader:   vk.ShaderUnused,
			AnyHitShader:       vk.ShaderUnused,
			IntersectionShader: vk.ShaderUnused,
		}

		// Determine group type and add general shader stage
		hasGeneral := g.RaygenShader != nil || g.MissShader != nil || g.CallableShader != nil

		if hasGeneral {
			group.Type = vk.RayTracingShaderGroupTypeGeneral
			var data []byte
			var stageBit vk.ShaderStageFlagBits
			switch {
			case g.RaygenShader != nil:
				data = g.RaygenShader
				stageBit = vk.ShaderStageFlagBits(vk.ShaderStageRaygenBit)
			case g.MissShader != nil:
				data = g.MissShader
				stageBit = vk.ShaderStageFlagBits(vk.ShaderStageMissBit)
			case g.CallableShader != nil:
				data = g.CallableShader
				stageBit = vk.ShaderStageFlagBits(vk.ShaderStageCallableBit)
			}
			idx, err := addStage(data, stageBit)
			if err != nil {
				cleanup()
				return p, fmt.Errorf("group %d general shader: %w", i, err)
			}
			group.GeneralShader = idx
		} else if g.IntersectionShader != nil {
			group.Type = vk.RayTracingShaderGroupTypeProceduralHitGroup
		} else {
			group.Type = vk.RayTracingShaderGroupTypeTrianglesHitGroup
		}

		if g.ClosestHitShader != nil {
			idx, err := addStage(g.ClosestHitShader, vk.ShaderStageFlagBits(vk.ShaderStageClosestHitBit))
			if err != nil {
				cleanup()
				return p, fmt.Errorf("group %d closest hit shader: %w", i, err)
			}
			group.ClosestHitShader = idx
		}

		if g.AnyHitShader != nil {
			idx, err := addStage(g.AnyHitShader, vk.ShaderStageFlagBits(vk.ShaderStageAnyHitBit))
			if err != nil {
				cleanup()
				return p, fmt.Errorf("group %d any hit shader: %w", i, err)
			}
			group.AnyHitShader = idx
		}

		if g.IntersectionShader != nil {
			idx, err := addStage(g.IntersectionShader, vk.ShaderStageFlagBits(vk.ShaderStageIntersectionBit))
			if err != nil {
				cleanup()
				return p, fmt.Errorf("group %d intersection shader: %w", i, err)
			}
			group.IntersectionShader = idx
		}

		groups = append(groups, group)
	}

	// Create pipeline
	createInfo := vk.RayTracingPipelineCreateInfo{
		SType:                        vk.StructureTypeRayTracingPipelineCreateInfo,
		StageCount:                   uint32(len(stages)),
		PStages:                      stages,
		GroupCount:                   uint32(len(groups)),
		PGroups:                      groups,
		MaxPipelineRayRecursionDepth: maxRecursion,
		Layout:                       p.layout,
	}
	if err := vk.Error(vk.CreateRayTracingPipelines(device, vk.DeferredOperation(vk.NullHandle), vk.NullPipelineCache, 1, &createInfo, nil, &p.pipeline)); err != nil {
		cleanup()
		return p, fmt.Errorf("vk.CreateRayTracingPipelines failed with %s", err)
	}

	// Shader modules no longer needed after pipeline creation
	for _, m := range modules {
		vk.DestroyShaderModule(device, m, nil)
	}

	return p, nil
}
