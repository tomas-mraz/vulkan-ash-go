package ash

import (
	"fmt"
	"log/slog"
	"strings"
	"unsafe"

	vk "github.com/tomas-mraz/vulkan"
)

var debug = false

type Device struct {
	Device   vk.Device
	Instance vk.Instance
	//Surface     vk.Surface
	GpuDevice   vk.PhysicalDevice
	deviceQueue vk.Queue
	dbg         vk.DebugReportCallback
}

func SetDebug(state bool) {
	debug = state
}

func (v *Device) GetDebugCallback() vk.DebugReportCallback {
	return v.dbg
}

// Destroy waits for the device to be idle and tears down the device,
// debug callback, surface, and instance.
func (v *Device) Destroy() {
	vk.DeviceWaitIdle(v.Device)
	vk.DestroyDevice(v.Device, nil)
	if v.dbg != vk.NullDebugReportCallback {
		vk.DestroyDebugReportCallback(v.Instance, v.dbg, nil)
	}
	vk.DestroySurface(v.Instance, v.Surface, nil)
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
// the given Device API version (created via vk.MakeVersion).
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
		extNames = append(extNames,
			vk.ToString(ext.ExtensionName[:]))
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

// DeviceOptions configures device creation for NewDeviceWithOptions.
type DeviceOptions struct {
	DeviceExtensions []string
	PNextChain       unsafe.Pointer // pNext chain for VkDeviceCreateInfo
	EnabledFeatures  *vk.PhysicalDeviceFeatures
	ApiVersion       uint32 // Device API version, e.g. vk.MakeVersion(1,2,0). 0 defaults to 1.0.
}

// NewDevice creates a Device device with custom options for extensions and features.
func NewDevice(appName string, instanceExtensions []string, createSurfaceFunc func(instance vk.Instance, window uintptr) (vk.Surface, error), window uintptr, opts *DeviceOptions) (Device, error) {

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
		instanceExtensions = append(instanceExtensions,
			"VK_EXT_debug_report\x00")
	}

	// ANDROID:
	// these layers must be included in APK,
	// see Android.mk and ValidationLayers.mk
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
	vo := Device{}
	err := vk.Error(vk.CreateInstance(&instanceCreateInfo, nil, &vo.Instance))
	if err != nil {
		err = fmt.Errorf("vk.CreateInstance failed with %s", err)
		return vo, err
	}
	err = vk.InitInstance(vo.Instance)
	if err != nil {
		return Device{}, err
	}

	vo.Surface, err = createSurfaceFunc(vo.Instance, window) // Android use a different way to get surface
	if err != nil {
		vk.DestroyInstance(vo.Instance, nil)
		err = fmt.Errorf("create surface failed with %s", err)
		return Device{}, err
	}
	var gpuDevices []vk.PhysicalDevice
	if gpuDevices, err = getPhysicalDevices(vo.Instance); err != nil {
		gpuDevices = nil
		vk.DestroySurface(vo.Instance, vo.Surface, nil)
		vk.DestroyInstance(vo.Instance, nil)
		return Device{}, err
	}

	slog.Debug(fmt.Sprintf("Found %d GPUs", len(gpuDevices)))
	for _, gpu := range gpuDevices {
		var aaa vk.PhysicalDeviceProperties
		vk.GetPhysicalDeviceProperties(gpu, &aaa)
		aaa.Deref()
		slog.Debug("Listed GPU: " + trimCString(aaa.DeviceName[:]))
		aaa.Free()
	}

	vo.GpuDevice = gpuDevices[0] //FIXME select GPU device
	existingExtensions = GetDeviceExtensions(vo.GpuDevice)
	slog.Debug(fmt.Sprintf("Device extensions: %v", existingExtensions))

	// Phase 3: vk.CreateDevice with vk.DeviceCreateInfo (a logical device)

	// ANDROID:
	// these layers must be included in APK,
	// "VK_LAYER_KHRONOS_validation\x00",
	// "VK_LAYER_LUNARG_api_dump\x00",
	deviceLayers := make([]string, 0)

	queueCreateInfos := []vk.DeviceQueueCreateInfo{{
		SType:            vk.StructureTypeDeviceQueueCreateInfo,
		QueueCount:       1,
		PQueuePriorities: []float32{1.0},
	}}
	var deviceExtensions []string
	if vo.Surface != vk.NullSurface {
		deviceExtensions = append(deviceExtensions, "VK_KHR_swapchain\x00")
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
		EnabledLayerCount:       uint32(len(deviceLayers)),
		PpEnabledLayerNames:     deviceLayers,
	}
	if opts != nil && opts.PNextChain != nil {
		deviceCreateInfo.PNext = opts.PNextChain
	}
	if opts != nil && opts.EnabledFeatures != nil {
		deviceCreateInfo.PEnabledFeatures = []vk.PhysicalDeviceFeatures{*opts.EnabledFeatures}
	}
	var device vk.Device // we choose the first GPU available for this device
	err = vk.Error(vk.CreateDevice(vo.GpuDevice, &deviceCreateInfo, nil, &device))
	if err != nil {
		gpuDevices = nil
		vk.DestroySurface(vo.Instance, vo.Surface, nil)
		vk.DestroyInstance(vo.Instance, nil)
		err = fmt.Errorf("vk.CreateDevice failed with %s", err)
		return vo, err
	}
	vo.Device = device
	var queue vk.Queue
	vk.GetDeviceQueue(device, 0, 0, &queue)
	vo.deviceQueue = queue

	if debug {
		// Phase 4: vk.CreateDebugReportCallback

		dbgCreateInfo := vk.DebugReportCallbackCreateInfo{
			SType:       vk.StructureTypeDebugReportCallbackCreateInfo,
			Flags:       vk.DebugReportFlags(vk.DebugReportErrorBit | vk.DebugReportWarningBit),
			PfnCallback: dbgCallbackFunc,
		}
		var dbg vk.DebugReportCallback
		err = vk.Error(vk.CreateDebugReportCallback(vo.Instance, &dbgCreateInfo, nil, &dbg))
		if err != nil {
			err = fmt.Errorf("vk.CreateDebugReportCallback failed with %s", err)
			slog.Warn(err.Error())
			return vo, nil
		}
		vo.dbg = dbg
	}
	return vo, nil
}
