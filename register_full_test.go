package epo_ops

import (
	"os"
	"testing"
)

// ParseRegister must capture the full register record - bibliographic data, statuses, and
// (from the procedural-steps constituent) the procedural step log - not just events.
func TestParseRegister_Biblio(t *testing.T) {
	b, err := os.ReadFile("testdata/register-biblio.xml")
	if err != nil {
		t.Skipf("no register biblio example: %v", err)
	}
	docs, err := ParseRegister(string(b))
	if err != nil {
		t.Fatalf("ParseRegister: %v", err)
	}
	if len(docs) == 0 {
		t.Fatal("no register-document parsed")
	}
	d := docs[0]
	if len(d.Statuses) == 0 {
		t.Error("no ep-patent-status entries captured")
	}
	if len(d.Biblio.PublicationRefs) == 0 {
		t.Error("bibliographic-data not captured (no publication references)")
	}
	hasTitle := false
	for _, ti := range d.Biblio.Titles {
		if ti.Text != "" {
			hasTitle = true
		}
	}
	if !hasTitle {
		t.Error("no invention title captured from register biblio")
	}
	if len(d.Biblio.TermsOfGrant) == 0 {
		t.Error("no term-of-grant entries captured (register-specific data dropped)")
	}
	if len(d.Biblio.MilestoneDates) == 0 {
		t.Error("no dates-rights-effective milestones captured")
	}
	if len(d.Biblio.SearchReports) == 0 {
		t.Error("no search-reports-information captured")
	}
	t.Logf("register-document: %d statuses, %d titles, %d terms-of-grant, %d citations, %d milestones, %d search-reports",
		len(d.Statuses), len(d.Biblio.Titles), len(d.Biblio.TermsOfGrant), len(d.Biblio.Citations),
		len(d.Biblio.MilestoneDates), len(d.Biblio.SearchReports))
}

func TestParseRegister_ProceduralSteps(t *testing.T) {
	b, err := os.ReadFile("testdata/register-procedural-steps.xml")
	if err != nil {
		t.Skipf("no register procedural-steps example: %v", err)
	}
	docs, err := ParseRegister(string(b))
	if err != nil {
		t.Fatalf("ParseRegister: %v", err)
	}
	if len(docs) == 0 {
		t.Fatal("no register-document parsed")
	}
	steps := docs[0].ProceduralSteps
	if len(steps) == 0 {
		t.Fatal("no procedural steps captured (this is the data ParseRegisterEvents drops)")
	}
	withCode := 0
	for _, s := range steps {
		if s.Code != "" {
			withCode++
		}
	}
	if withCode == 0 {
		t.Error("procedural steps captured but none has a code")
	}
	t.Logf("captured %d procedural steps (%d with codes)", len(steps), withCode)
}
