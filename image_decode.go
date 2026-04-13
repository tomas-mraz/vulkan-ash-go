package ash

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
)

// DecodePNG decodes a PNG image from raw bytes and returns RGBA pixels
// suitable for NewImageTexture.
func DecodePNG(data []byte) (pixels []byte, width, height uint32, err error) {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("png.Decode: %w", err)
	}
	rgba := image.NewRGBA(img.Bounds())
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{}, draw.Src)
	return rgba.Pix, uint32(rgba.Bounds().Dx()), uint32(rgba.Bounds().Dy()), nil
}
