package ash

import (
	"log/slog"
	"unsafe"

	vk "github.com/tomas-mraz/vulkan"
)

// DisplayTiming provides frame pacing using the VK_GOOGLE_display_timing extension.
// It queries actual display refresh rate and past presentation timestamps to calculate
// optimal target present times, eliminating stutter and unnecessary frame drops.
//
// Usage:
//  1. Enable VK_GOOGLE_display_timing as a device extension.
//  2. After creating the swapchain, call NewDisplayTiming.
//  3. Before each QueuePresent, call NextPresentInfo and chain it via PresentInfo.PNext.
//  4. The extension gracefully degrades: when unsupported, IsEnabled returns false
//     and NextPresentInfo returns nil (present without timing).
type DisplayTiming struct {
	device    vk.Device
	swapchain vk.Swapchain
	enabled   bool

	refreshDuration uint64 // nanoseconds per display refresh cycle
	presentID       uint32 // monotonically increasing frame ID
	targetTime      uint64 // desired present time for the next frame (ns)
}

// NewDisplayTiming creates a DisplayTiming for the given swapchain.
// The VK_GOOGLE_display_timing extension must be enabled on the device.
// If the extension is not available or the query fails, the returned
// DisplayTiming is disabled and all methods become no-ops.
func NewDisplayTiming(device vk.Device, swapchain vk.Swapchain) DisplayTiming {
	var dt DisplayTiming
	dt.device = device
	dt.swapchain = swapchain

	var props vk.RefreshCycleDurationGOOGLE
	ret := vk.GetRefreshCycleDurationGOOGLE(device, swapchain, &props)
	if ret != vk.Success {
		slog.Debug("DisplayTiming: GetRefreshCycleDurationGOOGLE not available, frame pacing disabled")
		return dt
	}
	props.Deref()
	dt.refreshDuration = props.RefreshDuration
	dt.enabled = true
	slog.Debug("DisplayTiming: enabled", "refreshDuration_ns", dt.refreshDuration)
	return dt
}

// IsEnabled reports whether display timing is active.
func (dt *DisplayTiming) IsEnabled() bool {
	return dt != nil && dt.enabled
}

// Rebind updates the swapchain handle after a swapchain recreation.
// Past-presentation timing state is reset because it belongs to the old swapchain.
func (dt *DisplayTiming) Rebind(swapchain vk.Swapchain) {
	if dt == nil {
		return
	}
	dt.swapchain = swapchain
	dt.targetTime = 0
	dt.presentID = 0
	if !dt.enabled {
		return
	}
	var props vk.RefreshCycleDurationGOOGLE
	if ret := vk.GetRefreshCycleDurationGOOGLE(dt.device, swapchain, &props); ret == vk.Success {
		props.Deref()
		dt.refreshDuration = props.RefreshDuration
	}
}

// GetRefreshDuration returns the display refresh cycle in nanoseconds.
// Returns 0 when display timing is disabled.
func (dt *DisplayTiming) GetRefreshDuration() uint64 {
	if !dt.IsEnabled() {
		return 0
	}
	return dt.refreshDuration
}

// SyncPastTiming drains all pending past presentation results and uses the
// latest actual present time to recalculate the next target present time.
func (dt *DisplayTiming) SyncPastTiming() {
	if !dt.IsEnabled() {
		return
	}

	var count uint32
	ret := vk.GetPastPresentationTimingGOOGLE(dt.device, dt.swapchain, &count, nil)
	if ret != vk.Success || count == 0 {
		return
	}
	timings := make([]vk.PastPresentationTimingGOOGLE, count)
	ret = vk.GetPastPresentationTimingGOOGLE(dt.device, dt.swapchain, &count, &timings[0])
	if ret != vk.Success || count == 0 {
		return
	}

	// Use the latest actual present time as the anchor for the next target.
	latest := timings[count-1]
	latest.Deref()

	if latest.ActualPresentTime > 0 {
		// Target the next refresh cycle after the most recent actual presentation.
		dt.targetTime = latest.ActualPresentTime + dt.refreshDuration
	}
}

// NextPresentInfo returns a PresentTimesInfoGOOGLE to chain into PresentInfo.PNext.
// Returns nil when display timing is disabled — the caller can safely ignore it.
//
//	timingInfo := dt.NextPresentInfo()
//	if timingInfo != nil {
//	    presentInfo.PNext = unsafe.Pointer(timingInfo.Ref())
//	}
func (dt *DisplayTiming) NextPresentInfo() *vk.PresentTimesInfoGOOGLE {
	if !dt.IsEnabled() {
		return nil
	}

	dt.SyncPastTiming()
	dt.presentID++

	return &vk.PresentTimesInfoGOOGLE{
		SType:          vk.StructureTypePresentTimesInfoGoogle,
		SwapchainCount: 1,
		PTimes: []vk.PresentTimeGOOGLE{{
			PresentID:          dt.presentID,
			DesiredPresentTime: dt.targetTime,
		}},
	}
}

// ChainPresentInfo is a convenience that sets presentInfo.PNext to the
// display timing info when enabled. It is a no-op when timing is disabled.
func (dt *DisplayTiming) ChainPresentInfo(presentInfo *vk.PresentInfo) {
	if !dt.IsEnabled() {
		return
	}
	timingInfo := dt.NextPresentInfo()
	if timingInfo == nil {
		return
	}
	presentInfo.PNext = unsafe.Pointer(timingInfo.Ref())
}
