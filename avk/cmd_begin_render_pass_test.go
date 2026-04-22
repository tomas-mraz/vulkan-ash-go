package avk

import (
	"testing"
	"unsafe"

	vk "github.com/tomas-mraz/vulkan"
)

func TestArenaResetReusesMemory(t *testing.T) {
	arena := NewArenaWithChunkSize(128)
	defer arena.Free()

	first := arena.Alloc(32)
	second := arena.Alloc(32)
	if first == nil || second == nil {
		t.Fatal("expected arena allocations")
	}
	if first == second {
		t.Fatal("expected distinct allocations before reset")
	}

	arena.Reset()

	reused := arena.Alloc(32)
	if reused != first {
		t.Fatalf("expected reset to reuse first block, got %p want %p", reused, first)
	}
}

func TestArenaRenderPassBeginInfoCopiesFields(t *testing.T) {
	arena := NewArenaWithChunkSize(256)
	defer arena.Free()

	clearValues := []vk.ClearValue{
		{1, 2, 3, 4},
		{9, 8, 7, 6},
	}
	begin := vk.RenderPassBeginInfo{
		SType: vk.StructureTypeRenderPassBeginInfo,
		PNext: unsafe.Pointer(uintptr(0x1234)),
		RenderArea: vk.Rect2D{
			Offset: vk.Offset2D{X: -7, Y: 9},
			Extent: vk.Extent2D{Width: 640, Height: 480},
		},
		ClearValueCount: uint32(len(clearValues)),
		PClearValues:    clearValues,
	}

	ref := arenaRenderPassBeginInfo(arena, &begin)
	raw := (*renderPassBeginInfo)(unsafe.Pointer(ref.Ref()))

	if raw.sType != int32(begin.SType) {
		t.Fatalf("unexpected sType: got %d want %d", raw.sType, begin.SType)
	}
	if raw.pNext != begin.PNext {
		t.Fatalf("unexpected pNext: got %p want %p", raw.pNext, begin.PNext)
	}
	if raw.renderArea.offset.x != begin.RenderArea.Offset.X || raw.renderArea.offset.y != begin.RenderArea.Offset.Y {
		t.Fatalf("unexpected offset: got (%d,%d)", raw.renderArea.offset.x, raw.renderArea.offset.y)
	}
	if raw.renderArea.extent.width != begin.RenderArea.Extent.Width || raw.renderArea.extent.height != begin.RenderArea.Extent.Height {
		t.Fatalf("unexpected extent: got (%d,%d)", raw.renderArea.extent.width, raw.renderArea.extent.height)
	}
	if raw.clearValueCount != begin.ClearValueCount {
		t.Fatalf("unexpected clearValueCount: got %d want %d", raw.clearValueCount, begin.ClearValueCount)
	}
	if raw.pClearValues == nil {
		t.Fatal("expected copied clear values")
	}

	gotClearValues := unsafe.Slice((*vk.ClearValue)(raw.pClearValues), len(clearValues))
	if gotClearValues[0] != clearValues[0] || gotClearValues[1] != clearValues[1] {
		t.Fatalf("unexpected clear values: got %v want %v", gotClearValues, clearValues)
	}
	if unsafe.Pointer(&gotClearValues[0]) == unsafe.Pointer(&clearValues[0]) {
		t.Fatal("expected arena-owned copy of clear values")
	}
}

func TestArenaRenderPassBeginInfoPanicsOnClearValueCountMismatch(t *testing.T) {
	tests := []vk.RenderPassBeginInfo{
		{
			ClearValueCount: 1,
			PClearValues:    nil,
		},
		{
			ClearValueCount: 0,
			PClearValues: []vk.ClearValue{
				{1, 2, 3, 4},
			},
		},
	}

	for _, begin := range tests {
		t.Run("", func(t *testing.T) {
			arena := NewArenaWithChunkSize(128)
			defer arena.Free()

			defer func() {
				if recover() == nil {
					t.Fatal("expected panic on ClearValueCount mismatch")
				}
			}()

			_ = arenaRenderPassBeginInfo(arena, &begin)
		})
	}
}
