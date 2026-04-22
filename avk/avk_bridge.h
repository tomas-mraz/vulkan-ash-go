#ifndef AVK_BRIDGE_H
#define AVK_BRIDGE_H

#include <stdint.h>

typedef void* VkQueue;
typedef void* VkCommandBuffer;

typedef uint64_t VkBuffer;
typedef uint64_t VkDeviceSize;
typedef uint64_t VkFence;
typedef uint64_t VkFramebuffer;
typedef uint64_t VkPipelineLayout;
typedef uint64_t VkRenderPass;
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

typedef union VkClearValue {
	uint32_t uint32_[4];
	float float32_[4];
} VkClearValue;

typedef struct VkRenderPassBeginInfo {
	VkStructureType sType;
	const void* pNext;
	VkRenderPass renderPass;
	VkFramebuffer framebuffer;
	VkRect2D renderArea;
	uint32_t clearValueCount;
	const VkClearValue* pClearValues;
} VkRenderPassBeginInfo;

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

VkResult callVkQueueSubmit(
	VkQueue queue,
	uint32_t submitCount,
	const VkSubmitInfo* pSubmits,
	VkFence fence);

VkResult callVkBeginCommandBuffer(
	VkCommandBuffer commandBuffer,
	const VkCommandBufferBeginInfo* pBeginInfo);

VkResult callVkResetCommandBuffer(
	VkCommandBuffer commandBuffer,
	VkCommandBufferResetFlags flags);

void callVkCmdBindVertexBuffers(
	VkCommandBuffer commandBuffer,
	uint32_t firstBinding,
	uint32_t bindingCount,
	const VkBuffer* pBuffers,
	const VkDeviceSize* pOffsets);

void callVkCmdDraw(
	VkCommandBuffer commandBuffer,
	uint32_t vertexCount,
	uint32_t instanceCount,
	uint32_t firstVertex,
	uint32_t firstInstance);

void callVkCmdPushConstants(
	VkCommandBuffer commandBuffer,
	VkPipelineLayout layout,
	VkShaderStageFlags stageFlags,
	uint32_t offset,
	uint32_t size,
	const void* pValues);

void callVkCmdBeginRenderPass(
	VkCommandBuffer commandBuffer,
	const VkRenderPassBeginInfo* pRenderPassBegin,
	VkSubpassContents contents);

VkResult callVkQueuePresentKHR(
	VkQueue queue,
	const VkPresentInfoKHR* pPresentInfo);

// NOTE: the callVk* functions declared above are *defined* in
// github.com/tomas-mraz/vulkan/vk_bridge.c and resolved at link time.
// That package already dynamically loads the Vulkan ICD; this header
// only re-declares the entry points we bridge, so avk can call them
// without going through the allocating c-for-go Go wrappers.

#endif
