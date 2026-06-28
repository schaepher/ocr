// Package imageutil provides image processing utilities for OCR.
package imageutil

import (
	"bytes"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/jpeg"
	_ "image/png"
	"os"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// GetDimensions returns the width and height of an image file.
func GetDimensions(path string) (int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}

// Slice represents one horizontal strip of a sliced image.
type Slice struct {
	Data   []byte // JPEG-encoded bytes
	Y      int    // Y offset of this slice in the original image
	Height int    // height of this slice (before encoding)
}

// SliceImage splits an image into overlapping horizontal strips.
// maxHeight is the maximum height of each slice in pixels.
// overlap is the vertical overlap between consecutive slices in pixels.
func SliceImage(path string, maxHeight, overlap int) ([]Slice, int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, 0, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, 0, 0, err
	}

	bounds := img.Bounds()
	imgW, imgH := bounds.Dx(), bounds.Dy()

	if imgH <= maxHeight {
		return nil, imgW, imgH, nil
	}

	var slices []Slice
	y := 0
	for y < imgH {
		h := maxHeight
		if y+h > imgH {
			h = imgH - y
			if h <= overlap {
				break
			}
		}

		// Crop the strip.
		strip := cropImage(img, bounds.Min.X, bounds.Min.Y+y, imgW, h)

		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, strip, &jpeg.Options{Quality: 92}); err != nil {
			return nil, 0, 0, err
		}

		slices = append(slices, Slice{
			Data:   buf.Bytes(),
			Y:      y,
			Height: h,
		})

		y += h - overlap
		if y >= imgH || h <= overlap {
			break
		}
	}

	return slices, imgW, imgH, nil
}

// cropImage extracts a sub-image from img.
func cropImage(img image.Image, x, y, w, h int) image.Image {
	switch src := img.(type) {
	case *image.RGBA:
		return src.SubImage(image.Rect(x, y, x+w, y+h))
	case *image.NRGBA:
		return src.SubImage(image.Rect(x, y, x+w, y+h))
	case *image.YCbCr:
		return src.SubImage(image.Rect(x, y, x+w, y+h))
	default:
		// Slow path: draw into new RGBA.
		dst := image.NewRGBA(image.Rect(0, 0, w, h))
		for dy := 0; dy < h; dy++ {
			for dx := 0; dx < w; dx++ {
				dst.Set(dx, dy, img.At(x+dx, y+dy))
			}
		}
		return dst
	}
}
