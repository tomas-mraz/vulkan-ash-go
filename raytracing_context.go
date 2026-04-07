package ash

import (
	"encoding/binary"
	"fmt"
	"unsafe"

	vk "github.com/tomas-mraz/vulkan"
)

// RaytracingContext is a lightweight orchestration object for ray tracing helpers.
// It does not own Manager resources; it only groups dependencies commonly needed
// to build acceleration structures and other RT resources.
type RaytracingContext struct {
	device vk.Device
	gpu    vk.PhysicalDevice
	queue  vk.Queue
	cmdCtx *CommandContext

	accelerationStructures []AccelerationStructure
}

// TLASInstance describes one TLAS instance referencing an already built BLAS.
type TLASInstance struct {
	Transform           [12]float32
	InstanceCustomIndex uint32
	Mask                uint8
	SBTRecordOffset     uint32
	Flags               vk.GeometryInstanceFlags
	BLAS                *AccelerationStructure
}

// NewRaytracingContext groups the common RT dependencies without taking ownership.
func NewRaytracingContext(device vk.Device, gpu vk.PhysicalDevice, queue vk.Queue, cmdCtx *CommandContext) RaytracingContext {
	return RaytracingContext{
		device: device,
		gpu:    gpu,
		queue:  queue,
		cmdCtx: cmdCtx,
	}
}

// Destroy releases all acceleration structures created by this context.
func (rt *RaytracingContext) Destroy() {
	if rt == nil {
		return
	}
	// Destroy in reverse order (TLAS before BLAS)
	for i := len(rt.accelerationStructures) - 1; i >= 0; i-- {
		rt.accelerationStructures[i].Destroy()
	}
	rt.accelerationStructures = nil
}

// NewBottomLevelAccelerationStructure builds a BLAS from pre-configured geometry slices.
// The caller is responsible for setting up the geometry and primitive counts;
// this method handles size queries, buffer allocation, scratch, build, and cleanup.
func (rt *RaytracingContext) NewBottomLevelAccelerationStructure(
	geometries []vk.AccelerationStructureGeometry,
	primitiveCounts []uint32,
	flags vk.BuildAccelerationStructureFlags,
) (AccelerationStructure, error) {
	if rt == nil {
		return AccelerationStructure{}, fmt.Errorf("raytracing context is nil")
	}
	if rt.cmdCtx == nil {
		return AccelerationStructure{}, fmt.Errorf("raytracing context has nil command context")
	}
	if len(geometries) == 0 {
		return AccelerationStructure{}, fmt.Errorf("blas requires at least one geometry")
	}

	buildInfo := vk.AccelerationStructureBuildGeometryInfo{
		SType:         vk.StructureTypeAccelerationStructureBuildGeometryInfo,
		Type:          vk.AccelerationStructureTypeBottomLevel,
		Flags:         flags,
		GeometryCount: uint32(len(geometries)),
		PGeometries:   geometries,
	}

	var sizeInfo vk.AccelerationStructureBuildSizesInfo
	sizeInfo.SType = vk.StructureTypeAccelerationStructureBuildSizesInfo
	vk.GetAccelerationStructureBuildSizes(rt.device, vk.AccelerationStructureBuildTypeDevice, &buildInfo, &primitiveCounts[0], &sizeInfo)
	sizeInfo.Deref()

	asBuf, err := NewBufferDeviceLocal(rt.device, rt.gpu, uint64(sizeInfo.AccelerationStructureSize), true,
		vk.BufferUsageFlags(vk.BufferUsageAccelerationStructureStorageBit|vk.BufferUsageShaderDeviceAddressBit))
	if err != nil {
		return AccelerationStructure{}, fmt.Errorf("create BLAS buffer: %w", err)
	}

	var as vk.AccelerationStructure
	if err := vk.Error(vk.CreateAccelerationStructure(rt.device, &vk.AccelerationStructureCreateInfo{
		SType:  vk.StructureTypeAccelerationStructureCreateInfo,
		Buffer: asBuf.Buffer,
		Size:   sizeInfo.AccelerationStructureSize,
		Type:   vk.AccelerationStructureTypeBottomLevel,
	}, nil, &as)); err != nil {
		asBuf.Destroy()
		return AccelerationStructure{}, fmt.Errorf("CreateAccelerationStructure (BLAS): %w", err)
	}

	scratchBuf, err := NewBufferDeviceLocal(rt.device, rt.gpu, uint64(sizeInfo.BuildScratchSize), true,
		vk.BufferUsageFlags(vk.BufferUsageStorageBufferBit|vk.BufferUsageShaderDeviceAddressBit))
	if err != nil {
		vk.DestroyAccelerationStructure(rt.device, as, nil)
		asBuf.Destroy()
		return AccelerationStructure{}, fmt.Errorf("create BLAS scratch buffer: %w", err)
	}
	defer scratchBuf.Destroy()

	buildInfo2 := vk.AccelerationStructureBuildGeometryInfo{
		SType:                    vk.StructureTypeAccelerationStructureBuildGeometryInfo,
		Type:                     vk.AccelerationStructureTypeBottomLevel,
		Flags:                    flags,
		Mode:                     vk.BuildAccelerationStructureModeBuild,
		DstAccelerationStructure: as,
		GeometryCount:            uint32(len(geometries)),
		PGeometries:              geometries,
	}
	setDeviceAddressRT(&buildInfo2.ScratchData, scratchBuf.DeviceAddress)

	rangeInfos := make([]vk.AccelerationStructureBuildRangeInfo, len(primitiveCounts))
	for i, c := range primitiveCounts {
		rangeInfos[i] = vk.AccelerationStructureBuildRangeInfo{PrimitiveCount: c}
	}

	cmd, err := rt.cmdCtx.BeginOneTime()
	if err != nil {
		vk.DestroyAccelerationStructure(rt.device, as, nil)
		asBuf.Destroy()
		return AccelerationStructure{}, fmt.Errorf("BeginOneTime: %w", err)
	}
	vk.CmdBuildAccelerationStructures(cmd, 1, &buildInfo2, [][]vk.AccelerationStructureBuildRangeInfo{rangeInfos})
	if err := rt.cmdCtx.EndOneTime(rt.queue, cmd); err != nil {
		vk.DestroyAccelerationStructure(rt.device, as, nil)
		asBuf.Destroy()
		return AccelerationStructure{}, fmt.Errorf("EndOneTime: %w", err)
	}

	result := AccelerationStructure{
		device:                rt.device,
		AccelerationStructure: as,
		Buffer:                asBuf,
		Type:                  vk.AccelerationStructureTypeBottomLevel,
	}
	rt.accelerationStructures = append(rt.accelerationStructures, result)
	return result, nil
}

// NewTopLevelAccelerationStructure builds a TLAS from the provided BLAS instances.
func (rt *RaytracingContext) NewTopLevelAccelerationStructure(instances []TLASInstance, flags vk.BuildAccelerationStructureFlags) (AccelerationStructure, error) {
	if rt == nil {
		return AccelerationStructure{}, fmt.Errorf("raytracing context is nil")
	}
	if rt.cmdCtx == nil {
		return AccelerationStructure{}, fmt.Errorf("raytracing context has nil command context")
	}
	if len(instances) == 0 {
		return AccelerationStructure{}, fmt.Errorf("tlas requires at least one instance")
	}

	instanceData, err := encodeTLASInstances(instances)
	if err != nil {
		return AccelerationStructure{}, err
	}

	instanceBuf, err := NewBufferHostVisible(rt.device, rt.gpu, instanceData, true,
		vk.BufferUsageFlags(vk.BufferUsageShaderDeviceAddressBit|vk.BufferUsageAccelerationStructureBuildInputReadOnlyBit))
	if err != nil {
		return AccelerationStructure{}, fmt.Errorf("create TLAS instance buffer: %w", err)
	}
	defer instanceBuf.Destroy()

	var instancesData vk.AccelerationStructureGeometryInstancesData
	instancesData.SType = vk.StructureTypeAccelerationStructureGeometryInstancesData
	setDeviceAddressConstRT(&instancesData.Data, instanceBuf.DeviceAddress)

	var geometry vk.AccelerationStructureGeometry
	geometry.SType = vk.StructureTypeAccelerationStructureGeometry
	geometry.GeometryType = vk.GeometryTypeInstances
	geometry.Flags = vk.GeometryFlags(vk.GeometryOpaqueBit)
	setGeometryInstancesRT(&geometry.Geometry, &instancesData)

	primitiveCount := uint32(len(instances))
	buildInfo := vk.AccelerationStructureBuildGeometryInfo{
		SType:         vk.StructureTypeAccelerationStructureBuildGeometryInfo,
		Type:          vk.AccelerationStructureTypeTopLevel,
		Flags:         flags,
		GeometryCount: 1,
		PGeometries:   []vk.AccelerationStructureGeometry{geometry},
	}

	var sizeInfo vk.AccelerationStructureBuildSizesInfo
	sizeInfo.SType = vk.StructureTypeAccelerationStructureBuildSizesInfo
	vk.GetAccelerationStructureBuildSizes(rt.device, vk.AccelerationStructureBuildTypeDevice, &buildInfo, &primitiveCount, &sizeInfo)
	sizeInfo.Deref()

	asBuf, err := NewBufferDeviceLocal(rt.device, rt.gpu, uint64(sizeInfo.AccelerationStructureSize), true,
		vk.BufferUsageFlags(vk.BufferUsageAccelerationStructureStorageBit|vk.BufferUsageShaderDeviceAddressBit))
	if err != nil {
		return AccelerationStructure{}, fmt.Errorf("create TLAS buffer: %w", err)
	}

	var as vk.AccelerationStructure
	if err := vk.Error(vk.CreateAccelerationStructure(rt.device, &vk.AccelerationStructureCreateInfo{
		SType:  vk.StructureTypeAccelerationStructureCreateInfo,
		Buffer: asBuf.Buffer,
		Size:   sizeInfo.AccelerationStructureSize,
		Type:   vk.AccelerationStructureTypeTopLevel,
	}, nil, &as)); err != nil {
		asBuf.Destroy()
		return AccelerationStructure{}, fmt.Errorf("CreateAccelerationStructure (TLAS): %w", err)
	}

	scratchBuf, err := NewBufferDeviceLocal(rt.device, rt.gpu, uint64(sizeInfo.BuildScratchSize), true,
		vk.BufferUsageFlags(vk.BufferUsageStorageBufferBit|vk.BufferUsageShaderDeviceAddressBit))
	if err != nil {
		vk.DestroyAccelerationStructure(rt.device, as, nil)
		asBuf.Destroy()
		return AccelerationStructure{}, fmt.Errorf("create TLAS scratch buffer: %w", err)
	}
	defer scratchBuf.Destroy()

	buildInfo2 := vk.AccelerationStructureBuildGeometryInfo{
		SType:                    vk.StructureTypeAccelerationStructureBuildGeometryInfo,
		Type:                     vk.AccelerationStructureTypeTopLevel,
		Flags:                    flags,
		Mode:                     vk.BuildAccelerationStructureModeBuild,
		DstAccelerationStructure: as,
		GeometryCount:            1,
		PGeometries:              []vk.AccelerationStructureGeometry{geometry},
	}
	setDeviceAddressRT(&buildInfo2.ScratchData, scratchBuf.DeviceAddress)

	rangeInfos := []vk.AccelerationStructureBuildRangeInfo{{
		PrimitiveCount: primitiveCount,
	}}

	cmd, err := rt.cmdCtx.BeginOneTime()
	if err != nil {
		vk.DestroyAccelerationStructure(rt.device, as, nil)
		asBuf.Destroy()
		return AccelerationStructure{}, fmt.Errorf("BeginOneTime: %w", err)
	}
	vk.CmdBuildAccelerationStructures(cmd, 1, &buildInfo2, [][]vk.AccelerationStructureBuildRangeInfo{rangeInfos})
	if err := rt.cmdCtx.EndOneTime(rt.queue, cmd); err != nil {
		vk.DestroyAccelerationStructure(rt.device, as, nil)
		asBuf.Destroy()
		return AccelerationStructure{}, fmt.Errorf("EndOneTime: %w", err)
	}

	result := AccelerationStructure{
		device:                rt.device,
		AccelerationStructure: as,
		Buffer:                asBuf,
		Type:                  vk.AccelerationStructureTypeTopLevel,
	}
	rt.accelerationStructures = append(rt.accelerationStructures, result)
	return result, nil
}

func encodeTLASInstances(instances []TLASInstance) ([]byte, error) {
	data := make([]byte, len(instances)*64)
	for i := range instances {
		inst := instances[i]
		if inst.BLAS == nil {
			return nil, fmt.Errorf("tlas instance %d has nil BLAS", i)
		}
		if inst.InstanceCustomIndex > 0xFFFFFF {
			return nil, fmt.Errorf("tlas instance %d custom index exceeds 24 bits", i)
		}
		if inst.SBTRecordOffset > 0xFFFFFF {
			return nil, fmt.Errorf("tlas instance %d SBT record offset exceeds 24 bits", i)
		}
		if uint32(inst.Flags) > 0xFF {
			return nil, fmt.Errorf("tlas instance %d flags exceed 8 bits", i)
		}

		blasAddr := inst.BLAS.GetDeviceAddress()
		if blasAddr == 0 {
			return nil, fmt.Errorf("tlas instance %d BLAS has no device address", i)
		}

		offset := i * 64
		copy(data[offset:offset+48], unsafe.Slice((*byte)(unsafe.Pointer(&inst.Transform[0])), 48))
		writePackedUint24AndByte(data[offset+48:offset+52], inst.InstanceCustomIndex, inst.Mask)
		writePackedUint24AndByte(data[offset+52:offset+56], inst.SBTRecordOffset, uint8(inst.Flags))
		binary.LittleEndian.PutUint64(data[offset+56:offset+64], uint64(blasAddr))
	}
	return data, nil
}

func writePackedUint24AndByte(dst []byte, value uint32, extra uint8) {
	dst[0] = byte(value)
	dst[1] = byte(value >> 8)
	dst[2] = byte(value >> 16)
	dst[3] = extra
}

func setGeometryInstancesRT(data *vk.AccelerationStructureGeometryData, inst *vk.AccelerationStructureGeometryInstancesData) {
	cInst, _ := inst.PassRef()
	src := unsafe.Slice((*byte)(unsafe.Pointer(cInst)), len(*data))
	copy((*data)[:], src)
}
