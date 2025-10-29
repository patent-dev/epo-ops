package epo_ops

import (
	"os"
	"testing"
)

func TestParseFamily(t *testing.T) {
	xmlData, err := os.ReadFile("testdata/family.xml")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	data, err := ParseFamily(string(xmlData))
	if err != nil {
		t.Fatalf("ParseFamily failed: %v", err)
	}

	if len(data.Members) == 0 {
		t.Error("Expected family members, got none")
	}

	t.Logf("Parsed %d family members", len(data.Members))
	for i, member := range data.Members {
		t.Logf("Member %d: %s%s (kind %s, date %s)", i+1, member.Country, member.DocNumber, member.Kind, member.Date)
	}
}

func TestParseLegal(t *testing.T) {
	xmlData, err := os.ReadFile("demo/examples/get_legal/response.xml")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	data, err := ParseLegal(string(xmlData))
	if err != nil {
		t.Fatalf("ParseLegal failed: %v", err)
	}

	if len(data.LegalEvents) == 0 {
		t.Error("Expected legal events, got none")
	}

	t.Logf("Parsed %d legal events for patent %s", len(data.LegalEvents), data.PatentNumber)
	for i, event := range data.LegalEvents {
		if i < 5 { // Log first 5 events
			t.Logf("Event %d: Code=%s, Desc=%s, Influence=%s", i+1, event.Code, event.Description, event.Influence)
		}
	}
}

func TestParseDescription(t *testing.T) {
	xmlData, err := os.ReadFile("testdata/description.xml")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	data, err := ParseDescription(string(xmlData))
	if err != nil {
		t.Fatalf("ParseDescription failed: %v", err)
	}

	if len(data.Paragraphs) == 0 {
		t.Error("Expected paragraphs, got none")
	}

	t.Logf("Parsed %d paragraphs for patent %s (language: %s)", len(data.Paragraphs), data.PatentNumber, data.Language)
	for i, para := range data.Paragraphs {
		if i < 3 { // Log first 3 paragraphs
			maxLen := 50
			if len(para.Text) < maxLen {
				maxLen = len(para.Text)
			}
			t.Logf("Paragraph %d (ID=%s): %s", i+1, para.ID, para.Text[:maxLen]+"...")
		}
	}
}

func TestParseSearch(t *testing.T) {
	xmlData, err := os.ReadFile("testdata/search.xml")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	data, err := ParseSearch(string(xmlData))
	if err != nil {
		t.Fatalf("ParseSearch failed: %v", err)
	}

	if data.TotalCount == 0 {
		t.Error("Expected total count > 0")
	}

	t.Logf("Search results: %d total, range %d-%d, found %d results",
		data.TotalCount, data.RangeBegin, data.RangeEnd, len(data.Results))
	t.Logf("Query: %s", data.Query)

	for i, result := range data.Results {
		t.Logf("Result %d: %s%s (Family %s)", i+1, result.Country, result.DocNumber, result.FamilyID)
	}
}

func TestParseEquivalents(t *testing.T) {
	xmlData, err := os.ReadFile("demo/examples/get_published_equivalents/response.xml")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	data, err := ParseEquivalents(string(xmlData))
	if err != nil {
		t.Fatalf("ParseEquivalents failed: %v", err)
	}

	if len(data.Equivalents) == 0 {
		t.Error("Expected equivalents, got none")
	}

	t.Logf("Found %d equivalents for patent %s", len(data.Equivalents), data.PatentNumber)
	for i, equiv := range data.Equivalents {
		t.Logf("Equivalent %d: %s%s", i+1, equiv.Country, equiv.DocNumber)
	}
}
