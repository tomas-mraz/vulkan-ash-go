package avk

/*
#include <stdlib.h>
*/
import "C"

import "unsafe"

const (
	defaultChunkSize = 4 << 10
	arenaAlignment   = 16
)

type arenaBlock struct {
	ptr    unsafe.Pointer
	size   uintptr
	offset uintptr
}

// Arena owns reusable C memory blocks for short-lived Vulkan call data.
// Call Reset to reuse the blocks, or Free to release them back to libc.
//
// An Arena is not safe for concurrent use. Create one Arena per render
// goroutine (typically one per frame-recording context) and Reset it at
// the start of each frame.
type Arena struct {
	blocks    []arenaBlock
	blockIdx  int
	chunkSize uintptr
}

func NewArena() *Arena {
	return &Arena{chunkSize: defaultChunkSize}
}

func NewArenaWithChunkSize(chunkSize uintptr) *Arena {
	if chunkSize == 0 {
		chunkSize = defaultChunkSize
	}
	return &Arena{chunkSize: alignUp(chunkSize, arenaAlignment)}
}

// Alloc returns aligned C memory owned by the arena.
func (a *Arena) Alloc(size uintptr) unsafe.Pointer {
	if size == 0 {
		return nil
	}
	if a == nil {
		panic("avk: nil arena")
	}
	if a.chunkSize == 0 {
		a.chunkSize = defaultChunkSize
	}

	size = alignUp(size, arenaAlignment)
	for {
		b := a.currentBlock(size)
		offset := alignUp(b.offset, arenaAlignment)
		if offset+size <= b.size {
			ptr := unsafe.Add(b.ptr, offset)
			b.offset = offset + size
			return ptr
		}
		a.blockIdx++
	}
}

// Reset keeps allocated blocks and rewinds the arena for reuse.
func (a *Arena) Reset() {
	if a == nil {
		return
	}
	for i := range a.blocks {
		a.blocks[i].offset = 0
	}
	a.blockIdx = 0
}

// Free releases every C block owned by the arena.
func (a *Arena) Free() {
	if a == nil {
		return
	}
	for i := range a.blocks {
		if a.blocks[i].ptr != nil {
			C.free(a.blocks[i].ptr)
		}
	}
	a.blocks = nil
	a.blockIdx = 0
}

func (a *Arena) currentBlock(minSize uintptr) *arenaBlock {
	for a.blockIdx < len(a.blocks) {
		b := &a.blocks[a.blockIdx]
		if b.ptr != nil && b.size >= minSize {
			return b
		}
		a.blockIdx++
	}

	blockSize := a.chunkSize
	if blockSize < minSize {
		blockSize = minSize
	}
	ptr := C.malloc(C.size_t(blockSize))
	if ptr == nil {
		panic("avk: arena allocation failed")
	}

	a.blocks = append(a.blocks, arenaBlock{
		ptr:  ptr,
		size: blockSize,
	})
	a.blockIdx = len(a.blocks) - 1
	return &a.blocks[a.blockIdx]
}

func alignUp(v, align uintptr) uintptr {
	if align == 0 {
		return v
	}
	mask := align - 1
	return (v + mask) &^ mask
}
