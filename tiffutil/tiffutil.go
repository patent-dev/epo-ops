// Package tiffutil provides utilities for TIFF image handling, specifically
// for converting EPO patent TIFF images to PNG format.
//
// EPO patent images are often in TIFF format with various compressions:
//   - CCITT Group 3/4 (for black and white technical drawings)
//   - LZW compression
//   - CMYK color model (for color images)
//
// This package uses github.com/hhrutter/tiff which provides enhanced TIFF
// support including CMYK color model handling.
package tiffutil

import (
	"bytes"
	"fmt"
	"image"
	"image/png"

	"github.com/disintegration/imaging"
	"github.com/hhrutter/tiff"
)

// maxDecodePixels caps the pixel count (width * height) accepted for decoding.
// A crafted or corrupt TIFF header can claim enormous dimensions and make the
// decoder allocate unbounded memory; 100 megapixels is far beyond any real
// patent scan.
const maxDecodePixels = 100_000_000

// decodeCapped decodes TIFF data after checking the declared dimensions
// against maxDecodePixels, so a hostile header cannot trigger a huge
// allocation. All decoding in this package goes through it.
func decodeCapped(tiffData []byte) (image.Image, error) {
	cfg, err := tiff.DecodeConfig(bytes.NewReader(tiffData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode TIFF: %w", err)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, fmt.Errorf("invalid TIFF dimensions %dx%d", cfg.Width, cfg.Height)
	}
	if int64(cfg.Width)*int64(cfg.Height) > maxDecodePixels {
		return nil, fmt.Errorf("TIFF dimensions %dx%d exceed the %d pixel decode limit",
			cfg.Width, cfg.Height, maxDecodePixels)
	}

	img, err := tiff.Decode(bytes.NewReader(tiffData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode TIFF: %w", err)
	}
	return img, nil
}

// TIFFToPNG converts TIFF image data to PNG format.
//
// This function handles various TIFF formats commonly used in patent images:
//   - CCITT Group 3/4 compression (black and white)
//   - LZW compression
//   - CMYK color model
//   - Uncompressed TIFF
//
// Landscape images (width > height) are automatically rotated 90 degrees
// counterclockwise to portrait orientation, which is standard for patent drawings.
//
// Returns the PNG image data as bytes.
func TIFFToPNG(tiffData []byte) ([]byte, error) {
	if len(tiffData) == 0 {
		return nil, fmt.Errorf("empty TIFF data")
	}

	// Decode TIFF using hhrutter/tiff (supports CMYK, CCITT, LZW)
	img, err := decodeCapped(tiffData)
	if err != nil {
		return nil, err
	}

	// Rotate if landscape orientation (width > height)
	// Patent drawings are typically portrait-oriented
	bounds := img.Bounds()
	if bounds.Dx() > bounds.Dy() {
		// Rotate 90 degrees counterclockwise
		img = imaging.Rotate90(img)
	}

	// Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}

	return buf.Bytes(), nil
}

// TIFFToPNGNoRotate converts TIFF to PNG without automatic rotation.
//
// Use this function when you want to preserve the original orientation
// of the image without automatic landscape-to-portrait conversion.
func TIFFToPNGNoRotate(tiffData []byte) ([]byte, error) {
	if len(tiffData) == 0 {
		return nil, fmt.Errorf("empty TIFF data")
	}

	// Decode TIFF
	img, err := decodeCapped(tiffData)
	if err != nil {
		return nil, err
	}

	// Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}

	return buf.Bytes(), nil
}

// DecodeTIFF decodes TIFF data and returns the image.
//
// This is useful if you need to perform additional image processing
// beyond simple PNG conversion.
func DecodeTIFF(tiffData []byte) (image.Image, error) {
	if len(tiffData) == 0 {
		return nil, fmt.Errorf("empty TIFF data")
	}

	img, err := decodeCapped(tiffData)
	if err != nil {
		return nil, err
	}

	return img, nil
}

// BatchTIFFToPNG converts multiple TIFF images to PNG format.
//
// This is useful for converting all pages of a multi-page patent document.
// Returns a slice of PNG byte slices, one for each input TIFF.
//
// If an error occurs during conversion of any image, processing stops and the
// error is returned with a nil slice (no partial results).
func BatchTIFFToPNG(tiffImages [][]byte) ([][]byte, error) {
	pngImages := make([][]byte, 0, len(tiffImages))

	for i, tiffData := range tiffImages {
		pngData, err := TIFFToPNG(tiffData)
		if err != nil {
			return nil, fmt.Errorf("failed to convert image %d: %w", i+1, err)
		}
		pngImages = append(pngImages, pngData)
	}

	return pngImages, nil
}
