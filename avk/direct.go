package avk

/*
#include <stdint.h>

typedef void* VkQueue;
typedef void* VkCommandBuffer;
typedef void* VkPipelineLayout;
typedef void* VkRenderPass;
typedef void* VkFence;

typedef uint64_t VkBuffer;
typedef uint64_t VkDeviceSize;
typedef uint64_t VkFramebuffer;
typedef uint64_t VkSemaphore;
typedef uint64_t VkSwapchainKHR;

typedef int32_t VkResult;
typedef int32_t VkStructureType;
typedef int32_t VkSubpassContents;

typedef uint32_t VkBool32;
typedef uint32_t VkCommandBufferResetFlags;
typedef uint32_t VkCommandBufferUsageFlags;
typedef uint32_t VkPipelineStageFlags;
typedef uint32_t VkQueryControlFlags;
typedef uint32_t VkQueryPipelineStatisticFlags;
typedef uint32_t VkShaderStageFlags;

typedef struct VkOffset2D {
	int32_t x;
	int32_t y;
} VkOffset2D;

typedef struct VkExtent2D {
	uint32_t width;
	uint32_t height;
} VkExtent2D;

typedef struct VkRect2D {
	VkOffset2D offset;
	VkExtent2D extent;
} VkRect2D;

typedef struct VkCommandBufferInheritanceInfo {
	VkStructureType sType;
	const void* pNext;
	VkRenderPass renderPass;
	uint32_t subpass;
	VkFramebuffer framebuffer;
	VkBool32 occlusionQueryEnable;
	VkQueryControlFlags queryFlags;
	VkQueryPipelineStatisticFlags pipelineStatistics;
} VkCommandBufferInheritanceInfo;

typedef struct VkCommandBufferBeginInfo {
	VkStructureType sType;
	const void* pNext;
	VkCommandBufferUsageFlags flags;
	const VkCommandBufferInheritanceInfo* pInheritanceInfo;
} VkCommandBufferBeginInfo;

typedef struct VkSubmitInfo {
	VkStructureType sType;
	const void* pNext;
	uint32_t waitSemaphoreCount;
	const VkSemaphore* pWaitSemaphores;
	const VkPipelineStageFlags* pWaitDstStageMask;
	uint32_t commandBufferCount;
	const VkCommandBuffer* pCommandBuffers;
	uint32_t signalSemaphoreCount;
	const VkSemaphore* pSignalSemaphores;
} VkSubmitInfo;

typedef struct VkPresentInfoKHR {
	VkStructureType sType;
	const void* pNext;
	uint32_t waitSemaphoreCount;
	const VkSemaphore* pWaitSemaphores;
	uint32_t swapchainCount;
	const VkSwapchainKHR* pSwapchains;
	const uint32_t* pImageIndices;
	VkResult* pResults;
} VkPresentInfoKHR;

extern VkResult callVkQueueSubmit(
	VkQueue queue,
	uint32_t submitCount,
	const VkSubmitInfo* pSubmits,
	VkFence fence);

extern VkResult callVkBeginCommandBuffer(
	VkCommandBuffer commandBuffer,
	const VkCommandBufferBeginInfo* pBeginInfo);

extern VkResult callVkResetCommandBuffer(
	VkCommandBuffer commandBuffer,
	VkCommandBufferResetFlags flags);

extern void callVkCmdBindVertexBuffers(
	VkCommandBuffer commandBuffer,
	uint32_t firstBinding,
	uint32_t bindingCount,
	const VkBuffer* pBuffers,
	const VkDeviceSize* pOffsets);

extern void callVkCmdDraw(
	VkCommandBuffer commandBuffer,
	uint32_t vertexCount,
	uint32_t instanceCount,
	uint32_t firstVertex,
	uint32_t firstInstance);

extern void callVkCmdPushConstants(
	VkCommandBuffer commandBuffer,
	VkPipelineLayout layout,
	VkShaderStageFlags stageFlags,
	uint32_t offset,
	uint32_t size,
	const void* pValues);

extern VkResult callVkQueuePresentKHR(
	VkQueue queue,
	const VkPresentInfoKHR* pPresentInfo);
*/
import "C"

import (
	"fmt"
	"runtime"
	"unsafe"

	vk "github.com/tomas-mraz/vulkan"
)

func CmdPushConstants(commandBuffer vk.CommandBuffer, layout vk.PipelineLayout, stageFlags vk.ShaderStageFlags, offset uint32, size uint32, pValues unsafe.Pointer) {
	if size != 0 && pValues == nil {
		panic("avk: CmdPushConstants requires pValues when size > 0")
	}

	ccommandBuffer := *(*C.VkCommandBuffer)(unsafe.Pointer(&commandBuffer))
	clayout := *(*C.VkPipelineLayout)(unsafe.Pointer(&layout))
	C.callVkCmdPushConstants(
		ccommandBuffer,
		clayout,
		C.VkShaderStageFlags(stageFlags),
		C.uint32_t(offset),
		C.uint32_t(size),
		pValues,
	)
	runtime.KeepAlive(pValues)
}

func CmdBindVertexBuffers(arena *Arena, commandBuffer vk.CommandBuffer, firstBinding uint32, bindingCount uint32, pBuffers []vk.Buffer, pOffsets []vk.DeviceSize) {
	if arena == nil {
		panic("avk: nil arena")
	}
	if bindingCount != uint32(len(pBuffers)) {
		panic(fmt.Sprintf("avk: bindingCount (%d) must match len(pBuffers) (%d)", bindingCount, len(pBuffers)))
	}
	if bindingCount != uint32(len(pOffsets)) {
		panic(fmt.Sprintf("avk: bindingCount (%d) must match len(pOffsets) (%d)", bindingCount, len(pOffsets)))
	}

	ccommandBuffer := *(*C.VkCommandBuffer)(unsafe.Pointer(&commandBuffer))
	var cpBuffers *C.VkBuffer
	var cpOffsets *C.VkDeviceSize
	if bindingCount != 0 {
		cpBuffers = (*C.VkBuffer)(arenaCopySlice(arena, pBuffers))
		cpOffsets = (*C.VkDeviceSize)(arenaCopySlice(arena, pOffsets))
	}

	C.callVkCmdBindVertexBuffers(
		ccommandBuffer,
		C.uint32_t(firstBinding),
		C.uint32_t(bindingCount),
		cpBuffers,
		cpOffsets,
	)
	runtime.KeepAlive(arena)
}

func CmdDraw(commandBuffer vk.CommandBuffer, vertexCount uint32, instanceCount uint32, firstVertex uint32, firstInstance uint32) {
	ccommandBuffer := *(*C.VkCommandBuffer)(unsafe.Pointer(&commandBuffer))
	C.callVkCmdDraw(
		ccommandBuffer,
		C.uint32_t(vertexCount),
		C.uint32_t(instanceCount),
		C.uint32_t(firstVertex),
		C.uint32_t(firstInstance),
	)
}

func QueueSubmit(arena *Arena, queue vk.Queue, submitCount uint32, pSubmits []vk.SubmitInfo, fence vk.Fence) vk.Result {
	if arena == nil {
		panic("avk: nil arena")
	}
	if submitCount != uint32(len(pSubmits)) {
		panic(fmt.Sprintf("avk: submitCount (%d) must match len(pSubmits) (%d)", submitCount, len(pSubmits)))
	}

	cqueue := *(*C.VkQueue)(unsafe.Pointer(&queue))
	cfence := *(*C.VkFence)(unsafe.Pointer(&fence))
	cpSubmits := arenaSubmitInfos(arena, pSubmits)

	ret := vk.Result(C.callVkQueueSubmit(cqueue, C.uint32_t(submitCount), cpSubmits, cfence))
	runtime.KeepAlive(arena)
	runtime.KeepAlive(pSubmits)
	return ret
}

func QueuePresent(arena *Arena, queue vk.Queue, pPresentInfo *vk.PresentInfo) vk.Result {
	cqueue := *(*C.VkQueue)(unsafe.Pointer(&queue))
	cpPresentInfo, cResults := arenaPresentInfo(arena, pPresentInfo)

	ret := vk.Result(C.callVkQueuePresentKHR(cqueue, cpPresentInfo))
	if len(pPresentInfo.PResults) != 0 {
		copy(pPresentInfo.PResults, unsafe.Slice((*vk.Result)(unsafe.Pointer(cResults)), len(pPresentInfo.PResults)))
	}
	runtime.KeepAlive(arena)
	runtime.KeepAlive(pPresentInfo)
	return ret
}

func BeginCommandBuffer(arena *Arena, commandBuffer vk.CommandBuffer, pBeginInfo *vk.CommandBufferBeginInfo) vk.Result {
	ccommandBuffer := *(*C.VkCommandBuffer)(unsafe.Pointer(&commandBuffer))
	cpBeginInfo := arenaCommandBufferBeginInfo(arena, pBeginInfo)

	ret := vk.Result(C.callVkBeginCommandBuffer(ccommandBuffer, cpBeginInfo))
	runtime.KeepAlive(arena)
	runtime.KeepAlive(pBeginInfo)
	return ret
}

func ResetCommandBuffer(commandBuffer vk.CommandBuffer, flags vk.CommandBufferResetFlags) vk.Result {
	ccommandBuffer := *(*C.VkCommandBuffer)(unsafe.Pointer(&commandBuffer))
	ret := C.callVkResetCommandBuffer(ccommandBuffer, C.VkCommandBufferResetFlags(flags))
	return vk.Result(ret)
}

func arenaCommandBufferBeginInfo(arena *Arena, begin *vk.CommandBufferBeginInfo) *C.VkCommandBufferBeginInfo {
	if begin == nil {
		return nil
	}
	if arena == nil {
		panic("avk: nil arena")
	}
	if len(begin.PInheritanceInfo) > 1 {
		panic(fmt.Sprintf("avk: len(PInheritanceInfo) must be 0 or 1, got %d", len(begin.PInheritanceInfo)))
	}

	ref := (*C.VkCommandBufferBeginInfo)(arena.Alloc(unsafe.Sizeof(C.VkCommandBufferBeginInfo{})))
	ref.sType = C.VkStructureType(begin.SType)
	ref.pNext = begin.PNext
	ref.flags = C.VkCommandBufferUsageFlags(begin.Flags)
	if len(begin.PInheritanceInfo) == 1 {
		ref.pInheritanceInfo = arenaCommandBufferInheritanceInfo(arena, &begin.PInheritanceInfo[0])
	}
	return ref
}

func arenaCommandBufferInheritanceInfo(arena *Arena, info *vk.CommandBufferInheritanceInfo) *C.VkCommandBufferInheritanceInfo {
	ref := (*C.VkCommandBufferInheritanceInfo)(arena.Alloc(unsafe.Sizeof(C.VkCommandBufferInheritanceInfo{})))
	ref.sType = C.VkStructureType(info.SType)
	ref.pNext = info.PNext
	ref.renderPass = *(*C.VkRenderPass)(unsafe.Pointer(&info.RenderPass))
	ref.subpass = C.uint32_t(info.Subpass)
	ref.framebuffer = *(*C.VkFramebuffer)(unsafe.Pointer(&info.Framebuffer))
	ref.occlusionQueryEnable = C.VkBool32(info.OcclusionQueryEnable)
	ref.queryFlags = C.VkQueryControlFlags(info.QueryFlags)
	ref.pipelineStatistics = C.VkQueryPipelineStatisticFlags(info.PipelineStatistics)
	return ref
}

func arenaSubmitInfos(arena *Arena, submits []vk.SubmitInfo) *C.VkSubmitInfo {
	if len(submits) == 0 {
		return nil
	}

	ref := unsafe.Slice((*C.VkSubmitInfo)(arena.Alloc(uintptr(len(submits))*unsafe.Sizeof(C.VkSubmitInfo{}))), len(submits))
	for i := range submits {
		submit := &submits[i]
		validateSubmitInfo(submit)

		ref[i].sType = C.VkStructureType(submit.SType)
		ref[i].pNext = submit.PNext
		ref[i].waitSemaphoreCount = C.uint32_t(submit.WaitSemaphoreCount)
		ref[i].commandBufferCount = C.uint32_t(submit.CommandBufferCount)
		ref[i].signalSemaphoreCount = C.uint32_t(submit.SignalSemaphoreCount)

		if submit.WaitSemaphoreCount != 0 {
			ref[i].pWaitSemaphores = (*C.VkSemaphore)(arenaCopySlice(arena, submit.PWaitSemaphores))
			ref[i].pWaitDstStageMask = (*C.VkPipelineStageFlags)(arenaCopySlice(arena, submit.PWaitDstStageMask))
		}
		if submit.CommandBufferCount != 0 {
			ref[i].pCommandBuffers = (*C.VkCommandBuffer)(arenaCopySlice(arena, submit.PCommandBuffers))
		}
		if submit.SignalSemaphoreCount != 0 {
			ref[i].pSignalSemaphores = (*C.VkSemaphore)(arenaCopySlice(arena, submit.PSignalSemaphores))
		}
	}
	return &ref[0]
}

func arenaPresentInfo(arena *Arena, info *vk.PresentInfo) (*C.VkPresentInfoKHR, *C.VkResult) {
	if info == nil {
		return nil, nil
	}
	if arena == nil {
		panic("avk: nil arena")
	}
	validatePresentInfo(info)

	ref := (*C.VkPresentInfoKHR)(arena.Alloc(unsafe.Sizeof(C.VkPresentInfoKHR{})))
	ref.sType = C.VkStructureType(info.SType)
	ref.pNext = info.PNext
	ref.waitSemaphoreCount = C.uint32_t(info.WaitSemaphoreCount)
	ref.swapchainCount = C.uint32_t(info.SwapchainCount)

	if info.WaitSemaphoreCount != 0 {
		ref.pWaitSemaphores = (*C.VkSemaphore)(arenaCopySlice(arena, info.PWaitSemaphores))
	}
	if info.SwapchainCount != 0 {
		ref.pSwapchains = (*C.VkSwapchainKHR)(arenaCopySlice(arena, info.PSwapchains))
		ref.pImageIndices = (*C.uint32_t)(arenaCopySlice(arena, info.PImageIndices))
	}

	var cResults *C.VkResult
	if len(info.PResults) != 0 {
		cResults = (*C.VkResult)(arenaCopySlice(arena, info.PResults))
		ref.pResults = cResults
	}
	return ref, cResults
}

func validateSubmitInfo(submit *vk.SubmitInfo) {
	if submit.WaitSemaphoreCount != uint32(len(submit.PWaitSemaphores)) {
		panic(fmt.Sprintf(
			"avk: WaitSemaphoreCount (%d) must match len(PWaitSemaphores) (%d)",
			submit.WaitSemaphoreCount,
			len(submit.PWaitSemaphores),
		))
	}
	if submit.WaitSemaphoreCount != uint32(len(submit.PWaitDstStageMask)) {
		panic(fmt.Sprintf(
			"avk: WaitSemaphoreCount (%d) must match len(PWaitDstStageMask) (%d)",
			submit.WaitSemaphoreCount,
			len(submit.PWaitDstStageMask),
		))
	}
	if submit.CommandBufferCount != uint32(len(submit.PCommandBuffers)) {
		panic(fmt.Sprintf(
			"avk: CommandBufferCount (%d) must match len(PCommandBuffers) (%d)",
			submit.CommandBufferCount,
			len(submit.PCommandBuffers),
		))
	}
	if submit.SignalSemaphoreCount != uint32(len(submit.PSignalSemaphores)) {
		panic(fmt.Sprintf(
			"avk: SignalSemaphoreCount (%d) must match len(PSignalSemaphores) (%d)",
			submit.SignalSemaphoreCount,
			len(submit.PSignalSemaphores),
		))
	}
}

func validatePresentInfo(info *vk.PresentInfo) {
	if info.WaitSemaphoreCount != uint32(len(info.PWaitSemaphores)) {
		panic(fmt.Sprintf(
			"avk: WaitSemaphoreCount (%d) must match len(PWaitSemaphores) (%d)",
			info.WaitSemaphoreCount,
			len(info.PWaitSemaphores),
		))
	}
	if info.SwapchainCount != uint32(len(info.PSwapchains)) {
		panic(fmt.Sprintf(
			"avk: SwapchainCount (%d) must match len(PSwapchains) (%d)",
			info.SwapchainCount,
			len(info.PSwapchains),
		))
	}
	if info.SwapchainCount != uint32(len(info.PImageIndices)) {
		panic(fmt.Sprintf(
			"avk: SwapchainCount (%d) must match len(PImageIndices) (%d)",
			info.SwapchainCount,
			len(info.PImageIndices),
		))
	}
	if len(info.PResults) != 0 && info.SwapchainCount != uint32(len(info.PResults)) {
		panic(fmt.Sprintf(
			"avk: len(PResults) must be 0 or match SwapchainCount (%d), got %d",
			info.SwapchainCount,
			len(info.PResults),
		))
	}
}

func arenaCopySlice[T any](arena *Arena, src []T) unsafe.Pointer {
	if len(src) == 0 {
		return nil
	}
	var zero T
	size := uintptr(len(src)) * unsafe.Sizeof(zero)
	dst := arena.Alloc(size)
	copy(unsafe.Slice((*T)(dst), len(src)), src)
	return dst
}
