package tiffutil

import (
	"testing"
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
