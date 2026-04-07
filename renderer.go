package ash

import (
	"fmt"

	vk "github.com/tomas-mraz/vulkan"
)

// RasterizationPass owns a render pass used by rasterization pipelines.
// Framebuffers remain owned by Display and command buffers by CommandContext.
type RasterizationPass struct {
	device     vk.Device
	RenderPass vk.RenderPass
}

// NewRasterPass creates a render pass with a single color attachment.
func NewRasterPass(device vk.Device, displayFormat vk.Format) (RasterizationPass, error) {
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
	return createRasterPass(device, rpInfo)
}

// NewRasterPassWithDepth creates a render pass with color + depth attachments.
func NewRasterPassWithDepth(device vk.Device, displayFormat vk.Format, depthFormat vk.Format) (RasterizationPass, error) {
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
	return createRasterPass(device, rpInfo)
}

func createRasterPass(device vk.Device, renderPassInfo vk.RenderPassCreateInfo) (RasterizationPass, error) {
	var r RasterizationPass
	err := vk.Error(vk.CreateRenderPass(device, &renderPassInfo, nil, &r.RenderPass))
	if err != nil {
		return r, fmt.Errorf("vk.CreateRenderPass failed with %s", err)
	}
	r.device = device
	return r, nil
}

func (r *RasterizationPass) GetRenderPass() vk.RenderPass {
	return r.RenderPass
}

func (r *RasterizationPass) Destroy() {
	if r == nil {
		return
	}
	if r.RenderPass != vk.NullRenderPass {
		vk.DestroyRenderPass(r.device, r.RenderPass, nil)
		r.RenderPass = vk.NullRenderPass
	}
}
