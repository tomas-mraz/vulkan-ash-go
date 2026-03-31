package ash

import (
	"fmt"
	"math"
	"path/filepath"
	"unsafe"

	"github.com/qmuntal/gltf"
	"github.com/qmuntal/gltf/modeler"
	vk "github.com/tomas-mraz/vulkan"
)

type gltfGeometryNode struct {
	VertexBufferDeviceAddress uint64
	IndexBufferDeviceAddress  uint64
	TextureIndexBaseColor     int32
	TextureIndexOcclusion     int32
}

// LoadGLTFModel loads a glTF scene into GPU buffers, creates the geometry SSBO,
// and builds a single BLAS containing one geometry per primitive.
func LoadGLTFModel(dev vk.Device, gpu vk.PhysicalDevice, queue vk.Queue, cmdCtx *VulkanCommandContext, path string) (GLTFModel, error) {
	doc, err := gltf.Open(path)
	if err != nil {
		return GLTFModel{}, fmt.Errorf("gltf.Open: %w", err)
	}
	if len(doc.Scenes) == 0 {
		return GLTFModel{}, fmt.Errorf("gltf model has no scenes")
	}

	activeScene := 0
	if doc.Scene != nil {
		activeScene = *doc.Scene
	}
	if activeScene < 0 || activeScene >= len(doc.Scenes) {
		return GLTFModel{}, fmt.Errorf("gltf scene index %d out of range", activeScene)
	}

	textures, err := LoadGLTFTextures(dev, gpu, queue, cmdCtx, doc, filepath.Dir(path))
	if err != nil {
		return GLTFModel{}, err
	}

	var prims []GLTFPrimitive
	var visitNode func(nodeIndex int, parentTransform [16]float32) error
	visitNode = func(nodeIndex int, parentTransform [16]float32) error {
		node := doc.Nodes[nodeIndex]
		worldTransform := multiplyMat4(parentTransform, gltfNodeTransform(node))

		if node.Mesh != nil {
			meshIndex := *node.Mesh
			mesh := doc.Meshes[meshIndex]
			for pi, prim := range mesh.Primitives {
				positions, err := modeler.ReadPosition(doc, doc.Accessors[prim.Attributes[gltf.POSITION]], nil)
				if err != nil {
					return fmt.Errorf("node %d mesh %d prim %d ReadPosition: %w", nodeIndex, meshIndex, pi, err)
				}
				normals, err := modeler.ReadNormal(doc, doc.Accessors[prim.Attributes[gltf.NORMAL]], nil)
				if err != nil {
					return fmt.Errorf("node %d mesh %d prim %d ReadNormal: %w", nodeIndex, meshIndex, pi, err)
				}
				uvs, err := modeler.ReadTextureCoord(doc, doc.Accessors[prim.Attributes[gltf.TEXCOORD_0]], nil)
				if err != nil {
					return fmt.Errorf("node %d mesh %d prim %d ReadTextureCoord: %w", nodeIndex, meshIndex, pi, err)
				}
				indices, err := modeler.ReadIndices(doc, doc.Accessors[*prim.Indices], nil)
				if err != nil {
					return fmt.Errorf("node %d mesh %d prim %d ReadIndices: %w", nodeIndex, meshIndex, pi, err)
				}

				vertices := make([]float32, 0, len(positions)*8)
				for i := range positions {
					nx, ny, nz := transformNormal(worldTransform, normals[i][0], normals[i][1], normals[i][2])
					vertices = append(vertices,
						positions[i][0], positions[i][1], positions[i][2],
						nx, ny, nz,
						uvs[i][0], uvs[i][1],
					)
				}

				rtUsage := vk.BufferUsageFlags(vk.BufferUsageShaderDeviceAddressBit | vk.BufferUsageAccelerationStructureBuildInputReadOnlyBit | vk.BufferUsageStorageBufferBit)
				vertexBuf, err := NewBufferHostVisible(dev, gpu, vertices, true, rtUsage)
				if err != nil {
					destroyGLTFPrimitives(prims)
					destroyImageResources(textures)
					return fmt.Errorf("create vertex buffer for node %d mesh %d prim %d: %w", nodeIndex, meshIndex, pi, err)
				}
				indexBuf, err := NewBufferHostVisible(dev, gpu, indices, true, rtUsage)
				if err != nil {
					vertexBuf.Destroy()
					destroyGLTFPrimitives(prims)
					destroyImageResources(textures)
					return fmt.Errorf("create index buffer for node %d mesh %d prim %d: %w", nodeIndex, meshIndex, pi, err)
				}

				baseColorTex := int32(0)
				occlusionTex := int32(-1)
				if prim.Material != nil && *prim.Material >= 0 && *prim.Material < len(doc.Materials) {
					material := doc.Materials[*prim.Material]
					if material != nil {
						if material.PBRMetallicRoughness != nil && material.PBRMetallicRoughness.BaseColorTexture != nil {
							baseColorTex = int32(material.PBRMetallicRoughness.BaseColorTexture.Index + 1)
						}
						if material.OcclusionTexture != nil && material.OcclusionTexture.Index != nil {
							occlusionTex = int32(*material.OcclusionTexture.Index + 1)
						}
					}
				}

				prims = append(prims, GLTFPrimitive{
					VertexBuffer:  vertexBuf,
					IndexBuffer:   indexBuf,
					VertexCount:   uint32(len(positions)),
					TriangleCount: uint32(len(indices) / 3),
					Transform:     vkTransformMatrix(worldTransform),
					BaseColorTex:  baseColorTex,
					OcclusionTex:  occlusionTex,
				})
			}
		}

		for _, childIndex := range node.Children {
			if err := visitNode(childIndex, worldTransform); err != nil {
				return err
			}
		}
		return nil
	}

	for _, rootNode := range doc.Scenes[activeScene].Nodes {
		if err := visitNode(rootNode, identityMat4()); err != nil {
			destroyGLTFPrimitives(prims)
			destroyImageResources(textures)
			return GLTFModel{}, err
		}
	}

	if len(prims) == 0 {
		destroyImageResources(textures)
		return GLTFModel{}, fmt.Errorf("gltf model has no primitives")
	}

	blas, err := buildGLTFModelBLAS(dev, gpu, queue, cmdCtx, prims)
	if err != nil {
		destroyGLTFPrimitives(prims)
		destroyImageResources(textures)
		return GLTFModel{}, err
	}

	geometryBuf, err := createGLTFGeometryNodesBuffer(dev, gpu, prims)
	if err != nil {
		blas.Destroy()
		destroyGLTFPrimitives(prims)
		destroyImageResources(textures)
		return GLTFModel{}, err
	}

	return NewGLTFModel(dev, prims, geometryBuf, blas, textures), nil
}

func destroyGLTFPrimitives(prims []GLTFPrimitive) {
	for i := range prims {
		prims[i].IndexBuffer.Destroy()
		prims[i].VertexBuffer.Destroy()
	}
}

func createGLTFGeometryNodesBuffer(dev vk.Device, gpu vk.PhysicalDevice, prims []GLTFPrimitive) (VulkanBufferResource, error) {
	nodes := make([]gltfGeometryNode, len(prims))
	for i := range prims {
		nodes[i] = gltfGeometryNode{
			VertexBufferDeviceAddress: uint64(prims[i].VertexBuffer.DeviceAddress),
			IndexBufferDeviceAddress:  uint64(prims[i].IndexBuffer.DeviceAddress),
			TextureIndexBaseColor:     prims[i].BaseColorTex,
			TextureIndexOcclusion:     prims[i].OcclusionTex,
		}
	}
	buf, err := NewBufferHostVisible(dev, gpu, nodes, true, vk.BufferUsageFlags(vk.BufferUsageStorageBufferBit|vk.BufferUsageShaderDeviceAddressBit))
	if err != nil {
		return VulkanBufferResource{}, fmt.Errorf("create geometry nodes buffer: %w", err)
	}
	return buf, nil
}

func buildGLTFModelBLAS(dev vk.Device, gpu vk.PhysicalDevice, queue vk.Queue, cmdCtx *VulkanCommandContext, prims []GLTFPrimitive) (VulkanAccelerationStructure, error) {
	geometries := make([]vk.AccelerationStructureGeometry, 0, len(prims))
	primitiveCounts := make([]uint32, 0, len(prims))
	rangeInfos := make([]vk.AccelerationStructureBuildRangeInfo, 0, len(prims))
	transformMatrices := make([][12]float32, len(prims))

	for i := range prims {
		transformMatrices[i] = prims[i].Transform
	}

	transformBuf, err := NewBufferHostVisible(dev, gpu, transformMatrices, true,
		vk.BufferUsageFlags(vk.BufferUsageShaderDeviceAddressBit|vk.BufferUsageAccelerationStructureBuildInputReadOnlyBit))
	if err != nil {
		return VulkanAccelerationStructure{}, fmt.Errorf("create transform buffer: %w", err)
	}
	defer transformBuf.Destroy()

	transformAddr := transformBuf.DeviceAddress
	transformStride := vk.DeviceAddress(unsafe.Sizeof(transformMatrices[0]))

	for i := range prims {
		vertexAddr := prims[i].VertexBuffer.DeviceAddress
		indexAddr := prims[i].IndexBuffer.DeviceAddress

		var trianglesData vk.AccelerationStructureGeometryTrianglesData
		trianglesData.SType = vk.StructureTypeAccelerationStructureGeometryTrianglesData
		trianglesData.VertexFormat = vk.FormatR32g32b32Sfloat
		setDeviceAddressConstRT(&trianglesData.VertexData, vertexAddr)
		trianglesData.VertexStride = 32
		trianglesData.MaxVertex = prims[i].VertexCount - 1
		trianglesData.IndexType = vk.IndexTypeUint32
		setDeviceAddressConstRT(&trianglesData.IndexData, indexAddr)
		setDeviceAddressConstRT(&trianglesData.TransformData, transformAddr+vk.DeviceAddress(i)*transformStride)

		var geometry vk.AccelerationStructureGeometry
		geometry.SType = vk.StructureTypeAccelerationStructureGeometry
		geometry.GeometryType = vk.GeometryTypeTriangles
		geometry.Flags = vk.GeometryFlags(vk.GeometryOpaqueBit)
		setGeometryTrianglesRT(&geometry.Geometry, &trianglesData)

		geometries = append(geometries, geometry)
		primitiveCounts = append(primitiveCounts, prims[i].TriangleCount)
		rangeInfos = append(rangeInfos, vk.AccelerationStructureBuildRangeInfo{
			PrimitiveCount: prims[i].TriangleCount,
		})
	}

	buildInfo := vk.AccelerationStructureBuildGeometryInfo{
		SType:         vk.StructureTypeAccelerationStructureBuildGeometryInfo,
		Type:          vk.AccelerationStructureTypeBottomLevel,
		Flags:         vk.BuildAccelerationStructureFlags(vk.BuildAccelerationStructurePreferFastTraceBit),
		GeometryCount: uint32(len(geometries)),
		PGeometries:   geometries,
	}

	var sizeInfo vk.AccelerationStructureBuildSizesInfo
	sizeInfo.SType = vk.StructureTypeAccelerationStructureBuildSizesInfo
	vk.GetAccelerationStructureBuildSizes(dev, vk.AccelerationStructureBuildTypeDevice, &buildInfo, &primitiveCounts[0], &sizeInfo)
	sizeInfo.Deref()

	asBuf, err := NewBufferDeviceLocal(dev, gpu, uint64(sizeInfo.AccelerationStructureSize), true,
		vk.BufferUsageFlags(vk.BufferUsageAccelerationStructureStorageBit|vk.BufferUsageShaderDeviceAddressBit))
	if err != nil {
		return VulkanAccelerationStructure{}, fmt.Errorf("create BLAS buffer: %w", err)
	}

	var as vk.AccelerationStructure
	if err := vk.Error(vk.CreateAccelerationStructure(dev, &vk.AccelerationStructureCreateInfo{
		SType:  vk.StructureTypeAccelerationStructureCreateInfo,
		Buffer: asBuf.Buffer,
		Size:   sizeInfo.AccelerationStructureSize,
		Type:   vk.AccelerationStructureTypeBottomLevel,
	}, nil, &as)); err != nil {
		asBuf.Destroy()
		return VulkanAccelerationStructure{}, fmt.Errorf("CreateAccelerationStructure (BLAS): %w", err)
	}

	scratchBuf, err := NewBufferDeviceLocal(dev, gpu, uint64(sizeInfo.BuildScratchSize), true,
		vk.BufferUsageFlags(vk.BufferUsageStorageBufferBit|vk.BufferUsageShaderDeviceAddressBit))
	if err != nil {
		vk.DestroyAccelerationStructure(dev, as, nil)
		asBuf.Destroy()
		return VulkanAccelerationStructure{}, fmt.Errorf("create BLAS scratch buffer: %w", err)
	}
	defer scratchBuf.Destroy()

	buildInfo2 := vk.AccelerationStructureBuildGeometryInfo{
		SType:                    vk.StructureTypeAccelerationStructureBuildGeometryInfo,
		Type:                     vk.AccelerationStructureTypeBottomLevel,
		Flags:                    vk.BuildAccelerationStructureFlags(vk.BuildAccelerationStructurePreferFastTraceBit),
		Mode:                     vk.BuildAccelerationStructureModeBuild,
		DstAccelerationStructure: as,
		GeometryCount:            uint32(len(geometries)),
		PGeometries:              geometries,
	}
	setDeviceAddressRT(&buildInfo2.ScratchData, scratchBuf.DeviceAddress)

	cmd, err := cmdCtx.BeginOneTime()
	if err != nil {
		vk.DestroyAccelerationStructure(dev, as, nil)
		asBuf.Destroy()
		return VulkanAccelerationStructure{}, fmt.Errorf("BeginOneTime: %w", err)
	}
	vk.CmdBuildAccelerationStructures(cmd, 1, &buildInfo2, [][]vk.AccelerationStructureBuildRangeInfo{rangeInfos})
	if err := cmdCtx.EndOneTime(queue, cmd); err != nil {
		vk.DestroyAccelerationStructure(dev, as, nil)
		asBuf.Destroy()
		return VulkanAccelerationStructure{}, fmt.Errorf("EndOneTime: %w", err)
	}

	return VulkanAccelerationStructure{
		device:                dev,
		AccelerationStructure: as,
		Buffer:                asBuf,
		Type:                  vk.AccelerationStructureTypeBottomLevel,
	}, nil
}

func identityMat4() [16]float32 {
	return [16]float32{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
}

func multiplyMat4(a, b [16]float32) [16]float32 {
	var out [16]float32
	for col := 0; col < 4; col++ {
		for row := 0; row < 4; row++ {
			var sum float32
			for k := 0; k < 4; k++ {
				sum += a[k*4+row] * b[col*4+k]
			}
			out[col*4+row] = sum
		}
	}
	return out
}

func gltfNodeTransform(node *gltf.Node) [16]float32 {
	if node.Matrix != gltf.DefaultMatrix && node.Matrix != [16]float64{} {
		return gltfMatrixToArray(node.Matrix)
	}

	translation := node.TranslationOrDefault()
	rotation := node.RotationOrDefault()
	scale := node.ScaleOrDefault()

	var t Mat4x4
	t.Translate(float32(translation[0]), float32(translation[1]), float32(translation[2]))

	var r Mat4x4
	r.FromQuat(&Quat{
		float32(rotation[0]),
		float32(rotation[1]),
		float32(rotation[2]),
		float32(rotation[3]),
	})

	var rs Mat4x4
	rs.ScaleAniso(&r, float32(scale[0]), float32(scale[1]), float32(scale[2]))

	var trs Mat4x4
	trs.Mult(&t, &rs)
	return mat4ToArray(&trs)
}

func transformNormal(m [16]float32, nx, ny, nz float32) (float32, float32, float32) {
	ox := m[0]*nx + m[4]*ny + m[8]*nz
	oy := m[1]*nx + m[5]*ny + m[9]*nz
	oz := m[2]*nx + m[6]*ny + m[10]*nz
	l := float32(math.Sqrt(float64(ox*ox + oy*oy + oz*oz)))
	if l > 0 {
		ox /= l
		oy /= l
		oz /= l
	}
	return ox, oy, oz
}

func vkTransformMatrix(m [16]float32) [12]float32 {
	return [12]float32{
		m[0], m[4], m[8], m[12],
		m[1], m[5], m[9], m[13],
		m[2], m[6], m[10], m[14],
	}
}

func gltfMatrixToArray(m [16]float64) [16]float32 {
	var out [16]float32
	for i := range out {
		out[i] = float32(m[i])
	}
	return out
}

func mat4ToArray(m *Mat4x4) [16]float32 {
	var out [16]float32
	for col := 0; col < 4; col++ {
		for row := 0; row < 4; row++ {
			out[col*4+row] = m[col][row]
		}
	}
	return out
}

func setDeviceAddressConstRT(addr *vk.DeviceOrHostAddressConst, da vk.DeviceAddress) {
	*(*vk.DeviceAddress)(unsafe.Pointer(&addr[0])) = da
}

func setDeviceAddressRT(addr *vk.DeviceOrHostAddress, da vk.DeviceAddress) {
	*(*vk.DeviceAddress)(unsafe.Pointer(&addr[0])) = da
}

func setGeometryTrianglesRT(data *vk.AccelerationStructureGeometryData, tri *vk.AccelerationStructureGeometryTrianglesData) {
	cTri, _ := tri.PassRef()
	src := unsafe.Slice((*byte)(unsafe.Pointer(cTri)), len(*data))
	copy((*data)[:], src)
}
