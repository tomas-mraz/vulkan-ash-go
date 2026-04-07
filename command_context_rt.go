package ash

import vk "github.com/tomas-mraz/vulkan"

// BindRTPipeline binds a ray tracing pipeline and a descriptor set to the command buffer.
func (c *CommandContext) BindRTPipeline(cmd vk.CommandBuffer, pipeline PipelineRaytracing, descriptorSet vk.DescriptorSet) {
	vk.CmdBindPipeline(cmd, vk.PipelineBindPointRayTracing, pipeline.GetPipeline())
	vk.CmdBindDescriptorSets(cmd, vk.PipelineBindPointRayTracing, pipeline.GetLayout(), 0, 1, []vk.DescriptorSet{descriptorSet}, 0, nil)
}

// TraceRays records a ray tracing dispatch using the shader binding table regions.
func (c *CommandContext) TraceRays(cmd vk.CommandBuffer, sbt *ShaderBindingTable, displaySize vk.Extent2D) {
	vk.CmdTraceRays(cmd, &sbt.Raygen, &sbt.Miss, &sbt.Hit, &sbt.Callable, displaySize.Width, displaySize.Height, 1)
}
