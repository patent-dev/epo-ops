package tiffutil

import (
	"bytes"
	"encoding/binary"
	"image"
	"strings"
	"testing"

	"github.com/hhrutter/tiff"
)

// Note: Full TIFF conversion tests require real TIFF images from EPO.
// These tests focus on error handling. Integration tests will validate
// with actual patent images.

// TestEmptyTIFF tests error handling for empty input.
func TestEmptyTIFF(t *testing.T) {
	_, err := TIFFToPNG([]byte{})
	if err == nil {
		t.Error("Expected error for empty TIFF data, got nil")
	}

	_, err = TIFFToPNGNoRotate([]byte{})
	if err == nil {
		t.Error("Expected error for empty TIFF data, got nil")
	}

	_, err = DecodeTIFF([]byte{})
	if err == nil {
		t.Error("Expected error for empty TIFF data, got nil")
	}
}

// TestInvalidTIFF tests error handling for invalid TIFF data.
func TestInvalidTIFF(t *testing.T) {
	invalidData := []byte("not a valid TIFF file")

	_, err := TIFFToPNG(invalidData)
	if err == nil {
		t.Error("Expected error for invalid TIFF data, got nil")
	}
}

// hugeDimsTIFF builds a syntactically valid TIFF header whose IFD claims
// enormous image dimensions without carrying any pixel data, mimicking a
// crafted or corrupt file that would otherwise trigger a huge allocation.
func hugeDimsTIFF(t *testing.T, width, height uint32) []byte {
	t.Helper()

	var buf bytes.Buffer
	le := binary.LittleEndian

	// Header: "II", magic 42, IFD at offset 8.
	buf.WriteString("II")
	_ = binary.Write(&buf, le, uint16(42))
	_ = binary.Write(&buf, le, uint32(8))

	type entry struct {
		tag, typ uint16
		count    uint32
		value    uint32
	}
	entries := []entry{
		{256, 4, 1, width},  // ImageWidth (LONG)
		{257, 4, 1, height}, // ImageLength (LONG)
		{258, 3, 1, 8},      // BitsPerSample (SHORT)
		{262, 3, 1, 1},      // PhotometricInterpretation: BlackIsZero (SHORT)
	}
	_ = binary.Write(&buf, le, uint16(len(entries)))
	for _, e := range entries {
		_ = binary.Write(&buf, le, e.tag)
		_ = binary.Write(&buf, le, e.typ)
		_ = binary.Write(&buf, le, e.count)
		if e.typ == 3 {
			// SHORT values are stored left-justified in the 4-byte field.
			_ = binary.Write(&buf, le, uint16(e.value))
			_ = binary.Write(&buf, le, uint16(0))
		} else {
			_ = binary.Write(&buf, le, e.value)
		}
	}
	_ = binary.Write(&buf, le, uint32(0)) // no next IFD

	return buf.Bytes()
}

// TestOversizedDimensionsRejected verifies that TIFF headers claiming more
// than the pixel cap are rejected before any decode allocation happens.
func TestOversizedDimensionsRejected(t *testing.T) {
	data := hugeDimsTIFF(t, 200000, 200000) // 4e10 pixels, far past the cap

	tests := []struct {
		name string
		call func([]byte) error
	}{
		{"TIFFToPNG", func(b []byte) error { _, err := TIFFToPNG(b); return err }},
		{"TIFFToPNGNoRotate", func(b []byte) error { _, err := TIFFToPNGNoRotate(b); return err }},
		{"DecodeTIFF", func(b []byte) error { _, err := DecodeTIFF(b); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call(data)
			if err == nil {
				t.Fatal("Expected error for oversized TIFF dimensions, got nil")
			}
			if !strings.Contains(err.Error(), "pixel decode limit") {
				t.Errorf("Expected pixel decode limit error, got: %v", err)
			}
		})
	}
}

// TestNormalTIFFStillDecodes verifies the size cap does not break decoding of
// ordinary images: a small encoded TIFF round-trips through all helpers.
func TestNormalTIFFStillDecodes(t *testing.T) {
	src := image.NewGray(image.Rect(0, 0, 20, 10)) // landscape to exercise rotation
	var buf bytes.Buffer
	if err := tiff.Encode(&buf, src, nil); err != nil {
		t.Fatalf("failed to encode test TIFF: %v", err)
	}
	data := buf.Bytes()

	if _, err := DecodeTIFF(data); err != nil {
		t.Errorf("DecodeTIFF failed on a normal image: %v", err)
	}
	if _, err := TIFFToPNGNoRotate(data); err != nil {
		t.Errorf("TIFFToPNGNoRotate failed on a normal image: %v", err)
	}
	png, err := TIFFToPNG(data)
	if err != nil {
		t.Fatalf("TIFFToPNG failed on a normal image: %v", err)
	}
	if len(png) == 0 {
		t.Error("TIFFToPNG returned empty PNG data")
	}
}

// TestBatchTIFFToPNG_ReturnsNilOnError verifies that a conversion failure
// returns a nil slice rather than a partial result.
func TestBatchTIFFToPNG_ReturnsNilOnError(t *testing.T) {
	invalid := []byte("not a valid TIFF file")

	images, err := BatchTIFFToPNG([][]byte{invalid, invalid})
	if err == nil {
		t.Fatal("Expected error for invalid TIFF data, got nil")
	}
	if images != nil {
		t.Errorf("Expected nil slice on error, got %d images", len(images))
	}
}
