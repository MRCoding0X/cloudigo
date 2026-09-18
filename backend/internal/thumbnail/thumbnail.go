// Package thumbnail generates small JPEG preview images for uploaded image
// files. Supports the formats decodable by the Go standard library (JPEG,
// PNG, GIF); anything else (including WebP) is silently skipped — a missing
// thumbnail is a cosmetic gap, not a failed upload.
package thumbnail

import (
	"errors"
	"image"
	"image/jpeg"
	_ "image/png" // register PNG decoder
	"os"

	_ "image/gif" // register GIF decoder

	"golang.org/x/image/draw"
)

const MaxDimension = 320

var ErrUnsupported = errors.New("thumbnail: unsupported or non-image file")

// Generate decodes srcPath as an image, downscales it so its longest side is
// at most MaxDimension, and writes a JPEG thumbnail to dstPath.
func Generate(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	img, _, err := image.Decode(src)
	if err != nil {
		return ErrUnsupported
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w == 0 || h == 0 {
		return ErrUnsupported
	}

	scale := 1.0
	if w > h && w > MaxDimension {
		scale = float64(MaxDimension) / float64(w)
	} else if h >= w && h > MaxDimension {
		scale = float64(MaxDimension) / float64(h)
	}

	dstW, dstH := w, h
	if scale < 1.0 {
		dstW = int(float64(w) * scale)
		dstH = int(float64(h) * scale)
		if dstW < 1 {
			dstW = 1
		}
		if dstH < 1 {
			dstH = 1
		}
	}

	dstImg := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	draw.CatmullRom.Scale(dstImg, dstImg.Bounds(), img, bounds, draw.Over, nil)

	out, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer out.Close()

	return jpeg.Encode(out, dstImg, &jpeg.Options{Quality: 80})
}
