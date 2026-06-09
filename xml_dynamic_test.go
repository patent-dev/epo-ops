package epo_ops

import (
	"encoding/xml"
	"testing"
)

// TestParseLegal_DynamicFields tests that legal event field extraction
// dynamically captures all L*EP fields using reflection, not just the
// hardcoded L001EP-L010EP range.
func TestParseLegal_DynamicFields(t *testing.T) {
	// Test XML with various L*EP fields including those beyond L010EP
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<ops:world-patent-data xmlns:ops="http://ops.epo.org">
  <ops:patent-family>
    <ops:publication-reference>
      <ops:document-id>
        <ops:country>EP</ops:country>
        <ops:doc-number>1000000</ops:doc-number>
        <ops:kind>B1</ops:kind>
      </ops:document-id>
    </ops:publication-reference>
    <ops:family-member family-id="12345">
      <ops:legal code="TEST1" desc="Test Event 1" infl="P">
        <ops:L001EP>Value 1</ops:L001EP>
        <ops:L005EP>Value 5</ops:L005EP>
        <ops:L010EP>Value 10</ops:L010EP>
      </ops:legal>
      <ops:legal code="TEST2" desc="Test Event 2" infl="N">
        <ops:L015EP>Value 15</ops:L015EP>
        <ops:L020EP>Value 20</ops:L020EP>
        <ops:L025EP>Value 25</ops:L025EP>
        <ops:L030EP>Value 30</ops:L030EP>
      </ops:legal>
      <ops:legal code="TEST3" desc="Test Event 3" infl="">
        <ops:L001EP>Val A</ops:L001EP>
        <ops:L010EP>Val B</ops:L010EP>
        <ops:L020EP>Val C</ops:L020EP>
        <ops:L030EP>Val D</ops:L030EP>
        <ops:L040EP>Val E</ops:L040EP>
        <ops:L050EP>Val F</ops:L050EP>
      </ops:legal>
    </ops:family-member>
  </ops:patent-family>
</ops:world-patent-data>`

	data, err := ParseLegal(xmlData)
	if err != nil {
		t.Fatalf("ParseLegal failed: %v", err)
	}

	if len(data.LegalEvents) != 3 {
		t.Fatalf("Expected 3 legal events, got %d", len(data.LegalEvents))
	}

	// Test Event 1: Fields L001EP, L005EP, L010EP
	event1 := data.LegalEvents[0]
	if event1.Code != "TEST1" {
		t.Errorf("Event 1 code: got %q, want %q", event1.Code, "TEST1")
	}

	expectedFields1 := map[string]string{
		"L001EP": "Value 1",
		"L005EP": "Value 5",
		"L010EP": "Value 10",
	}

	for field, expectedValue := range expectedFields1 {
		if value, ok := event1.Fields[field]; !ok {
			t.Errorf("Event 1: Field %s not captured", field)
		} else if value != expectedValue {
			t.Errorf("Event 1: Field %s = %q, want %q", field, value, expectedValue)
		}
	}

	if len(event1.Fields) != len(expectedFields1) {
		t.Errorf("Event 1: Expected %d fields, got %d: %v", len(expectedFields1), len(event1.Fields), event1.Fields)
	}

	t.Logf("Event 1 successfully captured %d L*EP fields: %v", len(event1.Fields), event1.Fields)

	// Test Event 2: Fields beyond L010EP (L015EP, L020EP, L025EP, L030EP)
	event2 := data.LegalEvents[1]
	if event2.Code != "TEST2" {
		t.Errorf("Event 2 code: got %q, want %q", event2.Code, "TEST2")
	}

	expectedFields2 := map[string]string{
		"L015EP": "Value 15",
		"L020EP": "Value 20",
		"L025EP": "Value 25",
		"L030EP": "Value 30",
	}

	for field, expectedValue := range expectedFields2 {
		if value, ok := event2.Fields[field]; !ok {
			t.Errorf("Event 2: Field %s not captured (beyond L010EP range)", field)
		} else if value != expectedValue {
			t.Errorf("Event 2: Field %s = %q, want %q", field, value, expectedValue)
		}
	}

	if len(event2.Fields) != len(expectedFields2) {
		t.Errorf("Event 2: Expected %d fields, got %d: %v", len(expectedFields2), len(event2.Fields), event2.Fields)
	}

	t.Logf("Event 2 successfully captured %d L*EP fields beyond L010EP: %v", len(event2.Fields), event2.Fields)

	// Test Event 3: Wide range of fields (L001EP through L050EP)
	event3 := data.LegalEvents[2]
	if event3.Code != "TEST3" {
		t.Errorf("Event 3 code: got %q, want %q", event3.Code, "TEST3")
	}

	expectedFields3 := map[string]string{
		"L001EP": "Val A",
		"L010EP": "Val B",
		"L020EP": "Val C",
		"L030EP": "Val D",
		"L040EP": "Val E",
		"L050EP": "Val F",
	}

	for field, expectedValue := range expectedFields3 {
		if value, ok := event3.Fields[field]; !ok {
			t.Errorf("Event 3: Field %s not captured", field)
		} else if value != expectedValue {
			t.Errorf("Event 3: Field %s = %q, want %q", field, value, expectedValue)
		}
	}

	if len(event3.Fields) != len(expectedFields3) {
		t.Errorf("Event 3: Expected %d fields, got %d: %v", len(expectedFields3), len(event3.Fields), event3.Fields)
	}

	t.Logf("Event 3 successfully captured %d L*EP fields up to L050EP: %v", len(event3.Fields), event3.Fields)

	t.Logf("\nOK Dynamic field extraction successfully handles L001EP through L050EP")
}

// TestLegalEvent_UnmarshalXML tests the dynamic capture: empty values are skipped, and
// L-codes are captured regardless of range (including codes well beyond the old L050EP cap).
func TestLegalEvent_UnmarshalXML(t *testing.T) {
	tests := []struct {
		name     string
		xml      string
		expected map[string]string
	}{
		{
			name:     "No L fields",
			xml:      `<legal code="TEST" desc="Test"></legal>`,
			expected: map[string]string{},
		},
		{
			name:     "Empty values skipped",
			xml:      `<legal code="T"><L001EP></L001EP><L002EP>  </L002EP></legal>`,
			expected: map[string]string{},
		},
		{
			name: "Mixed empty and non-empty",
			xml:  `<legal code="T"><L001EP>Value</L001EP><L002EP></L002EP><L003EP>Another</L003EP></legal>`,
			expected: map[string]string{
				"L001EP": "Value",
				"L003EP": "Another",
			},
		},
		{
			name: "Codes beyond the old L050EP cap",
			xml:  `<legal code="T"><L001EP>a</L001EP><L520EP>b</L520EP><L525EP>c</L525EP></legal>`,
			expected: map[string]string{
				"L001EP": "a",
				"L520EP": "b",
				"L525EP": "c",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ev legalEventXML
			if err := xml.Unmarshal([]byte(tt.xml), &ev); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if len(ev.Fields) != len(tt.expected) {
				t.Errorf("got %d fields %v, want %d %v", len(ev.Fields), ev.Fields, len(tt.expected), tt.expected)
			}
			for field, want := range tt.expected {
				if got := ev.Fields[field]; got != want {
					t.Errorf("field %s = %q, want %q", field, got, want)
				}
			}
		})
	}
}
