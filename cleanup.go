package ash

import "github.com/tomas-mraz/vulkan"

// Cleanup collects destroyable objects and runs them in LIFO order.
// Register cleanup right after object creation with Add, then call Destroy at shutdown to tear down everything in reverse order.
type Cleanup struct {
	manager           *Manager
	destroyableObject []Destroyer
}

// Destroyer represents a type that can release its owned resources.
type Destroyer interface {
	Destroy()
}

// Add registers a destroyable object. Objects run in reverse order on Destroy.
func (d *Cleanup) Add(object Destroyer) {
	d.destroyableObject = append(d.destroyableObject, object)
}

// Destroy runs all registered destroyers in LIFO order and clears the list.
func (d *Cleanup) Destroy() {
	if d.manager != nil {
		vulkan.DeviceWaitIdle(d.manager.Device)
	}
	for i := len(d.destroyableObject) - 1; i >= 0; i-- {
		d.destroyableObject[i].Destroy()
	}
	d.destroyableObject = nil
}
