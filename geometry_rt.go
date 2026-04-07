package ash

import vk "github.com/tomas-mraz/vulkan"

// TriangleGeometryDesc describes a triangle geometry for BLAS construction.
type TriangleGeometryDesc struct {
	VertexAddress vk.DeviceAddress
	VertexFormat  vk.Format    // e.g. vk.FormatR32g32b32Sfloat
	VertexStride  uint32       // e.g. 12 for 3×float32
	MaxVertex     uint32
	IndexAddress  vk.DeviceAddress
	IndexType     vk.IndexType // e.g. vk.IndexTypeUint32
	Flags         vk.GeometryFlags
}

// NewTriangleGeometry builds a vk.AccelerationStructureGeometry from a triangle description.
func NewTriangleGeometry(desc TriangleGeometryDesc) vk.AccelerationStructureGeometry {
	var trianglesData vk.AccelerationStructureGeometryTrianglesData
	trianglesData.SType = vk.StructureTypeAccelerationStructureGeometryTrianglesData
	trianglesData.VertexFormat = desc.VertexFormat
	vk.SetDeviceAddressConst(&trianglesData.VertexData, desc.VertexAddress)
	trianglesData.VertexStride = vk.DeviceSize(desc.VertexStride)
	trianglesData.MaxVertex = desc.MaxVertex
	trianglesData.IndexType = desc.IndexType
	vk.SetDeviceAddressConst(&trianglesData.IndexData, desc.IndexAddress)

	var geometry vk.AccelerationStructureGeometry
	geometry.SType = vk.StructureTypeAccelerationStructureGeometry
	geometry.GeometryType = vk.GeometryTypeTriangles
	geometry.Flags = desc.Flags
	vk.SetGeometryTriangles(&geometry.Geometry, &trianglesData)
	return geometry
}
