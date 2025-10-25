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
	img, err := tiff.Decode(bytes.NewReader(tiffData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode TIFF: %w", err)
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
	img, err := tiff.Decode(bytes.NewReader(tiffData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode TIFF: %w", err)
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

	img, err := tiff.Decode(bytes.NewReader(tiffData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode TIFF: %w", err)
	}

	return img, nil
}

// BatchTIFFToPNG converts multiple TIFF images to PNG format.
//
// This is useful for converting all pages of a multi-page patent document.
// Returns a slice of PNG byte slices, one for each input TIFF.
//
// If an error occurs during conversion of any image, the error is returned
// and processing stops. Successfully converted images up to that point are returned.
func BatchTIFFToPNG(tiffImages [][]byte) ([][]byte, error) {
	pngImages := make([][]byte, 0, len(tiffImages))

	for i, tiffData := range tiffImages {
		pngData, err := TIFFToPNG(tiffData)
		if err != nil {
			return pngImages, fmt.Errorf("failed to convert image %d: %w", i+1, err)
		}
		pngImages = append(pngImages, pngData)
	}

	return pngImages, nil
}
