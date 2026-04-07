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

const MACOS = "darwin"

var debug = false

type Manager struct {
	Device   vk.Device
	Instance vk.Instance
	Surface  vk.Surface
	Gpu      vk.PhysicalDevice
	Queue    vk.Queue
	debugClb vk.DebugReportCallback
}

// DeviceOptions configures device creation for NewDeviceWithOptions.
type DeviceOptions struct {
	DeviceExtensions []string
	PNextChain       unsafe.Pointer // pNext chain for VkDeviceCreateInfo
	EnabledFeatures  *vk.PhysicalDeviceFeatures
	ApiVersion       uint32 // Manager API version, e.g. vk.MakeVersion(1,2,0). 0 defaults to 1.0.
}

// CreateSurfaceFunc creates a VkSurface from a Vulkan instance.
// On desktop (GLFW) this typically calls window.CreateWindowSurface;
// on Android it receives the native window pointer.
type CreateSurfaceFunc func(instance vk.Instance) (vk.Surface, error)

// NewManager creates a Manager device with custom options for extensions and features.
func NewManager(appName string, instanceExtensions []string, createSurfaceFn CreateSurfaceFunc, opts *DeviceOptions) (Manager, error) {

	apiVersion := vk.MakeVersion(1, 0, 0)
	if opts != nil && opts.ApiVersion != 0 {
		apiVersion = opts.ApiVersion
	}
	var appInfo = &vk.ApplicationInfo{
		SType:              vk.StructureTypeApplicationInfo,
		ApiVersion:         apiVersion,
		ApplicationVersion: vk.MakeVersion(1, 0, 0),
		PApplicationName:   []byte(appName + "\x00"),
		PEngineName:        []byte("no engine\x00"),
	}

	// Phase 1: vk.CreateInstance with vk.InstanceCreateInfo

	existingExtensions := getInstanceExtensions()
	slog.Debug(fmt.Sprintf("Instance extensions: %v", existingExtensions))

	// instanceExtensions := vk.GetRequiredInstanceExtensions()
	if debug {
		instanceExtensions = append(instanceExtensions, vk.ExtDebugReportExtensionName)
	}
	if runtime.GOOS == MACOS {
		instanceExtensions = append(instanceExtensions, vk.KhrPortabilityEnumerationExtensionName)
	}

	// ANDROID: these layers must be included in APK
	instanceLayers := []string{
		"VK_LAYER_KHRONOS_validation\x00",
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
	if runtime.GOOS == MACOS {
		instanceCreateInfo.Flags = vk.InstanceCreateFlags(vk.InstanceCreateEnumeratePortabilityBit)
	}

	manager := Manager{}
	err := vk.Error(vk.CreateInstance(&instanceCreateInfo, nil, &manager.Instance))
	if err != nil {
		err = fmt.Errorf("vk.CreateInstance failed with %s", err)
		return manager, err
	}
	err = vk.InitInstance(manager.Instance)
	if err != nil {
		return Manager{}, err
	}

	if createSurfaceFn != nil {
		manager.Surface, err = createSurfaceFn(manager.Instance)
		if err != nil {
			return Manager{}, err
		}
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

	manager.Gpu = selectPhysicalDevice(gpuDevices)
	existingExtensions = GetDeviceExtensions(manager.Gpu)
	slog.Debug(fmt.Sprintf("Device extensions: %v", existingExtensions))

	queueCreateInfos := []vk.DeviceQueueCreateInfo{{
		SType:            vk.StructureTypeDeviceQueueCreateInfo,
		QueueCount:       1,
		PQueuePriorities: []float32{1.0},
	}}
	var deviceExtensions []string
	if manager.Surface != vk.NullSurface {
		deviceExtensions = append(deviceExtensions, vk.KhrSwapchainExtensionName)
	}
	if runtime.GOOS == MACOS && slices.Contains(existingExtensions, vk.KhrPortabilitySubsetExtensionName) {
		deviceExtensions = append(deviceExtensions, vk.KhrPortabilitySubsetExtensionName)
	}
	if opts != nil {
		for _, ext := range opts.DeviceExtensions {
			deviceExtensions = append(deviceExtensions, ext)
		}
	}
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

	if debug {
		dbgCreateInfo := vk.DebugReportCallbackCreateInfo{
			SType:       vk.StructureTypeDebugReportCallbackCreateInfo,
			Flags:       vk.DebugReportFlags(vk.DebugReportErrorBit | vk.DebugReportWarningBit),
			PfnCallback: dbgCallbackFunc,
		}
		err = vk.Error(vk.CreateDebugReportCallback(manager.Instance, &dbgCreateInfo, nil, &manager.debugClb))
		if err != nil {
			err = fmt.Errorf("vk.CreateDebugReportCallback failed with %s", err)
			slog.Warn(err.Error())
			return manager, nil
		}
	}
	return manager, nil
}

func SetDebug(state bool) {
	debug = state
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

// CheckDeviceApiVersion returns true if the physical device supports at least
// the given Manager API version (created via vk.MakeVersion).
func CheckDeviceApiVersion(gpu vk.PhysicalDevice, minVersion uint32) (ok bool, deviceVersion uint32) {
	var props vk.PhysicalDeviceProperties
	vk.GetPhysicalDeviceProperties(gpu, &props)
	props.Deref()
	return props.ApiVersion >= minVersion, props.ApiVersion
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

func selectPhysicalDevice(gpus []vk.PhysicalDevice) vk.PhysicalDevice {
	bestGPU := gpus[0]
	bestName := ""
	bestType := vk.PhysicalDeviceTypeOther
	bestScore := 0

	gpuTypes := []string{"Other", "Integrated GPU", "Discrete GPU", "Virtual GPU", "CPU"}

	var name string
	var gpuType vk.PhysicalDeviceType
	var score int
	// Vulkan put better gpu on the beginning of the list => same score does not win
	for i, gpu := range gpus {
		var props vk.PhysicalDeviceProperties
		vk.GetPhysicalDeviceProperties(gpu, &props)
		props.Deref()

		name = vk.ToString(props.DeviceName[:])
		gpuType = props.DeviceType
		score = 0

		switch props.ApiVersion {
		case vk.ApiVersion13:
			score += 300
		case vk.ApiVersion12:
			score += 200
		case vk.ApiVersion11:
			score += 100
		}
		if i == 0 {
			score += 100 // selected by vulkan as best
		}
		if gpuType == vk.PhysicalDeviceTypeDiscreteGpu {
			score += 100 // more powerful
		}
		if runtime.GOOS == MACOS && strings.Contains(strings.ToLower(name), "kosmickrisp") {
			score += 500 // preferred before MoltenVK
		}
		fmt.Printf("Listed GPU: %s (type=%s, score=%d)\n", name, gpuTypes[gpuType], score)

		if score > bestScore {
			bestGPU = gpu
			bestName = name
			bestType = gpuType
			bestScore = score
		}
		props.Free()
	}

	fmt.Printf("Selected GPU: %s (type=%s, score=%d)\n", bestName, gpuTypes[bestType], bestScore)
	return bestGPU
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
