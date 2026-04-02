package ash

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"

	"github.com/qmuntal/gltf"
	vk "github.com/tomas-mraz/vulkan"
)

// DecodeGLTFTexture decodes a glTF image to tightly packed RGBA pixels.
func DecodeGLTFTexture(doc *gltf.Document, baseDir string, imageIndex int) ([]byte, uint32, uint32, error) {
	if imageIndex < 0 || imageIndex >= len(doc.Images) {
		return nil, 0, 0, fmt.Errorf("image index %d out of range", imageIndex)
	}
	imageDef := doc.Images[imageIndex]
	if imageDef == nil {
		return nil, 0, 0, fmt.Errorf("image %d is nil", imageIndex)
	}

	var raw []byte
	var err error
	switch {
	case imageDef.IsEmbeddedResource():
		raw, err = imageDef.MarshalData()
	case imageDef.URI != "":
		raw, err = os.ReadFile(filepath.Join(baseDir, imageDef.URI))
	case imageDef.BufferView != nil:
		return nil, 0, 0, fmt.Errorf("bufferView-backed images are not supported")
	default:
		return nil, 0, 0, fmt.Errorf("image %d has no data source", imageIndex)
	}
	if err != nil {
		return nil, 0, 0, err
	}

	decoded, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, 0, 0, err
	}
	bounds := decoded.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, decoded, bounds.Min, draw.Src)
	return rgba.Pix, uint32(bounds.Dx()), uint32(bounds.Dy()), nil
}

// LoadGLTFTextures loads glTF textures into Device image resources.
// Index 0 always contains a 1x1 white fallback texture.
func LoadGLTFTextures(dev vk.Device, gpu vk.PhysicalDevice, queue vk.Queue, cmdCtx *CommandContext, doc *gltf.Document, baseDir string) ([]VulkanImageResource, error) {
	textures := make([]VulkanImageResource, 0, len(doc.Textures)+1)

	fallback, err := newFallbackGLTFTexture(dev, gpu, queue, cmdCtx)
	if err != nil {
		return nil, fmt.Errorf("create fallback texture: %w", err)
	}
	textures = append(textures, fallback)

	for i, tex := range doc.Textures {
		if tex == nil || tex.Source == nil {
			fallback, err := newFallbackGLTFTexture(dev, gpu, queue, cmdCtx)
			if err != nil {
				destroyImageResources(textures)
				return nil, fmt.Errorf("create fallback texture %d: %w", i, err)
			}
			textures = append(textures, fallback)
			continue
		}

		pixels, width, height, err := DecodeGLTFTexture(doc, baseDir, *tex.Source)
		if err != nil {
			destroyImageResources(textures)
			return nil, fmt.Errorf("decode texture %d: %w", i, err)
		}

		texture, err := NewImageTextureWithSampler(dev, gpu, queue, cmdCtx, width, height, pixels, samplerCreateInfoForGLTFTexture(doc, tex))
		if err != nil {
			destroyImageResources(textures)
			return nil, fmt.Errorf("create texture %d: %w", i, err)
		}
		textures = append(textures, texture)
	}

	return textures, nil
}

func newFallbackGLTFTexture(dev vk.Device, gpu vk.PhysicalDevice, queue vk.Queue, cmdCtx *CommandContext) (VulkanImageResource, error) {
	return NewImageTextureWithSampler(dev, gpu, queue, cmdCtx, 1, 1, []byte{255, 255, 255, 255}, defaultGLTFSamplerCreateInfo())
}

func destroyImageResources(resources []VulkanImageResource) {
	for i := range resources {
		resources[i].Destroy()
	}
}

func defaultGLTFSamplerCreateInfo() vk.SamplerCreateInfo {
	return vk.SamplerCreateInfo{
		SType:        vk.StructureTypeSamplerCreateInfo,
		MagFilter:    vk.FilterLinear,
		MinFilter:    vk.FilterLinear,
		MipmapMode:   vk.SamplerMipmapModeLinear,
		AddressModeU: vk.SamplerAddressModeRepeat,
		AddressModeV: vk.SamplerAddressModeRepeat,
		AddressModeW: vk.SamplerAddressModeRepeat,
		MaxLod:       0,
		BorderColor:  vk.BorderColorIntOpaqueWhite,
	}
}

func samplerCreateInfoForGLTFTexture(doc *gltf.Document, tex *gltf.Texture) vk.SamplerCreateInfo {
	info := defaultGLTFSamplerCreateInfo()
	if tex == nil || tex.Sampler == nil || *tex.Sampler < 0 || *tex.Sampler >= len(doc.Samplers) {
		return info
	}
	sampler := doc.Samplers[*tex.Sampler]
	if sampler == nil {
		return info
	}

	info.MagFilter = magFilterFromGLTF(sampler.MagFilter)
	info.MinFilter, info.MipmapMode = minFilterFromGLTF(sampler.MinFilter)
	info.AddressModeU = addressModeFromGLTF(sampler.WrapS)
	info.AddressModeV = addressModeFromGLTF(sampler.WrapT)
	return info
}

func magFilterFromGLTF(filter gltf.MagFilter) vk.Filter {
	switch filter {
	case gltf.MagNearest:
		return vk.FilterNearest
	default:
		return vk.FilterLinear
	}
}

func minFilterFromGLTF(filter gltf.MinFilter) (vk.Filter, vk.SamplerMipmapMode) {
	switch filter {
	case gltf.MinNearest, gltf.MinNearestMipMapNearest, gltf.MinNearestMipMapLinear:
		return vk.FilterNearest, vk.SamplerMipmapModeNearest
	case gltf.MinLinearMipMapNearest:
		return vk.FilterLinear, vk.SamplerMipmapModeNearest
	default:
		return vk.FilterLinear, vk.SamplerMipmapModeLinear
	}
}

func addressModeFromGLTF(mode gltf.WrappingMode) vk.SamplerAddressMode {
	switch mode {
	case gltf.WrapClampToEdge:
		return vk.SamplerAddressModeClampToEdge
	case gltf.WrapMirroredRepeat:
		return vk.SamplerAddressModeMirroredRepeat
	default:
		return vk.SamplerAddressModeRepeat
	}
}
