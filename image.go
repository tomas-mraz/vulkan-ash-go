// Copyright (c) 2025 Cubyte.online under the AGPL License
// Copyright (c) 2022 Cogent Core. under the BSD-style License
// Copyright (c) 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package asch

import (
	"fmt"
	"image"

	vk "github.com/tomas-mraz/vulkan"
)

// Image represents a vulkan image with an associated ImageView.
// The vulkan Image is in device memory, in an optimized format.
// There can also be an optional host-visible, plain pixel buffer
// which can be a pointer into a larger buffer or owned by the Image.
type Image struct {

	// name of the image -- e.g., same as Value name if used that way -- helpful for debugging -- set to filename if loaded from a file and otherwise empty
	Name string

	// bit flags for image state, for indicating nature of ownership and state
	//Flags ImageFlags

	// format & size of image
	Format ImageFormat

	// vulkan image handle, in device memory
	Image vk.Image `display:"-"`

	// vulkan image view
	View vk.ImageView `display:"-"`

	// memory for image when we allocate it
	Mem vk.DeviceMemory `display:"-"`

	// keep track of device for destroying view
	Dev vk.Device `display:"-"`

	// host memory buffer representation of the image
	//Host HostImage

	// pointer to our GPU
	//GPU *GPU
}

// ConfigGoImage configures the image for storing an image
// of the given size, for images allocated in a shared host buffer.
// (i.e., not Var.TextureOwns).  Image format will be set to default
// unless format is already set.  Layers is number of separate images
// of given size allocated in a texture array.
// Once memory is allocated then SetGoImage can be called in a
// second pass.
func (im *Image) ConfigGoImage(sz image.Point, layers int) {
	if im.Format.Format != vk.FormatR8g8b8a8Srgb {
		im.Format.Defaults()
	}
	im.Format.Size = sz
	if layers <= 0 {
		layers = 1
	}
	im.Format.Layers = layers
}

// SetSize sets the size. If the size is not the same as current,
// and Image owns the Host and / or Image, then those are resized.
// returns true if resized.
func (im *Image) SetSize(size image.Point) bool {
	if im.Format.Size == size {
		return false
	}
	im.Format.Size = size

	//FIXME
	/*
		if im.IsHostOwner() {
			im.AllocHost()
		}
		if im.IsImageOwner() || im.HasFlag(DepthImage) {
			im.AllocImage()
		}
	*/
	return true
}

// ConfigStdView configures a standard 2D image view, for current image,
// format, and device.
func (im *Image) ConfigStdView() {
	im.DestroyView()
	var view vk.ImageView
	viewtyp := vk.ImageViewType2d
	//if !im.HasFlag(DepthImage) && !im.HasFlag(FramebufferImage) {
	//	viewtyp = vk.ImageViewType2dArray
	//}
	ret := vk.CreateImageView(im.Dev, &vk.ImageViewCreateInfo{
		SType:  vk.StructureTypeImageViewCreateInfo,
		Format: im.Format.Format,
		Components: vk.ComponentMapping{ // this is the default anyway
			R: vk.ComponentSwizzleIdentity,
			G: vk.ComponentSwizzleIdentity,
			B: vk.ComponentSwizzleIdentity,
			A: vk.ComponentSwizzleIdentity,
		},
		SubresourceRange: vk.ImageSubresourceRange{
			AspectMask: vk.ImageAspectFlags(vk.ImageAspectColorBit),
			LevelCount: 1,
			LayerCount: uint32(im.Format.Layers),
		},
		ViewType: viewtyp,
		Image:    im.Image,
	}, nil, &view)
	IfPanic(NewError(ret))
	im.View = view
	//FIXME im.SetFlag(true, ImageActive)
}

func (im *Image) ConfigDepthView() {
	//FIXME
}

// DestroyView destroys any existing view
func (im *Image) DestroyView() {
	if im.View == vk.NullImageView {
		return
	}
	//FIXME im.SetFlag(false, ImageActive)
	vk.DestroyImageView(im.Dev, im.View, nil)
	im.View = vk.NullImageView
}

// VulkanStorageImageInfo manages a storage image for ray tracing or compute output.
type VulkanStorageImageInfo struct {
	device vk.Device
	image  vk.Image
	mem    vk.DeviceMemory
	view   vk.ImageView
}

// NewStorageImage creates a device-local storage image and transitions it to General layout.
func NewStorageImage(device vk.Device, gpu vk.PhysicalDevice, queue vk.Queue, cmdPool vk.CommandPool, width, height uint32, format vk.Format) (VulkanStorageImageInfo, error) {
	var s VulkanStorageImageInfo
	s.device = device

	// Create image
	err := vk.Error(vk.CreateImage(device, &vk.ImageCreateInfo{
		SType:       vk.StructureTypeImageCreateInfo,
		ImageType:   vk.ImageType2d,
		Format:      format,
		Extent:      vk.Extent3D{Width: width, Height: height, Depth: 1},
		MipLevels:   1,
		ArrayLayers: 1,
		Samples:     vk.SampleCount1Bit,
		Tiling:      vk.ImageTilingOptimal,
		Usage:       vk.ImageUsageFlags(vk.ImageUsageTransferSrcBit | vk.ImageUsageStorageBit),
	}, nil, &s.image))
	if err != nil {
		return s, fmt.Errorf("vk.CreateImage (storage) failed with %s", err)
	}

	// Allocate device-local memory
	var memReqs vk.MemoryRequirements
	vk.GetImageMemoryRequirements(device, s.image, &memReqs)
	memReqs.Deref()

	memIdx, _ := vk.FindMemoryTypeIndex(gpu, memReqs.MemoryTypeBits, vk.MemoryPropertyDeviceLocalBit)
	err = vk.Error(vk.AllocateMemory(device, &vk.MemoryAllocateInfo{
		SType:           vk.StructureTypeMemoryAllocateInfo,
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memIdx,
	}, nil, &s.mem))
	if err != nil {
		vk.DestroyImage(device, s.image, nil)
		return s, fmt.Errorf("vk.AllocateMemory (storage) failed with %s", err)
	}
	vk.BindImageMemory(device, s.image, s.mem, 0)

	// Create image view
	err = vk.Error(vk.CreateImageView(device, &vk.ImageViewCreateInfo{
		SType:    vk.StructureTypeImageViewCreateInfo,
		Image:    s.image,
		ViewType: vk.ImageViewType2d,
		Format:   format,
		SubresourceRange: vk.ImageSubresourceRange{
			AspectMask: vk.ImageAspectFlags(vk.ImageAspectColorBit),
			LevelCount: 1,
			LayerCount: 1,
		},
	}, nil, &s.view))
	if err != nil {
		vk.FreeMemory(device, s.mem, nil)
		vk.DestroyImage(device, s.image, nil)
		return s, fmt.Errorf("vk.CreateImageView (storage) failed with %s", err)
	}

	// Transition to General layout
	TransitionImageLayout(device, queue, cmdPool, s.image, vk.ImageLayoutUndefined, vk.ImageLayoutGeneral)

	return s, nil
}

func (s *VulkanStorageImageInfo) GetImage() vk.Image {
	return s.image
}

func (s *VulkanStorageImageInfo) GetView() vk.ImageView {
	return s.view
}

func (s *VulkanStorageImageInfo) Destroy() {
	if s == nil {
		return
	}
	vk.DestroyImageView(s.device, s.view, nil)
	vk.FreeMemory(s.device, s.mem, nil)
	vk.DestroyImage(s.device, s.image, nil)
}

