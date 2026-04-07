package ash

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"

	"github.com/qmuntal/gltf"
	"github.com/qmuntal/gltf/modeler"
)

// Model holds parsed glTF model data with interleaved position(3) + normal(3) vertices.
type Model struct {
	Vertices        []float32 // interleaved: pos3 + norm3 = 6 floats per vertex
	FloatsPerVertex int
	Indices         []uint32
}

// VertexCount returns the number of vertices.
func (m *Model) VertexCount() int {
	if m == nil || m.FloatsPerVertex == 0 {
		return 0
	}
	return len(m.Vertices) / m.FloatsPerVertex
}

// IndexCount returns the number of indices.
func (m *Model) IndexCount() int {
	return len(m.Indices)
}

// TexturedModel holds parsed glTF model data with interleaved
// position(3) + normal(3) + texcoord(2) vertices and a base color texture.
type TexturedModel struct {
	Vertices        []float32 // interleaved: pos3 + norm3 + uv2 = 8 floats per vertex
	FloatsPerVertex int
	Indices         []uint32
	TextureRGBA     []byte
	TextureWidth    uint32
	TextureHeight   uint32
}

// VertexCount returns the number of vertices.
func (m *TexturedModel) VertexCount() int {
	return len(m.Vertices) / m.FloatsPerVertex
}

// IndexCount returns the number of indices.
func (m *TexturedModel) IndexCount() int {
	return len(m.Indices)
}

// LoadModel loads a glTF/GLB file and returns a Model with interleaved
// vertex data (pos3+norm3).
func LoadModel(path string) (Model, error) {
	doc, err := gltf.Open(path)
	if err != nil {
		return Model{}, fmt.Errorf("gltf.Open: %w", err)
	}

	vertices, indices, err := loadMeshesUntextured(doc)
	if err != nil {
		return Model{}, err
	}

	return Model{
		Vertices:        vertices,
		FloatsPerVertex: 6,
		Indices:         indices,
	}, nil
}

// LoadGLBModel loads a glTF/GLB file and returns a TexturedModel with interleaved
// vertex data (pos3+norm3+uv2), indices, and the base color texture.
func LoadGLBModel(path string) (TexturedModel, error) {
	doc, err := gltf.Open(path)
	if err != nil {
		return TexturedModel{}, fmt.Errorf("gltf.Open: %w", err)
	}

	vertices, indices, err := loadMeshesTextured(doc)
	if err != nil {
		return TexturedModel{}, err
	}

	rgba, w, h, err := loadBaseColorTexture(doc)
	if err != nil {
		return TexturedModel{}, err
	}

	return TexturedModel{
		Vertices:        vertices,
		FloatsPerVertex: 8,
		Indices:         indices,
		TextureRGBA:     rgba,
		TextureWidth:    w,
		TextureHeight:   h,
	}, nil
}

func loadMeshesUntextured(doc *gltf.Document) (interleaved []float32, indices []uint32, err error) {
	for _, mesh := range doc.Meshes {
		for _, prim := range mesh.Primitives {
			positions, err := modeler.ReadPosition(doc, doc.Accessors[prim.Attributes[gltf.POSITION]], nil)
			if err != nil {
				return nil, nil, fmt.Errorf("ReadPosition: %w", err)
			}
			normals, err := modeler.ReadNormal(doc, doc.Accessors[prim.Attributes[gltf.NORMAL]], nil)
			if err != nil {
				return nil, nil, fmt.Errorf("ReadNormal: %w", err)
			}

			vertexOffset := uint32(len(interleaved) / 6)

			primIndices, err := modeler.ReadIndices(doc, doc.Accessors[*prim.Indices], nil)
			if err != nil {
				return nil, nil, fmt.Errorf("ReadIndices: %w", err)
			}
			for _, idx := range primIndices {
				indices = append(indices, idx+vertexOffset)
			}

			for i := range positions {
				interleaved = append(interleaved,
					positions[i][0], positions[i][1], positions[i][2],
					normals[i][0], normals[i][1], normals[i][2],
				)
			}
		}
	}
	return interleaved, indices, nil
}

func loadMeshesTextured(doc *gltf.Document) (interleaved []float32, indices []uint32, err error) {
	for _, mesh := range doc.Meshes {
		for _, prim := range mesh.Primitives {
			positions, err := modeler.ReadPosition(doc, doc.Accessors[prim.Attributes[gltf.POSITION]], nil)
			if err != nil {
				return nil, nil, fmt.Errorf("ReadPosition: %w", err)
			}
			normals, err := modeler.ReadNormal(doc, doc.Accessors[prim.Attributes[gltf.NORMAL]], nil)
			if err != nil {
				return nil, nil, fmt.Errorf("ReadNormal: %w", err)
			}
			uvs, err := modeler.ReadTextureCoord(doc, doc.Accessors[prim.Attributes[gltf.TEXCOORD_0]], nil)
			if err != nil {
				return nil, nil, fmt.Errorf("ReadTextureCoord: %w", err)
			}

			vertexOffset := uint32(len(interleaved) / 8)

			primIndices, err := modeler.ReadIndices(doc, doc.Accessors[*prim.Indices], nil)
			if err != nil {
				return nil, nil, fmt.Errorf("ReadIndices: %w", err)
			}
			for _, idx := range primIndices {
				indices = append(indices, idx+vertexOffset)
			}

			for i := range positions {
				interleaved = append(interleaved,
					positions[i][0], positions[i][1], positions[i][2],
					normals[i][0], normals[i][1], normals[i][2],
					uvs[i][0], uvs[i][1],
				)
			}
		}
	}
	return interleaved, indices, nil
}

func loadBaseColorTexture(doc *gltf.Document) (rgba []byte, w, h uint32, err error) {
	if len(doc.Materials) == 0 {
		return nil, 0, 0, fmt.Errorf("no materials in model")
	}
	mat := doc.Materials[0]
	pbr := mat.PBRMetallicRoughness
	if pbr == nil || pbr.BaseColorTexture == nil {
		return nil, 0, 0, fmt.Errorf("no baseColorTexture in material")
	}
	texIdx := pbr.BaseColorTexture.Index
	imgIdx := doc.Textures[texIdx].Source
	imgDef := doc.Images[*imgIdx]

	bv := doc.BufferViews[*imgDef.BufferView]
	buf := doc.Buffers[bv.Buffer]
	imgData := buf.Data[bv.ByteOffset : bv.ByteOffset+bv.ByteLength]

	img, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("decode texture image: %w", err)
	}
	rgbaImg := image.NewRGBA(img.Bounds())
	draw.Draw(rgbaImg, rgbaImg.Bounds(), img, image.Point{}, draw.Src)

	return rgbaImg.Pix, uint32(rgbaImg.Bounds().Dx()), uint32(rgbaImg.Bounds().Dy()), nil
}
