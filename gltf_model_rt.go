package ash

import vk "github.com/tomas-mraz/vulkan"

// GLTFPrimitive holds per-primitive GPU resources and metadata used by the
// ray tracing examples.
type GLTFPrimitive struct {
	VertexBuffer  BufferResource
	IndexBuffer   BufferResource
	VertexCount   uint32
	TriangleCount uint32
	Transform     [12]float32
	BaseColorTex  int32
	OcclusionTex  int32
}

// GLTFModel owns primitive buffers, textures, a geometry buffer, and a single
// BLAS for the loaded glTF model.
type GLTFModel struct {
	device         vk.Device
	Primitives     []GLTFPrimitive
	GeometryBuffer BufferResource
	BLAS           AccelerationStructure
	Textures       []ImageResource
}

func NewGLTFModel(device vk.Device, primitives []GLTFPrimitive, geometryBuffer BufferResource, blas AccelerationStructure, textures []ImageResource) GLTFModel {
	return GLTFModel{
		device:         device,
		Primitives:     primitives,
		GeometryBuffer: geometryBuffer,
		BLAS:           blas,
		Textures:       textures,
	}
}

func (m *GLTFModel) Destroy() {
	if m == nil {
		return
	}
	for i := range m.Primitives {
		m.Primitives[i].IndexBuffer.Destroy()
		m.Primitives[i].VertexBuffer.Destroy()
	}
	for i := range m.Textures {
		m.Textures[i].Destroy()
	}
	m.GeometryBuffer.Destroy()
}
