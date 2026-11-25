package epo_ops

import (
	"errors"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// documents_test.go covers the responses that hold more than one document and the fields
// added so that nothing OPS sends is dropped. Every expectation is a value of a recorded
// response in testdata.

// TestParseBiblio_KindlessNumber uses the response to
// published-data/publication/epodoc/EP2400812/biblio: a number without a kind code, answered
// with one exchange-document per publication kind (A1 of 2011, B1 of 2019).
func TestParseBiblio_KindlessNumber(t *testing.T) {
	raw := string(readFixtureBytes(t, "biblio-kindless-ep2400812.xml"))

	docs, err := ParseBiblioDocuments(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 {
		t.Fatalf("ParseBiblioDocuments returned %d documents, want 2", len(docs))
	}
	want := []struct {
		number, kind, date string
		citations          int
	}{
		{"EP2400812A1", "A1", "20111228", 6},
		{"EP2400812B1", "B1", "20191127", 0},
	}
	for i, w := range want {
		d := docs[i]
		assertEq(t, "PatentNumber", d.PatentNumber, w.number)
		assertEq(t, "Kind", d.Kind, w.kind)
		assertEq(t, "PublicationDate", d.PublicationDate, w.date)
		assertEq(t, "FamilyID", d.FamilyID, "43088294")
		// Each document carries its own parties and classifications exactly once.
		assertLen(t, w.kind+" Applicants", len(d.Applicants), 1)
		assertLen(t, w.kind+" Inventors", len(d.Inventors), 4)
		assertLen(t, w.kind+" IPCClasses", len(d.IPCClasses), 1)
		assertLen(t, w.kind+" CPCClasses", len(d.CPCClasses), 4)
		if d.Document == nil {
			t.Fatalf("%s: Document is nil", w.kind)
		}
		assertEq(t, w.kind+" Document.Kind", d.Document.Kind, w.kind)
		assertLen(t, w.kind+" Document citations", len(d.Document.Biblio.Citations), w.citations)
	}

	// ParseBiblio returns the last document of the response, whole.
	got, err := ParseBiblio(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, docs[1]) {
		t.Errorf("ParseBiblio = %+v\nwant the last document %+v", got, docs[1])
	}
}

// TestParseBiblio_SingleDocument pins that a response with one exchange-document gives the
// same result through both functions.
func TestParseBiblio_SingleDocument(t *testing.T) {
	raw := string(readFixtureBytes(t, "biblio.xml"))
	one, err := ParseBiblio(raw)
	if err != nil {
		t.Fatal(err)
	}
	docs, err := ParseBiblioDocuments(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("got %d documents, want 1", len(docs))
	}
	if !reflect.DeepEqual(one, docs[0]) {
		t.Errorf("ParseBiblio and ParseBiblioDocuments[0] differ:\n%+v\n%+v", one, docs[0])
	}
}

// TestParseBiblio_NoDocument pins the result for a response without an exchange-document
// under world-patent-data/exchange-documents.
func TestParseBiblio_NoDocument(t *testing.T) {
	got, err := ParseBiblio(string(readFixtureBytes(t, "claims.xml")))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, &BiblioData{Titles: map[string]string{}}) {
		t.Errorf("got %+v, want an empty BiblioData", got)
	}
	docs, err := ParseBiblioDocuments(string(readFixtureBytes(t, "claims.xml")))
	if err != nil || len(docs) != 0 {
		t.Errorf("ParseBiblioDocuments = %d documents, err %v; want none", len(docs), err)
	}
}

// TestBiblioData_FullDocument checks the fields BiblioData did not expose before.
func TestBiblioData_FullDocument(t *testing.T) {
	d, err := ParseBiblio(string(readFixtureBytes(t, "biblio-kindless-ep2400812.xml")))
	if err != nil {
		t.Fatal(err)
	}
	doc := d.Document

	// Application reference: number in every format, and the filing date.
	app := doc.Biblio.ApplicationRefs[0]
	assertEq(t, "application docdb", app.ByType("docdb"), "10167109")
	assertEq(t, "application epodoc", app.ByType("epodoc"), "EP20100167109")
	assertEq(t, "application original", app.ByType("original"), "10167109.7")
	assertEq(t, "application date", app.DocumentIDs[0].Date, "20100624")

	// Titles in every language.
	titles := map[string]string{}
	for _, tt := range doc.Biblio.Titles {
		titles[tt.Lang] = tt.Text
	}
	assertEq(t, "title de", titles["de"], "BLUETOOTH-VERNETZUNG")
	assertEq(t, "title fr", titles["fr"], "MISE EN RÉSEAU BLUETOOTH")
	assertEq(t, "title en", titles["en"], "BLUETOOTH NETWORKING")

	// Party names in the original format, next to the epodoc ones.
	var original []string
	for _, a := range doc.Biblio.Applicants {
		if a.DataFormat == "original" {
			original = append(original, a.Name)
		}
	}
	if got := strings.Join(original, "|"); got != "9SOLUTIONS OY|9Solutions Oy" {
		t.Errorf("original applicant names = %q", got)
	}

	// Priority claim kind.
	assertEq(t, "priority kind", doc.Biblio.Priorities[0].Kind, "regional")

	// Two rows for H04W 84/20 that differ only by the generating office.
	c0, c1 := d.CPCClasses[0], d.CPCClasses[1]
	assertEq(t, "cpc 0", c0.Full, "H04W 84/20")
	assertEq(t, "cpc 1", c1.Full, "H04W 84/20")
	assertEq(t, "cpc 0 scheme", c0.Scheme, "CPCI")
	assertEq(t, "cpc 0 scheme office", c0.SchemeOffice, "EP")
	assertEq(t, "cpc 0 value", c0.Value, "I")
	assertEq(t, "cpc 0 generating office", c0.GeneratingOffice, "US")
	assertEq(t, "cpc 1 generating office", c1.GeneratingOffice, "EP")
	assertEq(t, "cpc 0 sequence", c0.Sequence, "1")
	assertEq(t, "cpc 1 sequence", c1.Sequence, "2")
}

// TestExchangeDocument_TagsCoverEveryDocument runs the tag-based coverage over EVERY
// exchange-document of the recordings (TestElementCoverage looks at the first only).
func TestExchangeDocument_TagsCoverEveryDocument(t *testing.T) {
	schemaEls, schemaAttrs := schemaPaths(reflect.TypeOf(ExchangeDocument{}), "exchange-document")
	for _, f := range []string{"biblio.xml", "biblio-kindless-ep2400812.xml", "family-biblio.xml", "search.xml", "abstract.xml"} {
		var missing []string
		seen := map[string]bool{}
		for _, fv := range collectFixtureValues(t, readFixtureBytes(t, f), "exchange-document") {
			path, attr, isAttr := strings.Cut(fv.where, " @")
			ok := schemaEls[path]
			if isAttr {
				ok = schemaAttrs[path+" @"+attr]
			}
			// The one attribute without a field: see TestIPCRSequenceIsPosition.
			if fv.where == "/exchange-document/bibliographic-data/classifications-ipcr/classification-ipcr @sequence" {
				ok = true
			}
			if !ok && !seen[fv.where] {
				seen[fv.where] = true
				missing = append(missing, fv.where)
			}
		}
		sort.Strings(missing)
		if len(missing) > 0 {
			t.Errorf("%s: no ExchangeDocument field for:\n  %s", f, strings.Join(missing, "\n  "))
		}
	}
}

// TestIPCRSequenceIsPosition proves the sequence attribute of classification-ipcr, the one
// attribute BibliographicData has no field for, is the entry's 1-based position in IPCR in
// every recorded document, so it can be read as index+1.
func TestIPCRSequenceIsPosition(t *testing.T) {
	type entry struct {
		Sequence string `xml:"sequence,attr"`
		Text     string `xml:"text"`
	}
	type doc struct {
		IPCR []entry `xml:"bibliographic-data>classifications-ipcr>classification-ipcr"`
	}
	for _, f := range []string{"biblio.xml", "biblio-kindless-ep2400812.xml", "family-biblio.xml", "search.xml"} {
		raw := string(readFixtureBytes(t, f))
		withSeq, err := decodeElements[doc](raw, "test", "exchange-document")
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := ParseExchangeDocuments(raw)
		if err != nil {
			t.Fatal(err)
		}
		if len(withSeq) != len(parsed) {
			t.Fatalf("%s: %d vs %d documents", f, len(withSeq), len(parsed))
		}
		for i, d := range withSeq {
			for j, e := range d.IPCR {
				if e.Sequence != strconv.Itoa(j+1) {
					t.Errorf("%s document %d: classification-ipcr %d has sequence %q", f, i, j+1, e.Sequence)
				}
				if parsed[i].Biblio.IPCR[j] != e.Text {
					t.Errorf("%s document %d: IPCR[%d] = %q, want %q", f, i, j, parsed[i].Biblio.IPCR[j], e.Text)
				}
			}
		}
	}
}

func parseRegisterFixture(t *testing.T, name string) RegisterDocument {
	t.Helper()
	docs, err := ParseRegister(string(readFixtureBytes(t, name)))
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("%s: got %d register documents, want 1", name, len(docs))
	}
	return docs[0]
}

// TestParseRegister_Parties checks former representatives, the change stamps, address lines
// and rep-type, and applicants and inventors with addresses (EP4303979).
func TestParseRegister_Parties(t *testing.T) {
	b := parseRegisterFixture(t, "register-biblio-ep4303979.xml").Biblio

	if len(b.AgentGroups) != 2 {
		t.Fatalf("got %d representative blocks, want 2", len(b.AgentGroups))
	}
	current := b.CurrentRepresentatives()[0]
	assertEq(t, "current name", current.Addressbook.Name, "Kraus & Lederer PartGmbB")
	assertEq(t, "current rep-type", current.RepType, "agent")
	assertEq(t, "current address", strings.Join(current.Addressbook.AddressLines(), "|"), "Thomas-Wimmer-Ring 15|80539 München")
	assertEq(t, "current country", current.Addressbook.Country, "DE")
	assertEq(t, "current cdsid", current.Addressbook.CDSID, "0101492436")

	former := b.AgentGroups[1]
	assertEq(t, "former change date", former.ChangeDate, "20231208")
	assertEq(t, "former gazette", former.GazetteNum, "2024/02")
	assertEq(t, "former name", former.Representatives[0].Addressbook.Name, "Beckmann, Claus")
	assertEq(t, "former address", strings.Join(former.Representatives[0].Addressbook.AddressLines(), "|"),
		"Kraus & Weisert|Patentanwälte PartGmbB|Thomas-Wimmer-Ring 15|80539 München")
	// Representatives and Agents are the same entries, index for index.
	for _, g := range b.AgentGroups {
		if len(g.Agents) != len(g.Representatives) {
			t.Fatalf("block %s: %d agents, %d representatives", g.ChangeDate, len(g.Agents), len(g.Representatives))
		}
		for i, a := range g.Agents {
			assertEq(t, "agent name", a.Name, g.Representatives[i].Addressbook.Name)
			assertEq(t, "agent country", a.Country, g.Representatives[i].Addressbook.Country)
		}
	}

	if len(b.ApplicantGroups) != 2 {
		t.Fatalf("got %d applicant blocks, want 2", len(b.ApplicantGroups))
	}
	now := b.ApplicantGroups[0]
	assertEq(t, "applicant block date", now.ChangeDate, "20240830")
	assertEq(t, "applicant block gazette", now.GazetteNum, "2024/40")
	assertEq(t, "applicant name", now.Applicants[0].Name, "Contemporary Amperex Technology (Hong Kong) Limited")
	assertEq(t, "applicant address", strings.Join(now.Applicants[0].Addressbook.AddressLines(), "|"),
		"Level 19, China Building|29 Queen's Road Central|Central, Central And Western District")
	assertEq(t, "applicant country", now.Applicants[0].Addressbook.Country, "HK")
	assertEq(t, "former applicant", b.ApplicantGroups[1].Applicants[0].Name, "Contemporary Amperex Technology Co., Limited")
	// The flat list still holds the applicants of all blocks.
	assertLen(t, "flat Applicants", len(b.Applicants), 2)

	inv := b.InventorGroups[0].Inventors[0]
	assertEq(t, "inventor name", inv.Name, "WU, Zeli")
	assertEq(t, "inventor address", strings.Join(inv.Addressbook.AddressLines(), "|"), "Ningde City, Fujian 352100")
	assertEq(t, "inventor country", inv.Addressbook.Country, "CN")
}

// TestParseRegister_Record checks the status text and its change dates, filing language,
// designated states, examination dates, divisional children and search report information
// (EP4303979).
func TestParseRegister_Record(t *testing.T) {
	doc := parseRegisterFixture(t, "register-biblio-ep4303979.xml")
	b := doc.Biblio

	assertEq(t, "status text", b.Status, "No opposition filed within time limit")
	assertEq(t, "status code", doc.Statuses[0].Code, "7")
	assertEq(t, "status change date", doc.Statuses[0].ChangeDate, "20261002")
	assertEq(t, "biblio id", b.ID, "EP22926312P")
	assertEq(t, "date produced", doc.DateProduced, "20261107")
	assertEq(t, "language of filing", b.LanguageOfFiling, "zh")

	g := b.DesignationGroups[0]
	assertEq(t, "designation change date", g.ChangeDate, "20231208")
	assertEq(t, "designation gazette", g.GazetteNum, "2024/02")
	assertEq(t, "designation region", strings.Join(g.PCTRegion, ","), "EP")
	assertLen(t, "designated states", len(g.PCTRegional), 38)

	assertEq(t, "milestone 0", b.Milestones[0].Name, "request-for-examination")
	assertEq(t, "milestone 0 date", b.Milestones[0].Date, "20230821")
	assertEq(t, "milestone 0 gazette", b.Milestones[0].ChangeGazetteNum, "2024/02")
	assertEq(t, "milestone 1", b.Milestones[1].Name, "first-examination-report-despatched")
	assertEq(t, "milestone 1 date", b.Milestones[1].Date, "20250717")
	assertEq(t, "MilestoneDates", b.MilestoneDates["first-examination-report-despatched"], "20250717")

	if len(b.RelatedDocuments) != 1 {
		t.Fatalf("got %d related documents, want 1", len(b.RelatedDocuments))
	}
	rel := b.RelatedDocuments[0]
	assertEq(t, "relation", rel.Type, "division")
	assertEq(t, "child application", rel.Children[0].Type+" "+rel.Children[0].Country+rel.Children[0].DocNumber, "application number EP20250205834")
	assertEq(t, "child publication", rel.Children[1].Type+" "+rel.Children[1].Country+rel.Children[1].DocNumber, "publication number EP4651263")

	isr, supp := b.SearchReports[0], b.SearchReports[1]
	assertEq(t, "isr office", isr.Office, "CN")
	assertEq(t, "isr type", isr.SearchType, "isr")
	assertEq(t, "isr no-search", isr.DeclarationOfNoSearch, "not-determined")
	assertEq(t, "isr gazette", isr.ChangeGazetteNum, "2023/48")
	assertEq(t, "isr publication", isr.Country+isr.DocNumber+isr.Kind, "WO2023225799A1")
	assertEq(t, "isr publication lang", isr.PublicationLang, "zh")
	assertEq(t, "supplementary type", supp.SearchType, "national-supplementary")
	assertEq(t, "supplementary mailed", supp.MailedDate, "20241205")

	assertEq(t, "opposition-not-filed date", b.Oppositions[0].NotFiledDate, "20260827")
	if !b.Opposition.NotFiled() {
		t.Error("Opposition.NotFiled() = false")
	}

	cit := b.Citations[0]
	assertEq(t, "citation id", cit.ID, "cit_6596814")
	assertEq(t, "citation office", cit.Office, "EP")
	assertEq(t, "citation url", cit.Patcit.URL, "https://worldwide.espacenet.com/patent/search?q=pn%3DCN108659773&called_by=EuropeanRegister")

	ipcr := b.IPCRGroups[1]
	assertEq(t, "ipcr block date", ipcr.ChangeDate, "20231208")
	assertEq(t, "ipcr text", ipcr.Classifications[0].Text, "H01M10/056, H01M10/0567")
}

// TestParseRegister_Opposition uses the register record of EP2771468, a patent that was
// opposed, revoked and appealed.
func TestParseRegister_Opposition(t *testing.T) {
	doc := parseRegisterFixture(t, "register-biblio-ep2771468.xml")
	b := doc.Biblio

	assertEq(t, "status", b.Status, "Patent revoked")
	assertEq(t, "status code", doc.Statuses[0].Code, "4")

	if len(b.Oppositions) != 12 {
		t.Fatalf("got %d opposition blocks, want 12", len(b.Oppositions))
	}
	cur := b.CurrentOpposition()
	assertEq(t, "block date", cur.ChangeDate, "20190830")
	assertEq(t, "block gazette", cur.GazetteNum, "N/P")
	if cur.NotFiled() {
		t.Error("an opposed patent reports no opposition filed")
	}
	o := cur.Opponents[0]
	assertEq(t, "opponent sequence", o.Sequence, "01")
	assertEq(t, "opponent name", o.Addressbook.Name, "Schlich, George William")
	assertEq(t, "opponent address", strings.Join(o.Addressbook.AddressLines(), "|"), "9 St Catherine's Road|Littlehampton|Sussex BN17 5HS")
	assertEq(t, "opponent country", o.Addressbook.Country, "GB")
	assertEq(t, "opponent status", o.Status, "ADMISSIBLE")
	assertEq(t, "opponent status date", o.StatusDate, "20151029")
	assertEq(t, "opposition filed", o.FiledDate, "20151026")
	assertEq(t, "opponent representative", o.Agent.Addressbook.Name, "Schlich, George William, et al")
	assertEq(t, "opponent representative type", o.Agent.RepType, "agent")

	assertEq(t, "revocation date", b.Revocations[0].Date, "20200116")
	assertEq(t, "revocation gazette", b.Revocations[0].GazetteNum, "2020/53")

	appeal := b.OfficeSpecific[0].Appeals[0]
	assertEq(t, "office", b.OfficeSpecific[0].Office, "EPO")
	assertEq(t, "appeal number", appeal.Number, "T0844/18-3308")
	assertEq(t, "appeal phase", appeal.Phase, "opposition")
	assertEq(t, "appeal date", appeal.Date, "20180328")
	assertEq(t, "grounds filed", appeal.StatementOfGroundsDate, "20180801")
	assertEq(t, "appeal result", appeal.Result, "APPEAL OF PROPRIETOR REJECTED")
	assertEq(t, "appeal result date", appeal.ResultDate, "20200116")

	cpc := b.CPCGroups[0].Classifications[0]
	assertEq(t, "cpc scheme", cpc.Scheme, "CPCI")
	assertEq(t, "cpc status", cpc.Status, "confirmed")
	assertEq(t, "cpc sequence", cpc.Sequence, "1")

	// Register priority claims state country, number and date on the claim itself.
	p := b.PriorityGroups[0].Priorities[0]
	assertEq(t, "priority", p.Kind+" "+p.Country+p.DocNumber+" "+p.Date, "national US201261736527P 20121212")

	// A parent lists each divisional application as a child.
	child := b.RelatedDocuments[0].Children[1]
	assertEq(t, "first divisional", child.Country+child.DocNumber, "EP2784162")
	assertLen(t, "applicant blocks", len(b.ApplicantGroups), 3)
}

// TestParseRegister_UnitaryPatent checks the procedural steps and events of the unitary-patent
// block, and the parent of a divisional (EP3970521).
func TestParseRegister_UnitaryPatent(t *testing.T) {
	doc := parseRegisterFixture(t, "register-unip-ep3970521.xml")

	if len(doc.UnitaryProceduralSteps) != 2 {
		t.Fatalf("got %d unitary procedural steps, want 2", len(doc.UnitaryProceduralSteps))
	}
	req, dec := doc.UnitaryProceduralSteps[0], doc.UnitaryProceduralSteps[1]
	assertEq(t, "step 0 code", req.Code, "UDFI")
	assertEq(t, "step 0 phase", req.Phase, "unitary-protection")
	assertEq(t, "step 0 date", req.DateByType("DATE_OF_REQUEST"), "20240104")
	assertEq(t, "step 0 gazette", req.Gazette.Num, "2024/06")
	assertEq(t, "step 1 code", dec.Code, "UDEC")
	assertEq(t, "step 1 description", dec.Description(), "Date and purport of the decision on the registration of unitary effect")
	assertEq(t, "step 1 date", dec.DateByType("DATE_OF_DECISION"), "20240408")
	assertEq(t, "step 1 result", strings.Join(dec.Results, ","), "Negative")
	assertEq(t, "step 1 gazette", dec.Gazette.Num, "2024/19")

	if len(doc.UnitaryEvents) != 2 {
		t.Fatalf("got %d unitary events, want 2", len(doc.UnitaryEvents))
	}
	ev := doc.UnitaryEvents[0]
	assertEq(t, "event id", ev.ID, "EVT_UP_1431522")
	assertEq(t, "event type", ev.EventType, "new")
	assertEq(t, "event date", ev.Date, "20240405")
	assertEq(t, "event code", ev.Code, "0009701UREQ02")
	assertEq(t, "event text", ev.Texts[0].Text, "Decision on the request for unitary effect")
	assertEq(t, "event gazette", ev.Gazette.Num+" "+ev.Gazette.Date, "2024/19 20240508")

	parent := doc.Biblio.RelatedDocuments[0].Parents
	assertEq(t, "parent application", parent[0].Type+" "+parent[0].Country+parent[0].DocNumber, "application number EP20190787177")
	assertEq(t, "parent publication", parent[1].Country+parent[1].DocNumber, "EP3700366")
}

// TestParseRegister_Events checks the dossier events of the events constituent.
func TestParseRegister_Events(t *testing.T) {
	doc := parseRegisterFixture(t, "register-events.xml")
	assertLen(t, "Events", len(doc.Events), 55)
	ev := doc.Events[0]
	assertEq(t, "id", ev.ID, "EVT_110974905")
	assertEq(t, "date", ev.Date, "20220708")
	assertEq(t, "code", ev.Code, "0009250")
	assertEq(t, "text type", ev.Texts[0].Type, "DESCRIPTION")
	assertEq(t, "text", ev.Texts[0].Text, "Lapse of the patent in a contracting state")
}

// TestParseLegal_NamedFields checks the L-codes that have a name of their own.
func TestParseLegal_NamedFields(t *testing.T) {
	d, err := ParseLegal(string(readFixtureBytes(t, "legal.xml")))
	if err != nil {
		t.Fatal(err)
	}
	assertLen(t, "LegalEvents", len(d.LegalEvents), 69)
	assertEq(t, "requested publication", d.Publication[0].Country+d.Publication[0].DocNumber+d.Publication[0].Kind, "EP2400812A1")
	if d.TotalCount != 5 {
		t.Errorf("TotalCount = %d, want 5", d.TotalCount)
	}

	// A lapse in Belgium, reported as a reference to the national event MM.
	lapse := d.LegalEvents[58]
	assertEq(t, "lapse code", lapse.Code, "REG")
	assertEq(t, "lapse state", lapse.RefCountry, "BE")
	assertEq(t, "lapse national event", lapse.RefEventCode, "MM")
	assertEq(t, "lapse reason", lapse.RefEventDescription, "LAPSED BECAUSE OF NON-PAYMENT OF THE ANNUAL FEE")
	assertEq(t, "lapse influence", lapse.RefEventInfluence, "-")
	assertEq(t, "lapse effective date", lapse.EffectiveDate, "20200630")
	assertLen(t, "lapse pre lines", len(lapse.Pre), 3)
	assertEq(t, "pre line number", lapse.Pre[0].Line, "00001")

	// A renewal fee payment in Finland.
	fee := d.LegalEvents[65]
	assertEq(t, "fee code", fee.Code, "PGFP")
	assertEq(t, "fee state", fee.RefCountry, "FI")
	assertEq(t, "fee payment date", fee.PaymentDate, "20250620")
	assertEq(t, "fee year", fee.FeeYear, "16")

	// A change of owner.
	assertEq(t, "owner", d.LegalEvents[7].Owner, "9SOLUTIONS OY")

	// FieldList keeps the label and nesting of each L-code.
	var eff LegalField
	for _, f := range lapse.FieldList {
		if f.Code == "L525EP" {
			eff = f
		}
	}
	assertEq(t, "L525EP label", eff.Desc, "Effective DATE")
	assertEq(t, "L525EP parent", eff.Parent, "L500EP")
	assertEq(t, "L525EP value", eff.Value, "20200630")

	// The events are attributed to their family member.
	assertLen(t, "Members", len(d.Members), 1)
	assertEq(t, "member publication", d.Members[0].PublicationRefs[0].ByType("epodoc"), "EP2400812")
	assertLen(t, "member events", len(d.Members[0].LegalEvents), 69)
}

// TestParseFamily_FullMembers checks the member data ParseFamily did not expose before.
func TestParseFamily_FullMembers(t *testing.T) {
	// legal.xml is a family response with the legal constituent.
	withLegal, err := ParseFamily(string(readFixtureBytes(t, "legal.xml")))
	if err != nil {
		t.Fatal(err)
	}
	m := withLegal.Members[0]
	assertLen(t, "member legal events", len(m.LegalEvents), 69)
	assertEq(t, "priority claim kind", m.PriorityClaims[0].ClaimKind, "regional")
	assertEq(t, "priority kind (full)", m.Priorities[0].Kind, "regional")
	assertEq(t, "application representative", m.ApplicationRefs[0].IsRepresentative, "YES")
	assertEq(t, "publication epodoc", m.PublicationRefs[0].ByType("epodoc"), "EP2400812")
	assertEq(t, "requested kind", withLegal.Publication[0].Kind, "A1")

	withBiblio, err := ParseFamily(string(readFixtureBytes(t, "family-biblio.xml")))
	if err != nil {
		t.Fatal(err)
	}
	for i, fm := range withBiblio.Members {
		if len(fm.Documents) != 1 {
			t.Fatalf("member %d has %d exchange-documents, want 1", i, len(fm.Documents))
		}
		if fm.Title != fm.Documents[0].Title() {
			t.Errorf("member %d: Title %q differs from its document's %q", i, fm.Title, fm.Documents[0].Title())
		}
	}
}

// TestParseSearch_FullHits checks that a search with the biblio constituent keeps each hit's
// whole exchange-document.
func TestParseSearch_FullHits(t *testing.T) {
	raw := string(readFixtureBytes(t, "search.xml"))
	d, err := ParseSearch(raw)
	if err != nil {
		t.Fatal(err)
	}
	assertEq(t, "QuerySyntax", d.QuerySyntax, "CQL")
	if d.PublicationsCount != 3 || d.RangeBegin != 1 || d.RangeEnd != 3 {
		t.Errorf("PublicationsCount %d, range %d-%d; want 3, 1-3", d.PublicationsCount, d.RangeBegin, d.RangeEnd)
	}
	hit := d.Results[2]
	assertEq(t, "hit", hit.Country+hit.DocNumber+hit.Kind, "US20260369753A1")
	assertEq(t, "hit title", hit.Title, "BATTERY CELL")
	if hit.Document == nil {
		t.Fatal("hit has no Document")
	}
	assertEq(t, "hit application", hit.Document.ApplicationNumber("epodoc"), "US202619661747")
	assertLen(t, "hit applicants", len(hit.Document.Biblio.Applicants), 2)

	// The same hits, one BiblioData each.
	docs, err := ParseBiblioDocuments(raw)
	if err != nil {
		t.Fatal(err)
	}
	assertLen(t, "ParseBiblioDocuments", len(docs), 3)
	for i, b := range docs {
		assertEq(t, "document number", b.PatentNumber, d.Results[i].Country+d.Results[i].DocNumber+d.Results[i].Kind)
	}
}

// TestParseImageInquiry_PerPublication checks that an inquiry for a number without a kind
// code keeps the instances of each publication apart.
func TestParseImageInquiry_PerPublication(t *testing.T) {
	d, err := ParseImageInquiry(string(readFixtureBytes(t, "image-inquiry.xml")))
	if err != nil {
		t.Fatal(err)
	}
	assertEq(t, "requested", d.Publication[0].Type+" "+d.Publication[0].DocNumber, "epodoc EP2400812")
	assertLen(t, "Results", len(d.Results), 2)
	for i, kind := range []string{"A1", "B1"} {
		assertEq(t, "result kind", d.Results[i].Publication[0].Kind, kind)
		assertLen(t, kind+" instances", len(d.Results[i].DocumentInstances), 3)
	}
	full := d.Results[1].DocumentInstances[1]
	assertEq(t, "B1 instance", full.Description, "FullDocument")
	assertEq(t, "B1 link", full.Link, "published-data/images/EP/2400812/B1/fullimage")
	if full.NumberOfPages != 17 {
		t.Errorf("B1 pages = %d, want 17", full.NumberOfPages)
	}
	assertEq(t, "B1 system", full.System, "ops.epo.org")
	assertEq(t, "B1 claims section", full.Sections[1].Name+" "+full.Sections[1].StartPage, "CLAIMS 8")
}

// TestParseDescription_Identity checks that the document's identity is read from the
// publication-reference, where OPS states it.
func TestParseDescription_Identity(t *testing.T) {
	d, err := ParseDescription(string(readFixtureBytes(t, "description.xml")))
	if err != nil {
		t.Fatal(err)
	}
	assertEq(t, "identity", d.Country+d.DocNumber+d.Kind, "EP2400812A1")
	assertEq(t, "PatentNumber", d.PatentNumber, "EP2400812")
	assertEq(t, "Language", d.Language, "EN")
}

// TestRealNotFoundIsNotFoundError checks the response OPS really sends for an
// unknown document (recorded for EP9999999): a <fault> with the string code
// SERVER.EntityNotFound. It must surface as *NotFoundError, like the numeric
// fault and the <error> form do.
func TestRealNotFoundIsNotFoundError(t *testing.T) {
	body, err := os.ReadFile("testdata/error_404_entity_not_found.xml")
	if err != nil {
		t.Fatal(err)
	}
	got := (&Client{}).handleErrorResponse(http.StatusNotFound, nil, body)
	var notFound *NotFoundError
	if !errors.As(got, &notFound) {
		t.Fatalf("error = %T %v, want *NotFoundError", got, got)
	}
	if notFound.Message != "No results found" {
		t.Errorf("Message = %q, want the office's own message", notFound.Message)
	}
}
