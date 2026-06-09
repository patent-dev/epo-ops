package epo_ops

import (
	"os"
	"strings"
	"testing"
)

func readFixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(b)
}

// TestParseExchangeDocuments_Contract pins the input contract: no exchange-document yields an
// empty slice + nil error; malformed XML returns an error.
func TestParseExchangeDocuments_Contract(t *testing.T) {
	for _, in := range []string{"", "<world-patent-data></world-patent-data>", "<other/>"} {
		docs, err := ParseExchangeDocuments(in)
		if err != nil || len(docs) != 0 {
			t.Errorf("ParseExchangeDocuments(%q) = %d docs, %v; want 0, nil", in, len(docs), err)
		}
	}
	if _, err := ParseExchangeDocuments("<broken"); err == nil {
		t.Error("ParseExchangeDocuments(malformed) = nil error; want error")
	}
}

func TestParseExchangeDocuments_FamilyBiblio(t *testing.T) {
	docs, err := ParseExchangeDocuments(readFixture(t, "family-biblio.xml"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(docs) != 6 {
		t.Fatalf("got %d exchange-documents, want 6", len(docs))
	}

	withOriginal := 0
	for _, d := range docs {
		orig := d.ApplicationNumber("original")
		if orig == "" {
			continue
		}
		withOriginal++
		// ApplicationNumber() defaults to original-preferred, so it must surface the original.
		if got := d.ApplicationNumber(); got != orig {
			t.Errorf("%s%s: ApplicationNumber()=%q, want original %q", d.Country, d.DocNumber, got, orig)
		}
	}
	if withOriginal < 4 {
		t.Errorf("only %d/6 members carry an original application number, want >=4", withOriginal)
	}

	// Every member should resolve a publication number.
	for _, d := range docs {
		if d.PublicationNumber() == "" {
			t.Errorf("%s%s: empty publication number", d.Country, d.DocNumber)
		}
	}
}

func TestParseExchangeDocuments_Citations(t *testing.T) {
	docs, err := ParseExchangeDocuments(readFixture(t, "biblio.xml"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("got %d exchange-documents, want 1", len(docs))
	}
	d := docs[0]

	if len(d.Biblio.Citations) == 0 {
		t.Fatal("no citations parsed (references-cited was previously dropped entirely)")
	}
	c := d.Biblio.Citations[0]
	if c.Category != "A" {
		t.Errorf("citation category = %q, want A", c.Category)
	}
	if c.RelClaims != "1-15" {
		t.Errorf("rel-claims = %q, want 1-15", c.RelClaims)
	}
	if len(c.RelPassages) != 5 {
		t.Errorf("rel-passages = %d, want 5", len(c.RelPassages))
	}
	if c.Patcit == nil {
		t.Fatal("patcit nil")
	}
	if got := c.Patcit.Number("docdb"); got != "WO02054790A2" {
		t.Errorf("patcit docdb number = %q, want WO02054790A2", got)
	}

	// CPC classification renders a symbol.
	var sym string
	for _, p := range d.Biblio.CPC {
		if s := p.Symbol(); s != "" {
			sym = s
			break
		}
	}
	if sym != "H04W84/20" {
		t.Errorf("first CPC symbol = %q, want H04W84/20", sym)
	}

	// Applicants are captured (per data-format).
	foundApplicant := false
	for _, a := range d.Biblio.Applicants {
		if strings.Contains(strings.ToUpper(a.Name), "9SOLUTIONS OY") {
			foundApplicant = true
		}
	}
	if !foundApplicant {
		t.Errorf("expected a 9SOLUTIONS OY applicant in %d applicants", len(d.Biblio.Applicants))
	}
}
