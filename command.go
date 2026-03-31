package ash

import (
	"fmt"

	vk "github.com/tomas-mraz/vulkan"
)

// CommandContext owns a command pool plus reusable frame command buffers.
// It also provides helpers for transient single-use command buffers.
type CommandContext struct {
	device     vk.Device
	cmdPool    vk.CommandPool
	cmdBuffers []vk.CommandBuffer
}

// NewCommandContext creates a resettable command pool and optionally allocates
// a fixed set of primary command buffers for per-frame recording.
func NewCommandContext(device vk.Device, queueFamilyIndex, commandBufferCount uint32) (CommandContext, error) {
	var ctx CommandContext
	ctx.device = device

	err := vk.Error(vk.CreateCommandPool(device, &vk.CommandPoolCreateInfo{
		SType:            vk.StructureTypeCommandPoolCreateInfo,
		Flags:            vk.CommandPoolCreateFlags(vk.CommandPoolCreateResetCommandBufferBit),
		QueueFamilyIndex: queueFamilyIndex,
	}, nil, &ctx.cmdPool))
	if err != nil {
		return ctx, fmt.Errorf("vk.CreateCommandPool failed with %s", err)
	}

	if commandBufferCount == 0 {
		return ctx, nil
	}

	ctx.cmdBuffers = make([]vk.CommandBuffer, commandBufferCount)
	err = vk.Error(vk.AllocateCommandBuffers(device, &vk.CommandBufferAllocateInfo{
		SType:              vk.StructureTypeCommandBufferAllocateInfo,
		CommandPool:        ctx.cmdPool,
		Level:              vk.CommandBufferLevelPrimary,
		CommandBufferCount: commandBufferCount,
	}, ctx.cmdBuffers))
	if err != nil {
		vk.DestroyCommandPool(device, ctx.cmdPool, nil)
		ctx.cmdPool = vk.NullCommandPool
		return ctx, fmt.Errorf("vk.AllocateCommandBuffers failed with %s", err)
	}

	return ctx, nil
}

func (c *CommandContext) GetCmdPool() vk.CommandPool {
	return c.cmdPool
}

func (c *CommandContext) GetCmdBuffers() []vk.CommandBuffer {
	return c.cmdBuffers
}

// BeginOneTime allocates and begins a transient primary command buffer.
func (c *CommandContext) BeginOneTime() (vk.CommandBuffer, error) {
	cmds := make([]vk.CommandBuffer, 1)
	err := vk.Error(vk.AllocateCommandBuffers(c.device, &vk.CommandBufferAllocateInfo{
		SType:              vk.StructureTypeCommandBufferAllocateInfo,
		CommandPool:        c.cmdPool,
		Level:              vk.CommandBufferLevelPrimary,
		CommandBufferCount: 1,
	}, cmds))
	if err != nil {
		var zero vk.CommandBuffer
		return zero, fmt.Errorf("vk.AllocateCommandBuffers failed with %s", err)
	}

	cmd := cmds[0]
	err = vk.Error(vk.BeginCommandBuffer(cmd, &vk.CommandBufferBeginInfo{
		SType: vk.StructureTypeCommandBufferBeginInfo,
		Flags: vk.CommandBufferUsageFlags(vk.CommandBufferUsageOneTimeSubmitBit),
	}))
	if err != nil {
		vk.FreeCommandBuffers(c.device, c.cmdPool, 1, cmds)
		var zero vk.CommandBuffer
		return zero, fmt.Errorf("vk.BeginCommandBuffer failed with %s", err)
	}

	return cmd, nil
}

// EndOneTime ends, submits, waits, and frees a transient command buffer.
func (c *CommandContext) EndOneTime(queue vk.Queue, cmd vk.CommandBuffer) error {
	err := vk.Error(vk.EndCommandBuffer(cmd))
	if err != nil {
		vk.FreeCommandBuffers(c.device, c.cmdPool, 1, []vk.CommandBuffer{cmd})
		return fmt.Errorf("vk.EndCommandBuffer failed with %s", err)
	}

	var fence vk.Fence
	err = vk.Error(vk.CreateFence(c.device, &vk.FenceCreateInfo{
		SType: vk.StructureTypeFenceCreateInfo,
	}, nil, &fence))
	if err != nil {
		vk.FreeCommandBuffers(c.device, c.cmdPool, 1, []vk.CommandBuffer{cmd})
		return fmt.Errorf("vk.CreateFence failed with %s", err)
	}
	defer vk.DestroyFence(c.device, fence, nil)

	err = vk.Error(vk.QueueSubmit(queue, 1, []vk.SubmitInfo{{
		SType:              vk.StructureTypeSubmitInfo,
		CommandBufferCount: 1,
		PCommandBuffers:    []vk.CommandBuffer{cmd},
	}}, fence))
	if err != nil {
		vk.FreeCommandBuffers(c.device, c.cmdPool, 1, []vk.CommandBuffer{cmd})
		return fmt.Errorf("vk.QueueSubmit failed with %s", err)
	}

	err = vk.Error(vk.WaitForFences(c.device, 1, []vk.Fence{fence}, vk.True, 10_000_000_000))
	vk.FreeCommandBuffers(c.device, c.cmdPool, 1, []vk.CommandBuffer{cmd})
	if err != nil {
		return fmt.Errorf("vk.WaitForFences failed with %s", err)
	}

	return nil
}

// Destroy frees reusable command buffers and destroys the command pool.
func (c *CommandContext) Destroy() {
	if c == nil {
		return
	}
	if len(c.cmdBuffers) > 0 {
		vk.FreeCommandBuffers(c.device, c.cmdPool, uint32(len(c.cmdBuffers)), c.cmdBuffers)
		c.cmdBuffers = nil
	}
	if c.cmdPool != vk.NullCommandPool {
		vk.DestroyCommandPool(c.device, c.cmdPool, nil)
		c.cmdPool = vk.NullCommandPool
	}
}
