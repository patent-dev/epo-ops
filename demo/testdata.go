package main

import (
	_ "embed"
	"fmt"
	"regexp"
	"strings"
)

//go:embed test_patents.txt
var testPatentsData string

var nonAlphanumeric = regexp.MustCompile(`[^a-zA-Z0-9]`)

// SanitizePatentNumber removes all non-alphanumeric characters
func SanitizePatentNumber(s string) string {
	return nonAlphanumeric.ReplaceAllString(s, "")
}

// LoadTestPatents loads real patent numbers from embedded test_patents.txt
func LoadTestPatents() []string {
	var patents []string
	lines := strings.Split(strings.TrimSpace(testPatentsData), "\n")
	for _, line := range lines {
		patent := strings.TrimSpace(line)
		if patent != "" {
			patents = append(patents, patent)
		}
	}
	fmt.Printf("  Loaded %d test patents from embedded test_patents.txt\n", len(patents))
	return patents
}

// GetBulkTestPatents returns a slice of 2-3 patents for bulk operations
func GetBulkTestPatents(mainPatent string) []string {
	allPatents := LoadTestPatents()

	// Include main patent + 2 others
	result := []string{mainPatent}
	added := 0
	for _, p := range allPatents {
		if p != mainPatent && added < 2 {
			result = append(result, p)
			added++
		}
	}

	return result
}
