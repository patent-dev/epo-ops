package epo_ops

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestParseFamily(t *testing.T) {
	xmlData, err := os.ReadFile("demo/examples/get_family/response.xml")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	data, err := ParseFamily(string(xmlData))
	if err != nil {
		t.Fatalf("ParseFamily failed: %v", err)
	}

	if len(data.Members) != 5 {
		t.Errorf("Expected 5 family members, got %d", len(data.Members))
	}

	if data.PatentNumber != "EP2400812" {
		t.Errorf("Expected PatentNumber EP2400812, got %q", data.PatentNumber)
	}

	if data.FamilyID != "43088294" {
		t.Errorf("Expected FamilyID 43088294, got %q", data.FamilyID)
	}

	if data.TotalCount != 5 {
		t.Errorf("Expected TotalCount 5, got %d", data.TotalCount)
	}

	// Verify first member fields
	first := data.Members[0]
	if first.Country != "EP" || first.DocNumber != "2400812" || first.Kind != "A1" || first.Date != "20111228" {
		t.Errorf("First member: got Country=%q DocNumber=%q Kind=%q Date=%q",
			first.Country, first.DocNumber, first.Kind, first.Date)
	}

	// Verify Countries deduplicated and sorted
	expectedCountries := []string{"CA", "EP", "US"}
	if len(data.Countries) != len(expectedCountries) {
		t.Errorf("Expected %d countries, got %d: %v", len(expectedCountries), len(data.Countries), data.Countries)
	} else {
		for i, c := range expectedCountries {
			if data.Countries[i] != c {
				t.Errorf("Countries[%d] = %q, want %q", i, data.Countries[i], c)
			}
		}
	}

	// Verify no biblio fields when not included
	if first.Title != "" {
		t.Errorf("Expected empty Title without biblio, got %q", first.Title)
	}
	if first.Applicants != nil {
		t.Errorf("Expected nil Applicants without biblio, got %v", first.Applicants)
	}

	// Verify Members is non-nil
	if data.Members == nil {
		t.Error("Members should be non-nil")
	}

	t.Logf("Parsed %d family members, countries: %v", len(data.Members), data.Countries)
}

func TestParseFamily_WithBiblio(t *testing.T) {
	xmlData, err := os.ReadFile("demo/examples/get_family_with_biblio/response.xml")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	data, err := ParseFamily(string(xmlData))
	if err != nil {
		t.Fatalf("ParseFamily failed: %v", err)
	}

	if len(data.Members) != 5 {
		t.Errorf("Expected 5 family members, got %d", len(data.Members))
	}

	// Verify Title extracted (English preferred)
	first := data.Members[0]
	if first.Title != "BLUETOOTH NETWORKING" {
		t.Errorf("Expected Title 'BLUETOOTH NETWORKING', got %q", first.Title)
	}

	// Verify Applicants extracted (epodoc format)
	if len(first.Applicants) == 0 {
		t.Error("Expected non-empty Applicants from biblio data")
	} else {
		t.Logf("First member applicants: %v", first.Applicants)
		if !strings.Contains(first.Applicants[0], "9SOLUTIONS OY") {
			t.Errorf("Expected first applicant to contain '9SOLUTIONS OY', got %q", first.Applicants[0])
		}
	}

	// Verify all members have titles
	withTitle := 0
	for _, m := range data.Members {
		if m.Title != "" {
			withTitle++
		}
	}
	t.Logf("%d/%d members have Title populated", withTitle, len(data.Members))
	if withTitle == 0 {
		t.Error("Expected at least some members with Title from biblio")
	}
}

func TestParseFamily_EmptyMembers(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<ops:world-patent-data xmlns="http://www.epo.org/exchange" xmlns:ops="http://ops.epo.org">
    <ops:patent-family legal="false" total-result-count="0">
        <ops:publication-reference>
            <document-id document-id-type="docdb">
                <country>EP</country>
                <doc-number>9999999</doc-number>
                <kind>A1</kind>
            </document-id>
        </ops:publication-reference>
    </ops:patent-family>
</ops:world-patent-data>`

	data, err := ParseFamily(xmlData)
	if err != nil {
		t.Fatalf("ParseFamily should not error on empty members, got: %v", err)
	}

	if data.Members == nil {
		t.Error("Members should be non-nil empty slice, got nil")
	}

	if len(data.Members) != 0 {
		t.Errorf("Expected 0 members, got %d", len(data.Members))
	}

	if data.Countries == nil {
		t.Error("Countries should be non-nil empty slice, got nil")
	}

	if len(data.Countries) != 0 {
		t.Errorf("Expected 0 countries, got %d", len(data.Countries))
	}

	if data.TotalCount != 0 {
		t.Errorf("Expected TotalCount 0, got %d", data.TotalCount)
	}

	if data.PatentNumber != "EP9999999" {
		t.Errorf("Expected PatentNumber EP9999999, got %q", data.PatentNumber)
	}
}

func TestParseFamily_BiblioMissingFields(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<ops:world-patent-data xmlns="http://www.epo.org/exchange" xmlns:ops="http://ops.epo.org">
    <ops:patent-family legal="false" total-result-count="1">
        <ops:family-member family-id="12345">
            <publication-reference>
                <document-id document-id-type="docdb">
                    <country>EP</country>
                    <doc-number>1234567</doc-number>
                    <kind>A1</kind>
                    <date>20200101</date>
                </document-id>
            </publication-reference>
            <exchange-document system="ops.epo.org" country="EP" doc-number="1234567" kind="A1">
                <bibliographic-data>
                </bibliographic-data>
            </exchange-document>
        </ops:family-member>
    </ops:patent-family>
</ops:world-patent-data>`

	data, err := ParseFamily(xmlData)
	if err != nil {
		t.Fatalf("ParseFamily failed: %v", err)
	}

	if len(data.Members) != 1 {
		t.Fatalf("Expected 1 member, got %d", len(data.Members))
	}

	m := data.Members[0]
	if m.Title != "" {
		t.Errorf("Expected empty Title when biblio has no invention-title, got %q", m.Title)
	}
	if m.Applicants != nil {
		t.Errorf("Expected nil Applicants when biblio has no applicants, got %v", m.Applicants)
	}
	if m.Country != "EP" || m.DocNumber != "1234567" {
		t.Errorf("Expected EP1234567, got %s%s", m.Country, m.DocNumber)
	}
}

func TestParseFamily_SimpleFixture(t *testing.T) {
	xmlData, err := os.ReadFile("testdata/family.xml")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	data, err := ParseFamily(string(xmlData))
	if err != nil {
		t.Fatalf("ParseFamily failed: %v", err)
	}

	if len(data.Members) != 3 {
		t.Errorf("Expected 3 family members, got %d", len(data.Members))
	}

	// Verify Countries from simple fixture
	expectedCountries := []string{"EP", "US", "WO"}
	if len(data.Countries) != len(expectedCountries) {
		t.Errorf("Expected countries %v, got %v", expectedCountries, data.Countries)
	} else {
		for i, c := range expectedCountries {
			if data.Countries[i] != c {
				t.Errorf("Countries[%d] = %q, want %q", i, data.Countries[i], c)
			}
		}
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
			t.Logf("Event %d: Code=%s, Desc=%s, Influence=%s, Country=%s, Date=%s, EventCode=%s",
				i+1, event.Code, event.Description, event.Influence, event.Country, event.Date, event.EventCode)
		}
	}

	// Verify new fields are populated
	firstEvent := data.LegalEvents[0]
	if firstEvent.Country == "" {
		t.Error("Expected Country field to be populated from L001EP")
	}
	if firstEvent.Date == "" {
		t.Error("Expected Date field to be populated from L007EP")
	}
	if firstEvent.EventCode == "" {
		t.Error("Expected EventCode field to be populated from L008EP")
	}

	// Verify Country is consistent across events
	hasCountry := 0
	for _, event := range data.LegalEvents {
		if event.Country != "" {
			hasCountry++
		}
	}
	t.Logf("%d/%d events have Country field populated", hasCountry, len(data.LegalEvents))
	if hasCountry == 0 {
		t.Error("Expected at least some events with Country field")
	}

	t.Logf("Parsed %d legal events for %s", len(data.LegalEvents), data.PatentNumber)
}

func TestParseLegal_EmptyEvents(t *testing.T) {
	// XML with valid structure but no legal events
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<ops:world-patent-data xmlns="http://www.epo.org/exchange" xmlns:ops="http://ops.epo.org">
    <ops:patent-family legal="true" total-result-count="1">
        <ops:publication-reference>
            <document-id document-id-type="docdb">
                <country>EP</country>
                <doc-number>9999999</doc-number>
                <kind>A1</kind>
            </document-id>
        </ops:publication-reference>
        <ops:family-member family-id="99999">
        </ops:family-member>
    </ops:patent-family>
</ops:world-patent-data>`

	data, err := ParseLegal(xmlData)
	if err != nil {
		t.Fatalf("ParseLegal failed for empty events: %v", err)
	}

	if data.LegalEvents == nil {
		t.Error("LegalEvents should be non-nil empty slice, got nil")
	}

	if len(data.LegalEvents) != 0 {
		t.Errorf("Expected 0 events, got %d", len(data.LegalEvents))
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

func TestParseRegisterEvents(t *testing.T) {
	xmlData, err := os.ReadFile("demo/examples/get_register_events/response.xml")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	data, err := ParseRegisterEvents(string(xmlData))
	if err != nil {
		t.Fatalf("ParseRegisterEvents failed: %v", err)
	}

	if data.PatentNumber != "EP2400812" {
		t.Errorf("Expected PatentNumber EP2400812, got %q", data.PatentNumber)
	}

	if data.Query == "" {
		t.Error("Expected non-empty Query")
	}

	// Verify patent statuses
	if len(data.Statuses) == 0 {
		t.Error("Expected non-empty Statuses")
	} else {
		t.Logf("Parsed %d patent statuses", len(data.Statuses))
		// EP2400812 has status "No opposition filed within time limit"
		found := false
		for _, s := range data.Statuses {
			if s.Code == "7" && strings.Contains(s.Description, "No opposition filed") {
				found = true
			}
			if s.Date == "" {
				t.Error("Expected all statuses to have a Date")
			}
		}
		if !found {
			t.Error("Expected to find status code 7 (No opposition filed)")
		}
	}

	// Verify events
	if len(data.Events) == 0 {
		t.Fatal("Expected non-empty Events")
	}
	t.Logf("Parsed %d dossier events", len(data.Events))

	// Verify chronological sort (oldest first)
	for i := 1; i < len(data.Events); i++ {
		if data.Events[i].Date < data.Events[i-1].Date {
			t.Errorf("Events not sorted chronologically: event %d date %s < event %d date %s",
				i, data.Events[i].Date, i-1, data.Events[i-1].Date)
			break
		}
	}

	// Verify first event (oldest) is the publication event
	first := data.Events[0]
	if first.EventCode != "0009012" {
		t.Errorf("Expected first (oldest) event code 0009012 (publication), got %q", first.EventCode)
	}
	if first.Category != "publication" {
		t.Errorf("Expected first event category 'publication', got %q", first.Category)
	}
	if first.Date == "" {
		t.Error("Expected first event to have a Date")
	}
	if first.Description == "" {
		t.Error("Expected first event to have a Description")
	}

	// Verify grant event exists
	grantFound := false
	for _, evt := range data.Events {
		if evt.EventCode == "0009210" {
			grantFound = true
			if evt.Category != "grant" {
				t.Errorf("Expected grant category for 0009210, got %q", evt.Category)
			}
		}
	}
	if !grantFound {
		t.Error("Expected to find grant event 0009210")
	}

	// Verify examination events exist
	examFound := false
	for _, evt := range data.Events {
		if evt.Category == "examination" {
			examFound = true
			break
		}
	}
	if !examFound {
		t.Error("Expected to find at least one examination event")
	}

	// Verify gazette reference on events that have it
	hasGazette := 0
	for _, evt := range data.Events {
		if evt.GazetteNum != "" {
			hasGazette++
		}
	}
	t.Logf("%d/%d events have gazette reference", hasGazette, len(data.Events))
	if hasGazette == 0 {
		t.Error("Expected at least some events with gazette reference")
	}
}

// TestParseRegisterEvents_AggregatesAllDocuments verifies that statuses and
// events are collected across every register-document in the response, not
// only the first, and that the ascending sort spans all of them.
func TestParseRegisterEvents_AggregatesAllDocuments(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<ns2:world-patent-data xmlns:ns2="http://ops.epo.org" xmlns:ns3="http://www.epo.org/register">
    <ns2:register-search total-result-count="2">
        <ns2:query syntax="CQL">publication=EP1000000</ns2:query>
        <ns2:range begin="1" end="2"/>
        <ns3:register-documents produced-by="RO">
            <ns3:register-document>
                <ns3:ep-patent-statuses>
                    <ns3:ep-patent-status change-date="20191025" status-code="8">The patent has been granted</ns3:ep-patent-status>
                </ns3:ep-patent-statuses>
                <ns3:events-data>
                    <ns3:dossier-event id="EVT2">
                        <ns3:event-date><ns3:date>20191127</ns3:date></ns3:event-date>
                        <ns3:event-code>0009210</ns3:event-code>
                        <ns3:event-text event-text-type="STATUS">Grant of the patent</ns3:event-text>
                    </ns3:dossier-event>
                </ns3:events-data>
            </ns3:register-document>
            <ns3:register-document>
                <ns3:ep-patent-statuses>
                    <ns3:ep-patent-status change-date="20111228" status-code="17">Application published</ns3:ep-patent-status>
                </ns3:ep-patent-statuses>
                <ns3:events-data>
                    <ns3:dossier-event id="EVT1">
                        <ns3:event-date><ns3:date>20111228</ns3:date></ns3:event-date>
                        <ns3:event-code>0009012</ns3:event-code>
                        <ns3:event-text event-text-type="STATUS">Publication of the application</ns3:event-text>
                    </ns3:dossier-event>
                </ns3:events-data>
            </ns3:register-document>
        </ns3:register-documents>
    </ns2:register-search>
</ns2:world-patent-data>`

	data, err := ParseRegisterEvents(xmlData)
	if err != nil {
		t.Fatalf("ParseRegisterEvents failed: %v", err)
	}

	if len(data.Statuses) != 2 {
		t.Errorf("Expected 2 statuses aggregated across documents, got %d", len(data.Statuses))
	}
	if len(data.Events) != 2 {
		t.Fatalf("Expected 2 events aggregated across documents, got %d", len(data.Events))
	}

	// The sort must remain ascending across document boundaries: the second
	// document's 2011 event comes before the first document's 2019 event.
	if data.Events[0].EventCode != "0009012" || data.Events[1].EventCode != "0009210" {
		t.Errorf("Events not sorted ascending across documents: got %s then %s",
			data.Events[0].EventCode, data.Events[1].EventCode)
	}
}

func TestParseRegisterEvents_EmptyEvents(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<ns2:world-patent-data xmlns:ns2="http://ops.epo.org" xmlns:ns3="http://www.epo.org/register">
    <ns2:register-search total-result-count="0">
        <ns2:query syntax="CQL">publication=EP9999999</ns2:query>
        <ns2:range begin="0" end="0"/>
        <ns3:register-documents produced-by="RO">
        </ns3:register-documents>
    </ns2:register-search>
</ns2:world-patent-data>`

	data, err := ParseRegisterEvents(xmlData)
	if err != nil {
		t.Fatalf("ParseRegisterEvents should not error on empty events, got: %v", err)
	}

	if data.Events == nil {
		t.Error("Events should be non-nil empty slice, got nil")
	}
	if len(data.Events) != 0 {
		t.Errorf("Expected 0 events, got %d", len(data.Events))
	}

	if data.Statuses == nil {
		t.Error("Statuses should be non-nil empty slice, got nil")
	}
	if len(data.Statuses) != 0 {
		t.Errorf("Expected 0 statuses, got %d", len(data.Statuses))
	}

	if data.PatentNumber != "EP9999999" {
		t.Errorf("Expected PatentNumber EP9999999, got %q", data.PatentNumber)
	}
}

func TestCategorizeRegisterEvent(t *testing.T) {
	tests := []struct {
		code     string
		desc     string
		expected string
	}{
		// Publication
		{"0009012", "Publication in section I.1 EP Bulletin", "publication"},
		// Examination
		{"0009171", "Request for examination filed", "examination"},
		{"0009185", "First examination report", "examination"},
		{"EPIDOSNEXR2", "Despatch of examination report", "examination"},
		{"EPIDOSNEXR3", "Reply to examination report", "examination"},
		{"EPIDOSCEXR2", "Change: Despatch of examination report", "examination"},
		// Grant
		{"0009210", "(Expected) grant", "grant"},
		{"EPIDOSNIGR1", "Communication of intention to grant", "grant"},
		{"EPIDOSNIGR3", "Payment of fee for grant", "grant"},
		{"EPIDOSNIGR5", "Payment of fee for publishing/printing", "grant"},
		{"EPIDOSNIGR7", "Receipt of the translation of the claim(s)", "grant"},
		// No-opposition (0009261 means "no opposition filed", not "opposition filed")
		{"0009261", "No opposition filed within time limit", "no_opposition"},
		// No-opposition via description fallback
		{"UNKNOWN00", "No opposition filed within time limit", "no_opposition"},
		// Actual opposition
		{"EPIDOSCOPPO", "Opposition filed", "opposition"},
		// Appeal (description-based fallback)
		{"UNKNOWN01", "Appeal filed by the patent proprietor", "appeal"},
		{"UNKNOWN02", "Decision on appeal", "appeal"},
		// Filing
		{"0009100", "Filing of application", "filing"},
		// Other (renewals, representative changes, lapses, etc.)
		{"0009250", "Lapse of the patent", "other"},
		{"EPIDOSNRFE2", "Renewal fee paid", "other"},
		{"EPIDOS FREP", "Change - representative", "other"},
		{"0009199APPR", "Change - applicant", "other"},
		{"0009199TIEN", "Change - English title", "other"},
		{"EPIDOS ABEX", "Amendment by applicant", "other"},
		{"0008199IPCL", "Change - classification", "other"},
		{"EPIDOSNEXP2", "Fee payment for extension state", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			got := categorizeRegisterEvent(tt.code, tt.desc)
			if got != tt.expected {
				t.Errorf("categorizeRegisterEvent(%q, %q) = %q, want %q", tt.code, tt.desc, got, tt.expected)
			}
		})
	}
}

func TestParseNumberConversion(t *testing.T) {
	xmlData, err := os.ReadFile("demo/examples/convert_patent_number_epodoc/response.xml")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	data, err := ParseNumberConversion(string(xmlData))
	if err != nil {
		t.Fatalf("ParseNumberConversion failed: %v", err)
	}

	if data.InputFormat != "docdb" {
		t.Errorf("Expected InputFormat 'docdb', got %q", data.InputFormat)
	}
	if data.OutputFormat != "epodoc" {
		t.Errorf("Expected OutputFormat 'epodoc', got %q", data.OutputFormat)
	}
	if data.DocNumber != "EP2400812" {
		t.Errorf("Expected DocNumber 'EP2400812', got %q", data.DocNumber)
	}
	if data.Kind != "A1" {
		t.Errorf("Expected Kind 'A1', got %q", data.Kind)
	}
	if data.Date != "20111228" {
		t.Errorf("Expected Date '20111228', got %q", data.Date)
	}
	// Epodoc format has no separate country element
	if data.Country != "" {
		t.Errorf("Expected empty Country for epodoc format, got %q", data.Country)
	}
}

func TestParseNumberConversion_Original(t *testing.T) {
	xmlData, err := os.ReadFile("demo/examples/convert_patent_number_original/response.xml")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	data, err := ParseNumberConversion(string(xmlData))
	if err != nil {
		t.Fatalf("ParseNumberConversion failed: %v", err)
	}

	if data.InputFormat != "docdb" {
		t.Errorf("Expected InputFormat 'docdb', got %q", data.InputFormat)
	}
	if data.OutputFormat != "original" {
		t.Errorf("Expected OutputFormat 'original', got %q", data.OutputFormat)
	}
	if data.Country != "EP" {
		t.Errorf("Expected Country 'EP', got %q", data.Country)
	}
	if data.DocNumber != "2400812" {
		t.Errorf("Expected DocNumber '2400812', got %q", data.DocNumber)
	}
	if data.Kind != "A1" {
		t.Errorf("Expected Kind 'A1', got %q", data.Kind)
	}
	if data.Date != "20111228" {
		t.Errorf("Expected Date '20111228', got %q", data.Date)
	}
}

func TestParseNumberConversion_EmptyOutput(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<ops:world-patent-data xmlns:ops="http://ops.epo.org">
  <ops:standardization inputFormat="docdb" outputFormat="epodoc">
  </ops:standardization>
</ops:world-patent-data>`

	_, err := ParseNumberConversion(xmlData)
	if err == nil {
		t.Fatal("Expected error for empty output, got nil")
	}
	var valErr *DataValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("Expected DataValidationError, got %T: %v", err, err)
	}
	if valErr.MissingField != "doc-number" {
		t.Errorf("Expected MissingField 'doc-number', got %q", valErr.MissingField)
	}
}

func TestParseClassificationSchema(t *testing.T) {
	xmlData, err := os.ReadFile("demo/examples/get_classification_schema/response.xml")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	data, err := ParseClassificationSchema(string(xmlData))
	if err != nil {
		t.Fatalf("ParseClassificationSchema failed: %v", err)
	}

	if data.Symbol != "H04W" {
		t.Errorf("Expected Symbol 'H04W', got %q", data.Symbol)
	}
	if data.Title != "WIRELESS COMMUNICATION NETWORKS" {
		t.Errorf("Expected Title 'WIRELESS COMMUNICATION NETWORKS', got %q", data.Title)
	}
	if data.Level != 5 {
		t.Errorf("Expected Level 5, got %d", data.Level)
	}
	if data.SchemeType != "cpc" {
		t.Errorf("Expected SchemeType 'cpc', got %q", data.SchemeType)
	}
	// H04W subclass-level lookup has no children
	if len(data.Children) != 0 {
		t.Errorf("Expected 0 children, got %d", len(data.Children))
	}
	if data.Children == nil {
		t.Error("Expected non-nil Children slice")
	}
}

func TestParseClassificationSchema_WithChildren(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<ops:world-patent-data xmlns:ops="http://ops.epo.org">
  <ops:classification-scheme>
    <ops:cpc>
      <cpc:class-scheme xmlns:cpc="http://www.epo.org/cpcexport" scheme-type="cpc">
        <cpc:classification-item level="7" status="published">
          <cpc:classification-symbol>H04W84</cpc:classification-symbol>
          <cpc:class-title><cpc:title-part><cpc:text>Topology</cpc:text></cpc:title-part></cpc:class-title>
          <cpc:classification-item level="8" status="published">
            <cpc:classification-symbol>H04W84/02</cpc:classification-symbol>
            <cpc:class-title><cpc:title-part><cpc:text>Hierarchically pre-organised networks</cpc:text></cpc:title-part></cpc:class-title>
          </cpc:classification-item>
          <cpc:classification-item level="8" status="published">
            <cpc:classification-symbol>H04W84/18</cpc:classification-symbol>
            <cpc:class-title><cpc:title-part><cpc:text>Self-organising networks</cpc:text></cpc:title-part></cpc:class-title>
          </cpc:classification-item>
        </cpc:classification-item>
      </cpc:class-scheme>
    </ops:cpc>
  </ops:classification-scheme>
</ops:world-patent-data>`

	data, err := ParseClassificationSchema(xmlData)
	if err != nil {
		t.Fatalf("ParseClassificationSchema failed: %v", err)
	}

	if data.Symbol != "H04W84" {
		t.Errorf("Expected Symbol 'H04W84', got %q", data.Symbol)
	}
	if data.Title != "Topology" {
		t.Errorf("Expected Title 'Topology', got %q", data.Title)
	}
	if data.Level != 7 {
		t.Errorf("Expected Level 7, got %d", data.Level)
	}
	if len(data.Children) != 2 {
		t.Fatalf("Expected 2 children, got %d", len(data.Children))
	}

	// Verify first child
	if data.Children[0].Symbol != "H04W84/02" {
		t.Errorf("Expected child[0] Symbol 'H04W84/02', got %q", data.Children[0].Symbol)
	}
	if data.Children[0].Title != "Hierarchically pre-organised networks" {
		t.Errorf("Expected child[0] Title 'Hierarchically pre-organised networks', got %q", data.Children[0].Title)
	}
	if data.Children[0].Level != 8 {
		t.Errorf("Expected child[0] Level 8, got %d", data.Children[0].Level)
	}

	// Verify second child
	if data.Children[1].Symbol != "H04W84/18" {
		t.Errorf("Expected child[1] Symbol 'H04W84/18', got %q", data.Children[1].Symbol)
	}
	if data.Children[1].Title != "Self-organising networks" {
		t.Errorf("Expected child[1] Title 'Self-organising networks', got %q", data.Children[1].Title)
	}
}

func TestParseClassificationSchema_HasChildrenAttr(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<ops:world-patent-data xmlns:ops="http://ops.epo.org">
  <ops:classification-scheme>
    <ops:cpc>
      <cpc:class-scheme xmlns:cpc="http://www.epo.org/cpcexport" scheme-type="cpc">
        <cpc:classification-item level="5" has-children="true">
          <cpc:classification-symbol>H04W</cpc:classification-symbol>
          <cpc:class-title><cpc:title-part><cpc:text>WIRELESS COMMUNICATION NETWORKS</cpc:text></cpc:title-part></cpc:class-title>
          <cpc:classification-item level="6" has-children="true">
            <cpc:classification-symbol>H04W4/00</cpc:classification-symbol>
            <cpc:class-title><cpc:title-part><cpc:text>Services specially adapted for wireless communication networks</cpc:text></cpc:title-part></cpc:class-title>
          </cpc:classification-item>
          <cpc:classification-item level="6" has-children="false">
            <cpc:classification-symbol>H04W8/00</cpc:classification-symbol>
            <cpc:class-title><cpc:title-part><cpc:text>Network data management</cpc:text></cpc:title-part></cpc:class-title>
          </cpc:classification-item>
        </cpc:classification-item>
      </cpc:class-scheme>
    </ops:cpc>
  </ops:classification-scheme>
</ops:world-patent-data>`

	data, err := ParseClassificationSchema(xmlData)
	if err != nil {
		t.Fatalf("ParseClassificationSchema failed: %v", err)
	}
	if !data.HasChildren {
		t.Error("Expected HasChildren=true on target symbol")
	}
	if len(data.Children) != 2 {
		t.Fatalf("Expected 2 children, got %d", len(data.Children))
	}
	if !data.Children[0].HasChildren {
		t.Errorf("Expected child[0] HasChildren=true, got false")
	}
	if data.Children[1].HasChildren {
		t.Errorf("Expected child[1] HasChildren=false, got true")
	}
}

func TestParseClassificationSchema_EmptyResponse(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<ops:world-patent-data xmlns:ops="http://ops.epo.org">
  <ops:classification-scheme>
    <ops:cpc>
      <cpc:class-scheme xmlns:cpc="http://www.epo.org/cpcexport" scheme-type="cpc">
      </cpc:class-scheme>
    </ops:cpc>
  </ops:classification-scheme>
</ops:world-patent-data>`

	data, err := ParseClassificationSchema(xmlData)
	if err != nil {
		t.Fatalf("ParseClassificationSchema failed: %v", err)
	}
	if data == nil {
		t.Fatal("Expected non-nil result")
	}
	if data.Children == nil {
		t.Error("Expected non-nil Children slice")
	}
	if len(data.Children) != 0 {
		t.Errorf("Expected 0 children, got %d", len(data.Children))
	}
}
