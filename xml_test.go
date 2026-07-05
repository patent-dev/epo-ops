package epo_ops

import (
	"embed"
	"testing"
)

//go:embed testdata/*.xml
var xmlTestData embed.FS

func TestParseAbstract(t *testing.T) {
	xmlData, err := xmlTestData.ReadFile("testdata/abstract.xml")
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}

	data, err := ParseAbstract(string(xmlData))
	if err != nil {
		t.Fatalf("ParseAbstract failed: %v", err)
	}

	if data.Country != "EP" {
		t.Errorf("Country: got %q, want %q", data.Country, "EP")
	}
	if data.DocNumber != "2400812" {
		t.Errorf("DocNumber: got %q, want %q", data.DocNumber, "2400812")
	}
	if data.Kind != "A1" {
		t.Errorf("Kind: got %q, want %q", data.Kind, "A1")
	}
	if data.PatentNumber != "EP2400812A1" {
		t.Errorf("PatentNumber: got %q, want %q", data.PatentNumber, "EP2400812A1")
	}
	if data.Language != "en" {
		t.Errorf("Language: got %q, want %q", data.Language, "en")
	}
	if len(data.Text) == 0 {
		t.Error("Text is empty")
	}
	if len(data.Text) < 100 {
		t.Errorf("Text too short: %d chars", len(data.Text))
	}
	t.Logf("Abstract text: %.100s...", data.Text)
}

func TestParseAbstract_LanguageSelection(t *testing.T) {
	const multiLang = `<?xml version="1.0" encoding="UTF-8"?>
<ops:world-patent-data xmlns="http://www.epo.org/exchange" xmlns:ops="http://ops.epo.org">
    <exchange-documents>
        <exchange-document country="EP" doc-number="1000000" kind="B1">
            <abstract lang="de">
                <p>Deutscher Absatz.</p>
            </abstract>
            <abstract lang="en">
                <p>First English paragraph.</p>
                <p>Second English paragraph.</p>
            </abstract>
        </exchange-document>
    </exchange-documents>
</ops:world-patent-data>`

	const noEnglish = `<?xml version="1.0" encoding="UTF-8"?>
<ops:world-patent-data xmlns="http://www.epo.org/exchange" xmlns:ops="http://ops.epo.org">
    <exchange-documents>
        <exchange-document country="EP" doc-number="1000000" kind="B1">
            <abstract lang="de">
                <p>Erster Absatz.</p>
                <p>Zweiter Absatz.</p>
            </abstract>
            <abstract lang="fr">
                <p>Paragraphe francais.</p>
            </abstract>
        </exchange-document>
    </exchange-documents>
</ops:world-patent-data>`

	tests := []struct {
		name     string
		xml      string
		wantLang string
		wantText string
	}{
		{
			name:     "prefers English over earlier language",
			xml:      multiLang,
			wantLang: "en",
			wantText: "First English paragraph.\nSecond English paragraph.",
		},
		{
			name:     "falls back to first abstract in document order",
			xml:      noEnglish,
			wantLang: "de",
			wantText: "Erster Absatz.\nZweiter Absatz.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := ParseAbstract(tt.xml)
			if err != nil {
				t.Fatalf("ParseAbstract failed: %v", err)
			}
			if data.Language != tt.wantLang {
				t.Errorf("Language: got %q, want %q", data.Language, tt.wantLang)
			}
			if data.Text != tt.wantText {
				t.Errorf("Text: got %q, want %q", data.Text, tt.wantText)
			}
		})
	}
}

func TestParseBiblio(t *testing.T) {
	xmlData, err := xmlTestData.ReadFile("testdata/biblio.xml")
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}

	data, err := ParseBiblio(string(xmlData))
	if err != nil {
		t.Fatalf("ParseBiblio failed: %v", err)
	}

	if data.Country != "EP" {
		t.Errorf("Country: got %q, want %q", data.Country, "EP")
	}
	if data.DocNumber != "2400812" {
		t.Errorf("DocNumber: got %q, want %q", data.DocNumber, "2400812")
	}
	if data.Kind != "A1" {
		t.Errorf("Kind: got %q, want %q", data.Kind, "A1")
	}
	if data.PatentNumber != "EP2400812A1" {
		t.Errorf("PatentNumber: got %q, want %q", data.PatentNumber, "EP2400812A1")
	}
	if data.PublicationDate != "20111228" {
		t.Errorf("PublicationDate: got %q, want %q", data.PublicationDate, "20111228")
	}
	if data.FamilyID != "43088294" {
		t.Errorf("FamilyID: got %q, want %q", data.FamilyID, "43088294")
	}
	if len(data.Titles) == 0 {
		t.Error("No titles found")
	}
	if len(data.Applicants) == 0 {
		t.Error("No applicants found")
	}
	if len(data.IPCClasses) == 0 {
		t.Error("No IPC classes found")
	}
	if len(data.CPCClasses) == 0 {
		t.Error("No CPC classes found")
	}

	t.Logf("Titles: %v", data.Titles)
	t.Logf("Applicants: %v", data.Applicants)
	t.Logf("IPC: %v", data.IPCClasses)
	t.Logf("CPC: %d classes", len(data.CPCClasses))
}

func TestParseClaims(t *testing.T) {
	xmlData, err := xmlTestData.ReadFile("testdata/claims.xml")
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}

	data, err := ParseClaims(string(xmlData))
	if err != nil {
		t.Fatalf("ParseClaims failed: %v", err)
	}

	if data.Country != "EP" {
		t.Errorf("Country: got %q, want %q", data.Country, "EP")
	}
	if data.DocNumber != "2400812" {
		t.Errorf("DocNumber: got %q, want %q", data.DocNumber, "2400812")
	}
	if data.Kind != "A1" {
		t.Errorf("Kind: got %q, want %q", data.Kind, "A1")
	}
	if data.PatentNumber != "EP2400812A1" {
		t.Errorf("PatentNumber: got %q, want %q", data.PatentNumber, "EP2400812A1")
	}
	if data.Language != "EN" {
		t.Errorf("Language: got %q, want %q", data.Language, "EN")
	}
	if len(data.Claims) == 0 {
		t.Fatal("No claims found")
	}
	if data.Claims[0].Number != 1 {
		t.Errorf("First claim number: got %d, want 1", data.Claims[0].Number)
	}
	if len(data.Claims[0].Text) < 50 {
		t.Errorf("First claim text too short: %d chars", len(data.Claims[0].Text))
	}

	t.Logf("Found %d claims", len(data.Claims))
	t.Logf("First claim: %.100s...", data.Claims[0].Text)
}

func TestParseImageInquiry(t *testing.T) {
	xmlData, err := xmlTestData.ReadFile("testdata/image-inquiry.xml")
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}

	data, err := ParseImageInquiry(string(xmlData))
	if err != nil {
		t.Fatalf("ParseImageInquiry failed: %v", err)
	}

	// Should have 3 document instances
	if len(data.DocumentInstances) != 3 {
		t.Fatalf("DocumentInstances: got %d, want 3", len(data.DocumentInstances))
	}

	// Check first instance (Drawing)
	drawing := data.DocumentInstances[0]
	if drawing.Description != "Drawing" {
		t.Errorf("Drawing desc: got %q, want %q", drawing.Description, "Drawing")
	}
	if drawing.NumberOfPages != 8 {
		t.Errorf("Drawing pages: got %d, want 8", drawing.NumberOfPages)
	}
	if drawing.DocType != "Drawing" {
		t.Errorf("Drawing type: got %q, want %q", drawing.DocType, "Drawing")
	}
	if len(drawing.Formats) != 2 {
		t.Errorf("Drawing formats: got %d, want 2", len(drawing.Formats))
	}
	if drawing.Link != "/rest-services/published-data/images/EP/1000000/B1/Drawing/fullimage" {
		t.Errorf("Drawing link: got %q", drawing.Link)
	}

	// Check second instance (FullDocument)
	fullDoc := data.DocumentInstances[1]
	if fullDoc.Description != "FullDocument" {
		t.Errorf("FullDocument desc: got %q, want %q", fullDoc.Description, "FullDocument")
	}
	if fullDoc.NumberOfPages != 15 {
		t.Errorf("FullDocument pages: got %d, want 15", fullDoc.NumberOfPages)
	}
	if len(fullDoc.Formats) != 1 {
		t.Errorf("FullDocument formats: got %d, want 1", len(fullDoc.Formats))
	}
	if fullDoc.Formats[0] != "application/pdf" {
		t.Errorf("FullDocument format: got %q, want %q", fullDoc.Formats[0], "application/pdf")
	}

	// Check third instance (FirstPageClipping)
	clipping := data.DocumentInstances[2]
	if clipping.Description != "FirstPageClipping" {
		t.Errorf("FirstPageClipping desc: got %q, want %q", clipping.Description, "FirstPageClipping")
	}
	if clipping.NumberOfPages != 1 {
		t.Errorf("FirstPageClipping pages: got %d, want 1", clipping.NumberOfPages)
	}
	if len(clipping.Formats) != 1 {
		t.Errorf("FirstPageClipping formats: got %d, want 1", len(clipping.Formats))
	}
	if clipping.Formats[0] != "image/tiff" {
		t.Errorf("FirstPageClipping format: got %q, want %q", clipping.Formats[0], "image/tiff")
	}

	t.Logf("Found %d document instances", len(data.DocumentInstances))
	for _, inst := range data.DocumentInstances {
		t.Logf("  - %s: %d pages, formats: %v", inst.Description, inst.NumberOfPages, inst.Formats)
	}
}
