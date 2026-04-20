package ash

import (
	"fmt"
	"log/slog"
	"runtime"
	"slices"
	"strings"
	"unsafe"

	vk "github.com/tomas-mraz/vulkan"
)

const (
	OSMac     = "darwin"
	OSAndroid = "android"
)

var (
	validationLayersSwitch = false
	debugSwitch            = false
)

type Manager struct {
	Device   vk.Device
	Instance vk.Instance
	Surface  vk.Surface
	Gpu      vk.PhysicalDevice
	Queue    vk.Queue
	debugClb vk.DebugReportCallback
}

// DeviceOptions configures instance and device creation for NewManager.
type DeviceOptions struct {
	InstanceExtensions []string
	DeviceExtensions   []string
	PNextChain         unsafe.Pointer // pNext chain for VkDeviceCreateInfo
	EnabledFeatures    *vk.PhysicalDeviceFeatures
	ApiVersion         uint32 // Manager API version, e.g. vk.MakeVersion(1,2,0). 0 defaults to 1.0.
}

// CreateSurfaceFunc creates a VkSurface from a Vulkan instance.
// On desktop (GLFW) this typically calls window.CreateWindowSurface;
// on Android it receives the native window pointer.
type CreateSurfaceFunc func(instance vk.Instance) (vk.Surface, error)

// NewManager creates a Manager device with custom options for extensions and features.
func NewManager(appName string, createSurfaceFn CreateSurfaceFunc, opts *DeviceOptions) (Manager, error) {

	// Phase 1: Create Instance

	apiVersion := vk.MakeVersion(1, 0, 0)
	var instanceExtensions []string
	var deviceExtensions []string
	if opts != nil {
		if opts.ApiVersion != 0 {
			apiVersion = opts.ApiVersion
		}
		instanceExtensions = opts.InstanceExtensions
		deviceExtensions = opts.DeviceExtensions
	}
	var appInfo = &vk.ApplicationInfo{
		SType:              vk.StructureTypeApplicationInfo,
		ApiVersion:         apiVersion,
		ApplicationVersion: vk.MakeVersion(1, 0, 0),
		PApplicationName:   []byte(appName + "\x00"),
		PEngineName:        []byte("no engine\x00"),
	}

	slog.Debug(fmt.Sprintf("Instance has extensions: %v", getInstanceExtensions()))
	if debugSwitch {
		instanceExtensions = append(instanceExtensions, vk.ExtDebugReportExtensionName)
	}
	if runtime.GOOS == OSMac {
		instanceExtensions = append(instanceExtensions, vk.KhrPortabilityEnumerationExtensionName)
	}
	if runtime.GOOS == OSAndroid {
		instanceExtensions = append(instanceExtensions, vk.GetAndroidRequiredInstanceExtensions()...)
	}
	instanceExtensions = makeUniqueCStrings(instanceExtensions)

	// ANDROID: these layers must be included in APK
	instanceLayers := make([]string, 0)
	if validationLayersSwitch {
		instanceLayers = append(instanceLayers, "VK_LAYER_KHRONOS_validation\x00")
		// "VK_LAYER_LUNARG_api_dump\x00",
	}

	instanceCreateInfo := vk.InstanceCreateInfo{
		SType:                   vk.StructureTypeInstanceCreateInfo,
		PApplicationInfo:        appInfo,
		EnabledExtensionCount:   uint32(len(instanceExtensions)),
		PpEnabledExtensionNames: instanceExtensions,
		EnabledLayerCount:       uint32(len(instanceLayers)),
		PpEnabledLayerNames:     instanceLayers,
	}
	// make possible detect KosmicKrisp and MoltenVK devices
	if runtime.GOOS == OSMac {
		instanceCreateInfo.Flags = vk.InstanceCreateFlags(vk.InstanceCreateEnumeratePortabilityBit)
	}

	manager := Manager{}
	err := vk.Error(vk.CreateInstance(&instanceCreateInfo, nil, &manager.Instance))
	if err != nil {
		err = fmt.Errorf("vk.CreateInstance failed with %s", err)
		return manager, err
	}
	if err = vk.InitInstance(manager.Instance); err != nil {
		vk.DestroyInstance(manager.Instance, nil)
		return Manager{}, err
	}

	// Phase 2: Create Surface and Device

	if createSurfaceFn != nil {
		manager.Surface, err = createSurfaceFn(manager.Instance)
		if err != nil {
			vk.DestroyInstance(manager.Instance, nil)
			return Manager{}, fmt.Errorf("create surface failed with %s", err)
		}
	}
	var gpuDevices []vk.PhysicalDevice
	if gpuDevices, err = getPhysicalDevices(manager.Instance); err != nil {
		gpuDevices = nil
		vk.DestroySurface(manager.Instance, manager.Surface, nil)
		vk.DestroyInstance(manager.Instance, nil)
		return Manager{}, err
	}

	slog.Debug(fmt.Sprintf("Found %d GPUs", len(gpuDevices)))
	for _, gpu := range gpuDevices {
		var aaa vk.PhysicalDeviceProperties
		vk.GetPhysicalDeviceProperties(gpu, &aaa)
		aaa.Deref()
		slog.Debug("Listed GPU: " + trimCString(aaa.DeviceName[:]))
		aaa.Free()
	}

	var gpuName string
	manager.Gpu, gpuName, err = selectPhysicalDevice(gpuDevices, apiVersion)
	if err != nil {
		vk.DestroySurface(manager.Instance, manager.Surface, nil)
		vk.DestroyInstance(manager.Instance, nil)
		return Manager{}, err
	}

	existingDeviceExtensions := GetDeviceExtensions(manager.Gpu)
	slog.Debug(fmt.Sprintf("Device has extensions: %v", existingDeviceExtensions))

	// Verify every user-declared device extension is actually reported by the
	// selected GPU. vkCreateDevice would otherwise fail with
	// VK_ERROR_EXTENSION_NOT_PRESENT, which gives no hint about which ones are
	// missing. Extensions the Manager appends itself (swapchain, portability
	// subset, display timing) are guarded separately below so they're only
	// added when present, and therefore aren't checked here.
	if len(deviceExtensions) > 0 {
		if ok, missing := containsAll(existingDeviceExtensions, deviceExtensions); !ok {
			vk.DestroySurface(manager.Instance, manager.Surface, nil)
			vk.DestroyInstance(manager.Instance, nil)
			return Manager{}, fmt.Errorf("missing required device extensions %v on GPU %q", missing, gpuName)
		}
	}

	queueCreateInfos := []vk.DeviceQueueCreateInfo{{
		SType:            vk.StructureTypeDeviceQueueCreateInfo,
		QueueCount:       1,
		PQueuePriorities: []float32{1.0},
	}}

	if manager.Surface != vk.NullSurface {
		deviceExtensions = append(deviceExtensions, vk.KhrSwapchainExtensionName)
	}
	if runtime.GOOS == OSMac && slices.Contains(existingDeviceExtensions, vk.KhrPortabilitySubsetExtensionName) {
		deviceExtensions = append(deviceExtensions, vk.KhrPortabilitySubsetExtensionName)
	}
	if runtime.GOOS == OSAndroid && slices.Contains(existingDeviceExtensions, vk.GoogleDisplayTimingExtensionName) {
		deviceExtensions = append(deviceExtensions, vk.GoogleDisplayTimingExtensionName)
	}
	deviceExtensions = makeUniqueCStrings(deviceExtensions)

	deviceCreateInfo := vk.DeviceCreateInfo{
		SType:                   vk.StructureTypeDeviceCreateInfo,
		QueueCreateInfoCount:    uint32(len(queueCreateInfos)),
		PQueueCreateInfos:       queueCreateInfos,
		EnabledExtensionCount:   uint32(len(deviceExtensions)),
		PpEnabledExtensionNames: deviceExtensions,
	}
	if opts != nil && opts.PNextChain != nil {
		deviceCreateInfo.PNext = opts.PNextChain
	}
	if opts != nil && opts.EnabledFeatures != nil {
		deviceCreateInfo.PEnabledFeatures = []vk.PhysicalDeviceFeatures{*opts.EnabledFeatures}
	}

	// create the logical device for the selected physical device
	err = vk.Error(vk.CreateDevice(manager.Gpu, &deviceCreateInfo, nil, &manager.Device))
	if err != nil {
		gpuDevices = nil
		vk.DestroySurface(manager.Instance, manager.Surface, nil)
		vk.DestroyInstance(manager.Instance, nil)
		err = fmt.Errorf("vk.CreateDevice failed with %s", err)
		return manager, err
	}
	vk.GetDeviceQueue(manager.Device, 0, 0, &manager.Queue)

	if debugSwitch {
		dbgCreateInfo := vk.DebugReportCallbackCreateInfo{
			SType:       vk.StructureTypeDebugReportCallbackCreateInfo,
			Flags:       vk.DebugReportFlags(vk.DebugReportErrorBit | vk.DebugReportWarningBit),
			PfnCallback: dbgCallbackFunc,
		}
		err = vk.Error(vk.CreateDebugReportCallback(manager.Instance, &dbgCreateInfo, nil, &manager.debugClb))
		if err != nil {
			slog.Warn(fmt.Errorf("vk.CreateDebugReportCallback failed with %s", err).Error())
			return manager, nil
		}
	}
	return manager, nil
}

func SetDebug(state bool) {
	debugSwitch = state
}

func SetValidations(state bool) {
	validationLayersSwitch = state
}

func (v *Manager) GetDebugCallback() vk.DebugReportCallback {
	return v.debugClb
}

// Destroy waits for the device to be idle and tears down the device,
// debug callback, surface, and instance.
func (v *Manager) Destroy() {
	vk.DeviceWaitIdle(v.Device)
	vk.DestroyDevice(v.Device, nil)
	if v.debugClb != vk.NullDebugReportCallback {
		vk.DestroyDebugReportCallback(v.Instance, v.debugClb, nil)
	}
	if v.Surface != vk.NullSurface {
		vk.DestroySurface(v.Instance, v.Surface, nil)
	}
	vk.DestroyInstance(v.Instance, nil)
}

// NewExtentSize needs for Wayland
func NewExtentSize(width, height int) vk.Extent2D {
	return vk.Extent2D{
		Width:  uint32(width),
		Height: uint32(height),
	}
}

// QuerySurfaceExtent returns the current platform surface extent reported by the driver.
// On Android this reflects rotation-induced size changes after a config change event.
// Falls back to the provided extent when the capabilities query fails or when the
// driver returns the Wayland-style "undefined" extent (all-ones) or a zero extent.
func (v *Manager) QuerySurfaceExtent(fallback vk.Extent2D) vk.Extent2D {
	if v == nil || v.Surface == vk.NullSurface || v.Gpu == nil {
		return fallback
	}
	var caps vk.SurfaceCapabilities
	if err := vk.Error(vk.GetPhysicalDeviceSurfaceCapabilities(v.Gpu, v.Surface, &caps)); err != nil {
		return fallback
	}
	caps.Deref()
	caps.CurrentExtent.Deref()
	w, h := caps.CurrentExtent.Width, caps.CurrentExtent.Height
	if w == vk.MaxUint32 && h == vk.MaxUint32 {
		return fallback
	}
	if w == 0 || h == 0 {
		return fallback
	}
	return vk.Extent2D{Width: w, Height: h}
}

func GetDeviceExtensions(gpu vk.PhysicalDevice) (extNames []string) {
	var deviceExtLen uint32
	ret := vk.EnumerateDeviceExtensionProperties(gpu, "", &deviceExtLen, nil)
	check(ret, "vk.EnumerateDeviceExtensionProperties")
	deviceExt := make([]vk.ExtensionProperties, deviceExtLen)
	ret = vk.EnumerateDeviceExtensionProperties(gpu, "", &deviceExtLen, deviceExt)
	check(ret, "vk.EnumerateDeviceExtensionProperties")
	for _, ext := range deviceExt {
		ext.Deref()
		extNames = append(extNames,
			vk.ToString(ext.ExtensionName[:]))
	}
	return extNames
}

// CheckDeviceExtensions returns true if the physical device supports all
// required extensions. Missing extensions are returned in the second value.
func CheckDeviceExtensions(gpu vk.PhysicalDevice, required []string) (ok bool, missing []string) {
	available := make(map[string]struct{})
	for _, name := range GetDeviceExtensions(gpu) {
		available[name] = struct{}{}
	}
	for _, req := range required {
		clean := strings.TrimRight(req, "\x00")
		if _, found := available[clean]; !found {
			missing = append(missing, clean)
		}
	}
	return len(missing) == 0, missing
}

// RequireInstanceApiVersion ensures the installed Vulkan loader supports at
// least minVersion. Must be called after vk.Init — i.e. after Host.InitVulkan
// or its manual equivalent — because it depends on the dispatch table.
//
// Returns a descriptive error when the loader is older, which gives callers a
// clearer failure mode than the raw VK_ERROR_INCOMPATIBLE_DRIVER that
// vkCreateInstance would otherwise produce if ApplicationInfo.ApiVersion is
// higher than the loader supports. Pre-1.1 loaders don't expose
// vkEnumerateInstanceVersion; in that case Result != Success and we fall back
// to reporting 1.0, which is the spec-defined default.
func RequireInstanceApiVersion(minVersion uint32) error {
	var actual uint32
	if ret := vk.EnumerateInstanceVersion(&actual); ret != vk.Success || actual == 0 {
		actual = vk.MakeVersion(1, 0, 0)
	}
	if actual < minVersion {
		return fmt.Errorf("Vulkan %s required, instance loader reports %s",
			vk.Version(minVersion), vk.Version(actual))
	}
	return nil
}

func getInstanceExtensions() (extNames []string) {
	var instanceExtLen uint32
	ret := vk.EnumerateInstanceExtensionProperties("", &instanceExtLen, nil)
	check(ret, "vk.EnumerateInstanceExtensionProperties")
	instanceExt := make([]vk.ExtensionProperties, instanceExtLen)
	ret = vk.EnumerateInstanceExtensionProperties("", &instanceExtLen, instanceExt)
	check(ret, "vk.EnumerateInstanceExtensionProperties")
	for _, ext := range instanceExt {
		ext.Deref()
		extNames = append(extNames, vk.ToString(ext.ExtensionName[:]))
	}
	return extNames
}

func getPhysicalDevices(instance vk.Instance) ([]vk.PhysicalDevice, error) {
	var gpuCount uint32
	err := vk.Error(vk.EnumeratePhysicalDevices(instance, &gpuCount, nil))
	if err != nil {
		err = fmt.Errorf("vk.EnumeratePhysicalDevices failed with %s", err)
		return nil, err
	}
	if gpuCount == 0 {
		err = fmt.Errorf("getPhysicalDevice: no GPUs found on the system")
		return nil, err
	}
	gpuList := make([]vk.PhysicalDevice, gpuCount)
	err = vk.Error(vk.EnumeratePhysicalDevices(instance, &gpuCount, gpuList))
	if err != nil {
		err = fmt.Errorf("vk.EnumeratePhysicalDevices failed with %s", err)
		return nil, err
	}
	return gpuList, nil
}

// selectPhysicalDevice scores each enumerated GPU and returns the best match
// along with its reported device name. When minApiVersion > 0, GPUs whose
// VkPhysicalDeviceProperties.apiVersion is below the threshold are excluded
// from selection — they're still listed in the log with a [skipped: ...]
// marker so the failure is diagnosable — and if no GPU qualifies the function
// returns an error instead of a silent fallback. This keeps ray tracing and
// other version-gated apps from proceeding with a GPU that would fail at
// vkCreateDevice time.
//
// The name is returned alongside the handle so callers can include it in
// diagnostics (e.g. extension-missing errors) without a second
// vkGetPhysicalDeviceProperties round-trip.
func selectPhysicalDevice(gpus []vk.PhysicalDevice, minApiVersion uint32) (vk.PhysicalDevice, string, error) {
	var bestGPU vk.PhysicalDevice
	bestName := ""
	bestScore := -1

	gpuTypes := []string{"other", "integrated", "discrete", "virtual", "cpu"}

	// Vulkan puts the preferred GPU first in the list; an equal-score tie
	// therefore goes to the first candidate.
	for i, gpu := range gpus {
		var props vk.PhysicalDeviceProperties
		vk.GetPhysicalDeviceProperties(gpu, &props)
		props.Deref()

		name := vk.ToString(props.DeviceName[:])
		gpuType := props.DeviceType
		score := 0

		switch props.ApiVersion {
		case vk.ApiVersion13:
			score += 300
		case vk.ApiVersion12:
			score += 200
		case vk.ApiVersion11:
			score += 100
		}
		switch gpuType {
		case vk.PhysicalDeviceTypeDiscreteGpu:
			score += 200 // more powerful
		case vk.PhysicalDeviceTypeIntegratedGpu:
			score += 100 // better than other possibilities
		default:
			// no score boost for other types
		}
		if i == 0 {
			score += 100 // selected by vulkan as best
		}
		if runtime.GOOS == OSMac && strings.Contains(strings.ToLower(name), "kosmickrisp") {
			score += 500 // preferred before MoltenVK
		}

		var skipReason string
		if minApiVersion > 0 && props.ApiVersion < minApiVersion {
			skipReason = fmt.Sprintf(" [skipped: API %s < required %s]", vk.Version(props.ApiVersion), vk.Version(minApiVersion))
		}
		slog.Debug(fmt.Sprintf("listed physical device: %s (type=%s, api=%s, score=%d) %s\n", name, gpuTypes[gpuType], vk.Version(props.ApiVersion), score, skipReason))
		props.Free()

		if skipReason != "" {
			continue
		}
		if score > bestScore {
			bestGPU = gpu
			bestName = name
			bestScore = score
		}
	}

	if bestScore < 0 {
		return bestGPU, "", fmt.Errorf("no physical device satisfies required Vulkan API version %s", vk.Version(minApiVersion))
	}

	slog.Debug(fmt.Sprintf("selected physical device: %s (score=%d)\n", bestName, bestScore))
	return bestGPU, bestName, nil
}

func dbgCallbackFunc(flags vk.DebugReportFlags, objectType vk.DebugReportObjectType, object uint64, location uint64, messageCode int32, pLayerPrefix string, pMessage string, pUserData unsafe.Pointer) vk.Bool32 {
	switch {
	case flags&vk.DebugReportFlags(vk.DebugReportErrorBit) != 0:
		slog.Error(fmt.Sprintf("[%d] %s on layer %s", messageCode, pMessage, pLayerPrefix))
	case flags&vk.DebugReportFlags(vk.DebugReportWarningBit) != 0:
		slog.Warn(fmt.Sprintf("[%d] %s on layer %s", messageCode, pMessage, pLayerPrefix))
	default:
		slog.Warn(fmt.Sprintf("unknown debug message %d (layer %s)", messageCode, pLayerPrefix))
	}
	return vk.False
}
