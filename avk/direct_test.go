package avk

import (
	"testing"

	vk "github.com/tomas-mraz/vulkan"
)

func TestValidateSubmitInfoPanicsOnCountMismatch(t *testing.T) {
	tests := []vk.SubmitInfo{
		{
			WaitSemaphoreCount: 1,
		},
		{
			WaitSemaphoreCount: 1,
			PWaitSemaphores:    make([]vk.Semaphore, 1),
		},
		{
			CommandBufferCount: 1,
		},
		{
			SignalSemaphoreCount: 1,
		},
	}

	for _, submit := range tests {
		t.Run("", func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected panic on SubmitInfo count mismatch")
				}
			}()
			validateSubmitInfo(&submit)
		})
	}
}

func TestValidatePresentInfoPanicsOnCountMismatch(t *testing.T) {
	tests := []vk.PresentInfo{
		{
			WaitSemaphoreCount: 1,
		},
		{
			SwapchainCount: 1,
		},
		{
			SwapchainCount: 1,
			PSwapchains:    make([]vk.Swapchain, 1),
		},
		{
			SwapchainCount: 1,
			PSwapchains:    make([]vk.Swapchain, 1),
			PImageIndices:  []uint32{0},
			PResults:       []vk.Result{vk.Success, vk.Success},
		},
	}

	for _, present := range tests {
		t.Run("", func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected panic on PresentInfo count mismatch")
				}
			}()
			validatePresentInfo(&present)
		})
	}
}

func TestCmdBindVertexBuffersPanicsOnCountMismatch(t *testing.T) {
	arena := NewArenaWithChunkSize(128)
	defer arena.Free()

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on binding count mismatch")
		}
	}()

	CmdBindVertexBuffers(arena, nil, 0, 1, make([]vk.Buffer, 1), nil)
}
