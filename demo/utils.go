package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileFormat represents the format of a file
type FileFormat string

const (
	FormatXML   FileFormat = "xml"
	FormatJSON  FileFormat = "json"
	FormatTIFF  FileFormat = "tiff"
	FormatPNG   FileFormat = "png"
	FormatGIF   FileFormat = "gif"
	FormatJPEG  FileFormat = "jpeg"
	FormatText  FileFormat = "txt"
	FormatBinary FileFormat = "bin"
)

// ExampleSaver saves request/response pairs to disk
type ExampleSaver struct {
	baseDir string
}

// NewExampleSaver creates a new ExampleSaver with the specified base directory
func NewExampleSaver(baseDir string) *ExampleSaver {
	return &ExampleSaver{baseDir: baseDir}
}

// SaveExample saves a request description and response data to the examples directory
func (s *ExampleSaver) SaveExample(endpointName string, requestDesc string, response []byte, format FileFormat) error {
	// Create endpoint directory
	dir := filepath.Join(s.baseDir, endpointName)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Save request description
	requestFile := filepath.Join(dir, "request.txt")
	if err := os.WriteFile(requestFile, []byte(requestDesc), 0600); err != nil {
		return fmt.Errorf("failed to save request: %w", err)
	}

	// Determine response filename based on format
	responseFile := filepath.Join(dir, fmt.Sprintf("response.%s", format))

	// Save response
	if err := os.WriteFile(responseFile, response, 0600); err != nil {
		return fmt.Errorf("failed to save response: %w", err)
	}

	return nil
}

// DetectFormat detects the format of data
func DetectFormat(data []byte) FileFormat {
	if len(data) < 4 {
		return FormatBinary
	}

	// Check for XML
	if strings.HasPrefix(string(data), "<?xml") || strings.HasPrefix(string(data), "<") {
		return FormatXML
	}

	// Check for JSON
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		if json.Valid(data) {
			return FormatJSON
		}
	}

	// Check for TIFF (little-endian or big-endian)
	if (data[0] == 'I' && data[1] == 'I' && data[2] == 42 && data[3] == 0) ||
		(data[0] == 'M' && data[1] == 'M' && data[2] == 0 && data[3] == 42) {
		return FormatTIFF
	}

	// Check for PNG
	if data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G' {
		return FormatPNG
	}

	// Check for JPEG
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return FormatJPEG
	}

	// Check for GIF
	if len(data) >= 6 && string(data[0:3]) == "GIF" {
		return FormatGIF
	}

	return FormatBinary
}

// FormatRequestDescription formats a request description with parameters
func FormatRequestDescription(method string, params map[string]string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Method: %s\n\n", method))
	sb.WriteString("Parameters:\n")
	for k, v := range params {
		sb.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
	}
	return sb.String()
}

// ValidateXML performs basic XML validation
func ValidateXML(data []byte) error {
	str := string(data)
	if !strings.Contains(str, "<") || !strings.Contains(str, ">") {
		return fmt.Errorf("data does not appear to be XML")
	}
	return nil
}

// ValidateJSON performs basic JSON validation
func ValidateJSON(data []byte) error {
	if !json.Valid(data) {
		return fmt.Errorf("invalid JSON data")
	}
	return nil
}

// ValidateTIFF performs basic TIFF validation
func ValidateTIFF(data []byte) error {
	if len(data) < 4 {
		return fmt.Errorf("data too short to be TIFF")
	}

	// Check TIFF magic numbers
	if !((data[0] == 'I' && data[1] == 'I' && data[2] == 42 && data[3] == 0) ||
		(data[0] == 'M' && data[1] == 'M' && data[2] == 0 && data[3] == 42)) {
		return fmt.Errorf("invalid TIFF magic number")
	}

	return nil
}
