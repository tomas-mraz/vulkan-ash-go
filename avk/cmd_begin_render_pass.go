package avk

import (
	"fmt"
	"runtime"
	"unsafe"

	vk "github.com/tomas-mraz/vulkan"
)

type offset2D struct {
	x int32
	y int32
}

type extent2D struct {
	width  uint32
	height uint32
}

type rect2D struct {
	offset offset2D
	extent extent2D
}

type renderPassBeginInfo struct {
	sType           int32
	pNext           unsafe.Pointer
	renderPass      vk.RenderPass
	framebuffer     vk.Framebuffer
	renderArea      rect2D
	clearValueCount uint32
	pClearValues    unsafe.Pointer
}

// CmdBeginRenderPass builds VkRenderPassBeginInfo in arena-owned C memory and
// reuses that storage across calls after Arena.Reset.
func CmdBeginRenderPass(arena *Arena, commandBuffer vk.CommandBuffer, begin *vk.RenderPassBeginInfo, contents vk.SubpassContents) {
	ref := arenaRenderPassBeginInfo(arena, begin)
	callCmdBeginRenderPass(commandBuffer, unsafe.Pointer(ref), contents)
	runtime.KeepAlive(arena)
	runtime.KeepAlive(begin)
}

func arenaRenderPassBeginInfo(arena *Arena, begin *vk.RenderPassBeginInfo) *renderPassBeginInfo {
	if begin == nil {
		return nil
	}
	if arena == nil {
		panic("avk: nil arena")
	}
	validateClearValueCount(begin)

	ref := (*renderPassBeginInfo)(arena.Alloc(unsafe.Sizeof(renderPassBeginInfo{})))
	*ref = renderPassBeginInfo{
		sType:       int32(begin.SType),
		pNext:       begin.PNext,
		renderPass:  begin.RenderPass,
		framebuffer: begin.Framebuffer,
		renderArea: rect2D{
			offset: offset2D{
				x: begin.RenderArea.Offset.X,
				y: begin.RenderArea.Offset.Y,
			},
			extent: extent2D{
				width:  begin.RenderArea.Extent.Width,
				height: begin.RenderArea.Extent.Height,
			},
		},
		clearValueCount: begin.ClearValueCount,
	}

	if len(begin.PClearValues) != 0 {
		size := uintptr(len(begin.PClearValues)) * unsafe.Sizeof(vk.ClearValue{})
		ref.pClearValues = arena.Alloc(size)
		copy(unsafe.Slice((*vk.ClearValue)(ref.pClearValues), len(begin.PClearValues)), begin.PClearValues)
	}

	return ref
}

func validateClearValueCount(begin *vk.RenderPassBeginInfo) {
	want := uint32(len(begin.PClearValues))
	if begin.ClearValueCount != want {
		panic(fmt.Sprintf(
			"avk: ClearValueCount (%d) must match len(PClearValues) (%d)",
			begin.ClearValueCount,
			want,
		))
	}
}
