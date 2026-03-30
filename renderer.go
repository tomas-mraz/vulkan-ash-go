package ash

import (
	"fmt"

	vk "github.com/tomas-mraz/vulkan"
)

type VulkanRenderInfo struct {
	device     vk.Device
	RenderPass vk.RenderPass
	cmdPool    vk.CommandPool
	cmdBuffers []vk.CommandBuffer
	semaphores []vk.Semaphore
	fences     []vk.Fence
}

// NewRenderer creates a renderer with a single color attachment in the render pass.
func NewRenderer(device vk.Device, displayFormat vk.Format) (VulkanRenderInfo, error) {
	colorRef := []vk.AttachmentReference{{
		Attachment: 0,
		Layout:     vk.ImageLayoutColorAttachmentOptimal,
	}}
	rpInfo := vk.RenderPassCreateInfo{
		SType:           vk.StructureTypeRenderPassCreateInfo,
		AttachmentCount: 1,
		PAttachments: []vk.AttachmentDescription{{
			Format:         displayFormat,
			Samples:        vk.SampleCount1Bit,
			LoadOp:         vk.AttachmentLoadOpClear,
			StoreOp:        vk.AttachmentStoreOpStore,
			StencilLoadOp:  vk.AttachmentLoadOpDontCare,
			StencilStoreOp: vk.AttachmentStoreOpDontCare,
			InitialLayout:  vk.ImageLayoutUndefined,
			FinalLayout:    vk.ImageLayoutPresentSrc,
		}},
		SubpassCount: 1,
		PSubpasses: []vk.SubpassDescription{{
			PipelineBindPoint:    vk.PipelineBindPointGraphics,
			ColorAttachmentCount: 1,
			PColorAttachments:    colorRef,
		}},
	}
	return createRenderer(device, rpInfo)
}

// NewRendererWithDepth creates a renderer with color + depth attachments in the render pass.
func NewRendererWithDepth(device vk.Device, displayFormat vk.Format, depthFormat vk.Format) (VulkanRenderInfo, error) {
	colorRef := []vk.AttachmentReference{{
		Attachment: 0,
		Layout:     vk.ImageLayoutColorAttachmentOptimal,
	}}
	rpInfo := vk.RenderPassCreateInfo{
		SType:           vk.StructureTypeRenderPassCreateInfo,
		AttachmentCount: 2,
		PAttachments: []vk.AttachmentDescription{
			{
				Format:         displayFormat,
				Samples:        vk.SampleCount1Bit,
				LoadOp:         vk.AttachmentLoadOpClear,
				StoreOp:        vk.AttachmentStoreOpStore,
				StencilLoadOp:  vk.AttachmentLoadOpDontCare,
				StencilStoreOp: vk.AttachmentStoreOpDontCare,
				InitialLayout:  vk.ImageLayoutUndefined,
				FinalLayout:    vk.ImageLayoutPresentSrc,
			},
			{
				Format:         depthFormat,
				Samples:        vk.SampleCount1Bit,
				LoadOp:         vk.AttachmentLoadOpClear,
				StoreOp:        vk.AttachmentStoreOpDontCare,
				StencilLoadOp:  vk.AttachmentLoadOpDontCare,
				StencilStoreOp: vk.AttachmentStoreOpDontCare,
				InitialLayout:  vk.ImageLayoutUndefined,
				FinalLayout:    vk.ImageLayoutDepthStencilAttachmentOptimal,
			},
		},
		SubpassCount: 1,
		PSubpasses: []vk.SubpassDescription{{
			PipelineBindPoint:    vk.PipelineBindPointGraphics,
			ColorAttachmentCount: 1,
			PColorAttachments:    colorRef,
			PDepthStencilAttachment: &vk.AttachmentReference{
				Attachment: 1,
				Layout:     vk.ImageLayoutDepthStencilAttachmentOptimal,
			},
		}},
	}
	return createRenderer(device, rpInfo)
}

// createRenderer creates a render pass from the given configuration and a command pool.
func createRenderer(device vk.Device, renderPassInfo vk.RenderPassCreateInfo) (VulkanRenderInfo, error) {
	var r VulkanRenderInfo
	err := vk.Error(vk.CreateRenderPass(device, &renderPassInfo, nil, &r.RenderPass))
	if err != nil {
		return r, fmt.Errorf("vk.CreateRenderPass failed with %s", err)
	}
	cmdPoolCreateInfo := vk.CommandPoolCreateInfo{
		SType:            vk.StructureTypeCommandPoolCreateInfo,
		Flags:            vk.CommandPoolCreateFlags(vk.CommandPoolCreateResetCommandBufferBit),
		QueueFamilyIndex: 0,
	}
	err = vk.Error(vk.CreateCommandPool(device, &cmdPoolCreateInfo, nil, &r.cmdPool))
	if err != nil {
		return r, fmt.Errorf("vk.CreateCommandPool failed with %s", err)
	}
	r.device = device
	return r, nil
}

func (r *VulkanRenderInfo) CreateCommandBuffers(n uint32) error {
	r.cmdBuffers = make([]vk.CommandBuffer, n)
	cmdBufferAllocateInfo := vk.CommandBufferAllocateInfo{
		SType:              vk.StructureTypeCommandBufferAllocateInfo,
		CommandPool:        r.cmdPool,
		Level:              vk.CommandBufferLevelPrimary,
		CommandBufferCount: n,
	}
	err := vk.Error(vk.AllocateCommandBuffers(r.device, &cmdBufferAllocateInfo, r.cmdBuffers))
	if err != nil {
		return fmt.Errorf("vk.AllocateCommandBuffers failed with %s", err)
	}
	return nil
}

func (r *VulkanRenderInfo) GetCmdPool() vk.CommandPool {
	return r.cmdPool
}

func (r *VulkanRenderInfo) GetCmdBuffers() []vk.CommandBuffer {
	return r.cmdBuffers
}

// Destroy frees command buffers, destroys the command pool, and destroys the render pass.
func (r *VulkanRenderInfo) Destroy() {
	if r == nil {
		return
	}
	if len(r.cmdBuffers) > 0 {
		vk.FreeCommandBuffers(r.device, r.cmdPool, uint32(len(r.cmdBuffers)), r.cmdBuffers)
		r.cmdBuffers = nil
	}
	vk.DestroyCommandPool(r.device, r.cmdPool, nil)
	vk.DestroyRenderPass(r.device, r.RenderPass, nil)
}

func (r *VulkanRenderInfo) DefaultFence() vk.Fence {
	return r.fences[0]
}

func (r *VulkanRenderInfo) DefaultSemaphore() vk.Semaphore {
	return r.semaphores[0]
}
