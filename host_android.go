//go:build android

package ash

import (
	"fmt"
	"sync"

	vk "github.com/tomas-mraz/vulkan"

	"github.com/tomas-mraz/android-go/android"
	"github.com/tomas-mraz/android-go/app"
)

// androidHost wraps an app.NativeActivity and fans events from the native
// callback channels (NativeWindow*, Lifecycle, InputQueue) into a single
// HostEvent channel consumed by the Session.
//
// The demux goroutine is started from Start and runs until OnDestroy is
// observed; it is responsible for:
//   - tracking the current native window pointer used by CreateSurface
//   - calling NativeWindowRedrawDone after emitting SurfaceInvalidated
//   - closing the events channel after HostEventClose
type androidHost struct {
	activity app.NativeActivity

	events chan HostEvent

	nativeWindowEvents chan app.NativeWindowEvent
	inputQueueEvents   chan app.InputQueueEvent
	inputQueueChan     chan *android.InputQueue

	windowMu sync.Mutex
	window   *android.NativeWindow

	started  bool
	shutdown bool
}

// NewAndroidHost returns a Host that observes lifecycle and surface events
// from the given NativeActivity. Must be called inside app.Main's callback.
func NewAndroidHost(a app.NativeActivity) Host {
	return &androidHost{
		activity:           a,
		events:             make(chan HostEvent, 8),
		nativeWindowEvents: make(chan app.NativeWindowEvent),
		inputQueueEvents:   make(chan app.InputQueueEvent, 1),
		inputQueueChan:     make(chan *android.InputQueue, 1),
	}
}

// NewAndroidSurface is an Android helper to get Vulkan surface.
func NewAndroidSurface(instance vk.Instance, windowPointer uintptr) (vk.Surface, error) {
	var surface vk.Surface
	err := vk.Error(vk.CreateWindowSurface(instance, windowPointer, nil, &surface))
	if err != nil {
		return vk.NullSurface, err
	}
	return surface, nil
}

func (h *androidHost) Start() error {
	if h.started {
		return nil
	}
	h.started = true

	h.activity.HandleNativeWindowEvents(h.nativeWindowEvents)
	h.activity.HandleInputQueueEvents(h.inputQueueEvents)
	go app.HandleInputQueues(h.inputQueueChan, func() {
		h.activity.InputQueueHandled()
	}, app.SkipInputEvents)
	h.activity.InitDone()

	go h.demux()
	return nil
}

// demux consumes native event channels and translates them into HostEvents.
// It runs until OnDestroy, after which it closes the events channel so that
// Session.Run terminates cleanly.
func (h *androidHost) demux() {
	defer close(h.events)
	for {
		select {
		case ev := <-h.activity.LifecycleEvents():
			switch ev.Kind {
			case app.OnDestroy:
				h.events <- HostEvent{Kind: HostEventClose}
				return
			case app.OnPause:
				// Activity is backgrounded but the surface is still valid.
				// Session quiesces GPU work until a matching OnResume.
				h.events <- HostEvent{Kind: HostEventPause}
			case app.OnResume:
				h.events <- HostEvent{Kind: HostEventResume}
			}

		case ev := <-h.inputQueueEvents:
			switch ev.Kind {
			case app.QueueCreated:
				h.inputQueueChan <- ev.Queue
			case app.QueueDestroyed:
				h.inputQueueChan <- nil
			}

		case ev := <-h.nativeWindowEvents:
			switch ev.Kind {
			case app.NativeWindowCreated:
				h.setWindow(ev.Window)
				h.events <- HostEvent{
					Kind:   HostEventSurfaceAvailable,
					Extent: windowExtent(ev.Window),
				}

			case app.NativeWindowDestroyed:
				h.events <- HostEvent{Kind: HostEventSurfaceLost}
				h.setWindow(nil)

			case app.NativeWindowRedrawNeeded:
				// Rotation / resize: let the Session rebuild the swapchain at the
				// next frame boundary. Ack the platform immediately — delaying the
				// Ack until after the rebuild caused visible stalls in practice,
				// and the surface remains valid throughout the rebuild.
				h.events <- HostEvent{
					Kind:   HostEventSurfaceInvalidated,
					Extent: windowExtent(ev.Window),
				}
				h.activity.NativeWindowRedrawDone()
			}
		}
	}
}

func (h *androidHost) setWindow(w *android.NativeWindow) {
	h.windowMu.Lock()
	h.window = w
	h.windowMu.Unlock()
}

func (h *androidHost) getWindow() *android.NativeWindow {
	h.windowMu.Lock()
	defer h.windowMu.Unlock()
	return h.window
}

func (h *androidHost) InitVulkan() error {
	if err := vk.SetDefaultGetInstanceProcAddr(); err != nil {
		return fmt.Errorf("vk.SetDefaultGetInstanceProcAddr: %w", err)
	}
	return vk.Init()
}

// InstanceExtensions returns nothing extra: Manager already appends
// VK_KHR_android_surface via GetAndroidRequiredInstanceExtensions.
func (h *androidHost) InstanceExtensions() []string {
	return nil
}

func (h *androidHost) CreateSurface(instance vk.Instance) (vk.Surface, error) {
	w := h.getWindow()
	if w == nil {
		return vk.NullSurface, fmt.Errorf("android host: no native window")
	}
	return NewAndroidSurface(instance, w.Ptr())
}

func (h *androidHost) CurrentExtent() (vk.Extent2D, bool) {
	w := h.getWindow()
	if w == nil {
		return vk.Extent2D{}, false
	}
	return windowExtent(w), true
}

func (h *androidHost) Events() <-chan HostEvent {
	return h.events
}

// Pump is a no-op on Android: events are fed by the demux goroutine.
func (h *androidHost) Pump() {}

func (h *androidHost) Shutdown() {
	h.shutdown = true
}

func windowExtent(w *android.NativeWindow) vk.Extent2D {
	if w == nil {
		return vk.Extent2D{}
	}
	return vk.Extent2D{
		Width:  uint32(android.NativeWindowGetWidth(w)),
		Height: uint32(android.NativeWindowGetHeight(w)),
	}
}
