//go:build !android

package ash

import (
	"fmt"

	"github.com/go-gl/glfw/v3.3/glfw"
	vk "github.com/tomas-mraz/vulkan"
)

// desktopHost is a GLFW-backed Host for Linux / macOS / Windows.
//
// GLFW's event pump (PollEvents, WindowShouldClose) has a hard requirement to
// run on the OS main thread. Run-loop methods (Pump) are therefore expected to
// be called by the Session from the main goroutine after runtime.LockOSThread.
//
// Events() is a buffered channel; Pump drains GLFW state and pushes events
// into it when window-close is detected. The first HostEventSurfaceAvailable is
// enqueued by Start so the Session immediately has something to consume.
type desktopHost struct {
	width, height int
	title         string

	window *glfw.Window
	events chan HostEvent
	closed bool
}

// NewDesktopHost returns a Host that drives GLFW on the calling main thread.
func NewDesktopHost(width, height int, title string) Host {
	return &desktopHost{
		width:  width,
		height: height,
		title:  title,
		events: make(chan HostEvent, 4),
	}
}

// NewDesktopSurface is a GLFW helper for creating a Vulkan surface.
func NewDesktopSurface(instance vk.Instance, window *glfw.Window) (vk.Surface, error) {
	surfacePointer, err := window.CreateWindowSurface(instance, nil)
	if err != nil {
		return vk.NullSurface, err
	}
	return vk.SurfaceFromPointer(surfacePointer), nil
}

func (h *desktopHost) Start() error {
	if err := glfw.Init(); err != nil {
		return fmt.Errorf("glfw.Init: %w", err)
	}
	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	glfw.WindowHint(glfw.Resizable, glfw.False)

	w, err := glfw.CreateWindow(h.width, h.height, h.title, nil, nil)
	if err != nil {
		glfw.Terminate()
		return fmt.Errorf("glfw.CreateWindow: %w", err)
	}
	h.window = w

	// Iconify callback bridges GLFW's minimize state to the same Pause/Resume
	// signal the Android host emits on onPause/onResume. GLFW invokes the
	// callback from inside glfw.PollEvents (main goroutine), so the channel
	// send is safe. A non-blocking send avoids wedging the main thread if the
	// consumer is momentarily slow to drain — the Session drains each loop
	// iteration, so a 4-slot buffer handles realistic bursts.
	w.SetIconifyCallback(func(_ *glfw.Window, iconified bool) {
		kind := HostEventResume
		if iconified {
			kind = HostEventPause
		}
		select {
		case h.events <- HostEvent{Kind: kind}:
		default:
		}
	})

	// The window is immediately usable; surface creation happens lazily when
	// the Session calls CreateSurface. Signal readiness upfront so Run can
	// kick off Vulkan setup.
	h.events <- HostEvent{
		Kind:   HostEventSurfaceAvailable,
		Extent: h.currentExtentUnchecked(),
	}
	return nil
}

func (h *desktopHost) InitVulkan() error {
	vk.SetGetInstanceProcAddr(glfw.GetVulkanGetInstanceProcAddress())
	return vk.Init()
}

func (h *desktopHost) InstanceExtensions() []string {
	if h.window == nil {
		return nil
	}
	return h.window.GetRequiredInstanceExtensions()
}

func (h *desktopHost) CreateSurface(instance vk.Instance) (vk.Surface, error) {
	if h.window == nil {
		return vk.NullSurface, fmt.Errorf("desktop host: no window")
	}
	return NewDesktopSurface(instance, h.window)
}

func (h *desktopHost) CurrentExtent() (vk.Extent2D, bool) {
	if h.window == nil {
		return vk.Extent2D{}, false
	}
	return h.currentExtentUnchecked(), true
}

func (h *desktopHost) currentExtentUnchecked() vk.Extent2D {
	w, ht := h.window.GetFramebufferSize()
	return vk.Extent2D{Width: uint32(w), Height: uint32(ht)}
}

func (h *desktopHost) Events() <-chan HostEvent {
	return h.events
}

// Pump drives GLFW on the main thread and emits HostEventClose when the user
// closes the window. Safe to call after Shutdown (becomes a no-op).
func (h *desktopHost) Pump() {
	if h.window == nil || h.closed {
		return
	}
	glfw.PollEvents()
	if h.window.ShouldClose() {
		h.closed = true
		// Non-blocking send: if the channel is already full (bursty events),
		// the consumer will still see Close via a subsequent drain.
		select {
		case h.events <- HostEvent{Kind: HostEventClose}:
		default:
		}
	}
}

func (h *desktopHost) Shutdown() {
	if h.window != nil {
		h.window.Destroy()
		h.window = nil
	}
	glfw.Terminate()
}
