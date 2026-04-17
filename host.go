package ash

import (
	vk "github.com/tomas-mraz/vulkan"
)

// HostEventKind enumerates platform lifecycle events the Session reacts to.
type HostEventKind int

const (
	// HostEventSurfaceAvailable signals that a platform surface is ready for use.
	// Extent carries the initial window size. Emitted on first startup and again
	// after a previously-lost surface has been recreated.
	HostEventSurfaceAvailable HostEventKind = iota

	// HostEventSurfaceLost signals that the platform surface has been destroyed.
	// Any Vulkan swapchain or device tied to it must be torn down before the next
	// HostEventSurfaceAvailable can be honored.
	HostEventSurfaceLost

	// HostEventSurfaceInvalidated signals that the current surface is still valid
	// but its size or orientation has changed. Extent carries the new size. The
	// Session responds by requesting a swapchain recreation at the next frame.
	HostEventSurfaceInvalidated

	// HostEventPause signals that the app has been backgrounded and must stop
	// issuing GPU work. The Vulkan device and swapchain remain valid — this is
	// a soft signal, orthogonal to surface lifetime. Android emits it on
	// onPause; the Session waits for the device to become idle and then gates
	// the render loop until HostEventResume arrives. If the platform later
	// destroys the surface while paused, HostEventSurfaceLost follows normally.
	HostEventPause

	// HostEventResume signals that the app has been foregrounded. The Session
	// un-gates the render loop. If the swapchain changed during the pause
	// (e.g. rotation while backgrounded), a HostEventSurfaceInvalidated is
	// emitted separately.
	HostEventResume

	// HostEventClose signals that the user or OS asked the app to exit.
	HostEventClose
)

// HostEvent is a single platform-originated message on the Host events channel.
type HostEvent struct {
	Kind   HostEventKind
	Extent vk.Extent2D
}

// Host abstracts the platform/windowing layer (GLFW on desktop, app.NativeActivity
// on Android). A Host is created by the application entry point and handed to
// NewSession, which drives the entire Vulkan lifecycle on top of it.
//
// Responsibilities split:
//   - The Host owns the platform window / native surface handle and emits
//     lifecycle events onto Events().
//   - The Session owns Vulkan instance/device/swapchain and consumes events.
//
// Thread model:
//   - Run-loop methods (Pump) must be called from the same goroutine that runs
//     the platform event pump. On desktop this is the OS main thread (locked
//     via runtime.LockOSThread). On Android it is the app.Main callback
//     goroutine, and Pump is a no-op.
//   - CreateSurface and InitVulkan may be called from the run-loop goroutine.
type Host interface {
	// Start prepares the platform layer and begins delivering events on Events().
	// It must be called before any other method except Events().
	Start() error

	// InitVulkan performs platform-specific proc-address loader setup and vk.Init.
	// Called by NewSession before instance creation.
	InitVulkan() error

	// InstanceExtensions reports platform-specific instance extensions required
	// to build a working surface. Manager may add further generic extensions.
	InstanceExtensions() []string

	// CreateSurface creates a VkSurfaceKHR for the current platform window.
	// Returns an error when no window is currently available.
	CreateSurface(instance vk.Instance) (vk.Surface, error)

	// CurrentExtent reports the platform window's logical size.
	// ok=false when no surface is known yet (e.g. before first SurfaceAvailable).
	CurrentExtent() (vk.Extent2D, bool)

	// Events returns a channel of platform lifecycle events. Consumers must
	// drain it promptly to avoid blocking the demux goroutine.
	Events() <-chan HostEvent

	// Pump advances the platform event loop on the run-loop goroutine. It is a
	// no-op on platforms whose event loop runs independently.
	Pump()

	// Shutdown releases platform resources. Safe to call multiple times.
	Shutdown()
}
