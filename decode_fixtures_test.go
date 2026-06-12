package epo_ops

import (
	"embed"
	"encoding/xml"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// decode_fixtures_test.go is the rock-solid, network-free regression for every
// PRIMARY parsed EPO OPS endpoint, run in the normal build with no credentials
// over the committed testdata/*.xml fixtures (real recorded OPS responses).
//
// EPO OPS speaks XML, so this is the XML analogue of the strict-decode JSON
// fixture tests used by the JSON clients in this repo. Each primary endpoint is
// asserted three ways:
//
//	(a) ELEMENT COVERAGE  - decode the fixture into the parser's raw unmarshal
//	    struct, then assert every element/attribute present in the real XML is
//	    either captured by an xml tag or in the case's documented `unmodeled`
//	    boundary. encoding/xml silently drops anything unmapped, so this is the
//	    guard that a parser does not lose data it claims to capture, and that a
//	    newly appearing element cannot slip through unnoticed
//	    (see xml_completeness_test.go for the mechanics).
//	(b) GOLDEN ROUND-TRIP - re-marshal the decoded raw struct and assert that,
//	    for every element/attribute path the struct models, the re-marshaled
//	    values reproduce the fixture's values (whitespace-normalised, namespace
//	    prefixes ignored, order-insensitive). Proves the modeled projection is
//	    lossless (see assertRoundTrip).
//	(c) KEY FIELDS        - 3-6 specific values per endpoint asserted through the
//	    public Parse* function, proving the value parsed into the right field.
//
// The OPS parsers deliberately PROJECT: the narrow per-endpoint structs
// (abstractXML, biblioXML, ...) intentionally keep only the subtree their typed
// result needs; the full bibliographic-data tree is the job of the comprehensive
// ParseExchangeDocuments / ParseRegister parsers (also covered here, with a much
// tighter unmodeled boundary). The `unmodeled` / `unmodeledPrefixes` sets are the
// documented, reviewed edge of each projection.
//
// *Raw endpoints (return the raw XML string) and image/media endpoints (return
// bytes) are NOT parsed, so they get a minimal well-formed/non-empty contract
// check instead of the three layers (see TestRawAndMediaContracts).

//go:embed testdata/*.xml
var fixtureFS embed.FS

func readFixtureBytes(t *testing.T, name string) []byte {
	t.Helper()
	b, err := fixtureFS.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return b
}

// TestElementCoverage runs layer (a) for every primary endpoint: the fixture's
// real elements/attributes must all be captured by the raw unmarshal struct or
// listed in the documented projection boundary.
func TestElementCoverage(t *testing.T) {
	for _, c := range completenessCases(t) {
		t.Run(c.name, func(t *testing.T) {
			assertCompleteness(t, c)
		})
	}
}

// TestGoldenRoundTrip runs layer (b): re-marshaling the raw struct reproduces
// the fixture values for every path the struct models.
func TestGoldenRoundTrip(t *testing.T) {
	for _, c := range completenessCases(t) {
		t.Run(c.name, func(t *testing.T) {
			assertRoundTrip(t, c)
		})
	}
}

// completenessCases is the authoritative table mapping each primary parsed
// endpoint to its fixture, raw unmarshal struct and documented projection
// boundary. One row per primary parsed shape.
func completenessCases(t *testing.T) []completenessCase {
	t.Helper()
	// Subtrees the narrow per-endpoint parsers intentionally do NOT model; the
	// data they hold is captured instead by ParseExchangeDocuments / ParseRegister.
	// Listed as substrings so a whole subtree is one documented boundary line.
	narrowBiblioDrops := []string{
		"/application-reference",    // app number: ParseExchangeDocuments models it
		"/abstract",                 // abstract constituent: ParseAbstract / ParseExchangeDocuments
		"/priority-claims",          // priorities: ParseExchangeDocuments
		"/priority-claim",           // family priorities: ParseFamily models priority-claim differently
		"/references-cited",         // citations: ParseExchangeDocuments
		"/classification-ipc",       // legacy IPC: ParseExchangeDocuments
		"/inventors",                // inventors in family member biblio: ParseExchangeDocuments
		"classification-value",      // CPC value/generating-office: ParseExchangeDocuments
		"generating-office",         //
		"classification-scheme",     // CPC scheme office/name: ParseExchangeDocuments
		"@sequence",                 // positional sequence attrs (kept by comprehensive parsers)
		"@system",                   // exchange-document system attr
		"document-id-type",          // format discriminator (handled via data-format / multiple ids)
		"priority-linkage-type",     //
		"priority-active-indicator", // family active flag: ParseFamily uses element form, not on this path
		"is-representative",         //
		"@data-format",              // publication-reference data-format (DOCDB bulk form)
		"@office",                   // citation office attr
		"/nplcit",                   // non-patent citations: ParseExchangeDocuments
	}

	return []completenessCase{
		// --- Narrow per-endpoint parsers (deliberate projections) -------------
		{
			name: "abstract", data: readFixtureBytes(t, "abstract.xml"), raw: abstractXML{},
			// abstractXML keeps only exchange-document attrs + the abstract text.
			// The publication reference and parties are out of its scope (GetBiblio
			// / GetExchangeDocuments carry those).
			unmodeledPrefixes: []string{"/bibliographic-data"},
		},
		{
			name: "biblio", data: readFixtureBytes(t, "biblio.xml"), raw: biblioXML{},
			unmodeledPrefixes: narrowBiblioDrops,
		},
		{
			name: "claims", data: readFixtureBytes(t, "claims.xml"), raw: claimsXML{},
			// claimsXML drops the fulltext-document format/system attrs and the
			// publication-reference data-format attr.
			unmodeled: []string{
				"/world-patent-data/fulltext-documents/fulltext-document @fulltext-format",
				"/world-patent-data/fulltext-documents/fulltext-document @system",
				"/world-patent-data/fulltext-documents/fulltext-document/bibliographic-data/publication-reference @data-format",
			},
		},
		{
			name: "description", data: readFixtureBytes(t, "description.xml"), raw: descriptionXML{},
			unmodeled: []string{
				"/world-patent-data/fulltext-documents/fulltext-document @fulltext-format",
				"/world-patent-data/fulltext-documents/fulltext-document @lang",
				"/world-patent-data/fulltext-documents/fulltext-document @status",
				"/world-patent-data/fulltext-documents/fulltext-document @system",
				// description headings are not part of the paragraph projection.
				"/world-patent-data/fulltext-documents/fulltext-document/description/heading",
				"/world-patent-data/fulltext-documents/fulltext-document/description/heading @id",
			},
		},
		{
			name: "fulltext", data: readFixtureBytes(t, "description.xml"), raw: fulltextXML{},
			// fulltextXML is the lightweight envelope; the constituent bodies
			// (description paragraphs/headings) are parsed by the per-constituent
			// parsers it delegates to (ParseDescription, ParseClaims, ...).
			unmodeled: []string{
				"/world-patent-data/fulltext-documents/fulltext-document @fulltext-format",
				"/world-patent-data/fulltext-documents/fulltext-document @system",
			},
			unmodeledPrefixes: []string{"/description"},
		},
		{
			name: "family", data: readFixtureBytes(t, "family.xml"), raw: familyXML{},
			// family.xml is the plain (no-constituent) family list; fully modeled.
		},
		{
			name: "family_biblio", data: readFixtureBytes(t, "family-biblio.xml"), raw: familyXML{},
			// familyXML keeps each member's publication/application references,
			// priority claims, and the biblio title + applicants. The remaining
			// bibliographic-data subtree (classifications, inventors, citations,
			// priorities, abstracts) is the job of ParseExchangeDocuments.
			unmodeledPrefixes: append([]string{
				"/exchange-document @",                       // exchange-document attrs
				"/bibliographic-data/publication-reference",  //
				"/bibliographic-data/classifications-ipcr",   //
				"/bibliographic-data/patent-classifications", //
			}, narrowBiblioDrops...),
		},
		{
			name: "legal", data: readFixtureBytes(t, "legal.xml"), raw: legalXML{},
			skipRoundTrip: "legalEventXML captures L-codes via a custom UnmarshalXML into a map (no reflective MarshalXML); losslessness covered by element coverage, key fields and xml_dynamic_test.go",
			// legalEventXML captures every L-code child dynamically via a custom
			// UnmarshalXML into a map (so reflection sees no xml tags for them); the
			// L*EP elements/attrs and the legal/pre lines therefore appear as
			// "unmodeled" here but ARE captured at runtime (see xml_dynamic_test.go).
			// The per-member publication/application/priority references are out of
			// the legal projection (ParseFamily / ParseExchangeDocuments carry them).
			unmodeledPrefixes: []string{
				"/legal/L",     // dynamic L-code fields (captured via UnmarshalXML map)
				"/legal/pre @", // raw <pre> line attrs (captured into Pre)
				"@legal", "@total-result-count",
				"/family-member/application-reference",
				"/family-member/publication-reference",
				"/family-member/priority-claim",
				"/publication-reference/document-id @document-id-type",
			},
		},
		{
			name: "search", data: readFixtureBytes(t, "search.xml"), raw: searchXML{},
			// searchXML keeps the search envelope + each hit's identity and title.
			// The nested publication-reference inside each exchange-document's
			// bibliographic-data (the hit already carries country/doc-number/kind as
			// attributes) and the query @syntax attr are not modeled.
			unmodeled: []string{
				"/world-patent-data/biblio-search/query @syntax",
			},
			unmodeledPrefixes: []string{"/bibliographic-data/publication-reference"},
		},
		{
			name: "equivalents", data: readFixtureBytes(t, "equivalents.xml"), raw: equivalentsXML{},
			unmodeled: []string{
				"/world-patent-data/equivalents-inquiry/inquiry-result/publication-reference/document-id @document-id-type",
				"/world-patent-data/equivalents-inquiry/publication-reference/document-id @document-id-type",
			},
		},
		{
			name: "register_events", data: readFixtureBytes(t, "register-events.xml"), raw: registerEventsXML{},
			// registerEventsXML models the events + statuses; the register-search
			// pagination (range), the produced-by/lang/dtd-version bookkeeping attrs
			// and the query @syntax are not part of the events projection.
			unmodeled: []string{
				"/world-patent-data/register-search/query @syntax",
				"/world-patent-data/register-search/range",
				"/world-patent-data/register-search/range @begin",
				"/world-patent-data/register-search/range @end",
				"/world-patent-data/register-search/register-documents @produced-by",
				"/world-patent-data/register-search/register-documents/register-document @date-produced",
				"/world-patent-data/register-search/register-documents/register-document @dtd-version",
				"/world-patent-data/register-search/register-documents/register-document @lang",
				"/world-patent-data/register-search/register-documents/register-document @produced-by",
			},
		},
		{
			name: "number_conversion", data: readFixtureBytes(t, "number-conversion.xml"), raw: numberConversionXML{},
			// numberConversionXML reads the standardised OUTPUT document-id; the
			// echoed input reference and the response <meta> bookkeeping are not
			// part of the conversion result.
			unmodeledPrefixes: []string{
				"/world-patent-data/meta",
				"/world-patent-data/standardization/input",
			},
		},
		{
			name: "classification_schema", data: readFixtureBytes(t, "classification-schema.xml"), raw: classificationSchemaXML{},
			// classificationSchemaXML keeps the symbol, title-part text, level and
			// has-children for the item tree. CPC's editorial metadata (revision
			// dates, sort keys, notes/warnings, definitions, cross-reference links)
			// is intentionally out of scope for a schema-navigation result.
			unmodeledPrefixes: []string{
				"/class-scheme @export-date",
				"/classification-item @", // editorial item attrs (date-revised, status, ...)
				"/class-title @date-revised",
				"/title-part/explanation", // explanatory cross-references
				"/meta-data",
				"/notes-and-warnings",
			},
		},
		{
			name: "image_inquiry", data: readFixtureBytes(t, "image-inquiry.xml"), raw: imageInquiryXML{},
			// fully modeled.
		},
		// --- Comprehensive parsers (tight boundary) ---------------------------
		{
			name: "exchange_biblio", data: readFixtureBytes(t, "biblio.xml"), raw: ExchangeDocument{}, root: "exchange-document",
			// ExchangeDocument is the INID-complete biblio parser. Its only drop on
			// this fixture is the positional @sequence attr on classification-ipcr
			// (the IPCR text itself is captured); documented as not load-bearing.
			unmodeled: []string{
				"/exchange-document/bibliographic-data/classifications-ipcr/classification-ipcr @sequence",
			},
		},
		{
			name: "exchange_family", data: readFixtureBytes(t, "family-biblio.xml"), raw: ExchangeDocument{}, root: "exchange-document",
			unmodeled: []string{
				"/exchange-document/bibliographic-data/classifications-ipcr/classification-ipcr @sequence",
			},
		},
		{
			name: "register_biblio", data: readFixtureBytes(t, "register-biblio.xml"), raw: RegisterDocument{}, root: "register-document",
			skipRoundTrip: "RegisterDocument embeds MilestoneDates, a custom-UnmarshalXML map encoding/xml cannot re-marshal; losslessness covered by element coverage, key fields and register_full_test.go",
			// RegisterDocument captures the register record's references, parties
			// (NAMES), classifications, citations, statuses, term-of-grant, milestone
			// dates and search reports. The documented boundary below covers the
			// register's editorial bookkeeping and the data the typed result does not
			// surface: full postal addressbooks, the change-date/change-gazette-num
			// audit attrs the register stamps on every element, nationality/app-type
			// party attrs, the PCT region nesting, and the document-level produced-by
			// metadata. None of it is silently lost beyond these lines.
			unmodeledPrefixes: []string{
				"/addressbook", // full postal address (only name is surfaced)
				"@change-date", "@change-gazette-num",
				"@date-produced", "@dtd-version", "@produced-by",
				"@app-type", "@designation", "@cdsid", "@sequence", "@id", "@lang",
				"/nationality", "/residence", // party nationality/residence detail
				"/designation-pct",      // PCT designation nesting (EPC contracting handled)
				"/invention-title @",    // title change attrs
				"/bibliographic-data @", // biblio-level status/country/id/lang attrs
				"/dates-rights-effective/first-examination-report-despatched", // extra milestone variants (map captures present ones)
				"/dates-rights-effective/request-for-examination",
				"/application-reference", // register app-reference (publication carries identity)
				"/classifications-ipcr",  // register IPCR (CPC modeled)
				"/language-of-filing",    //
				"/opposition-data/opposition-not-filed/date",
				"/references-cited/citation @", // citation office/id/phase attrs
				"/patcit @url", "/patcit/text", // citation patcit url/text variants
				"/search-report-information @", // search report editorial attrs
				"/date-search-report-mailed",
				"/search-report-publication/document-id @lang",
				"/term-of-grant/lapsed-in-country/country", // term-of-grant detailed form
				"/term-of-grant/lapsed-in-country/date",
				"/publication-reference @change-gazette-num",
				"/publication-reference/document-id @lang",
				"/publication-reference/document-id/date",
				"/citation/patcit/document-id/country",
				"/citation/patcit/document-id/doc-number",
				"/category", // register citation category form
			},
		},
		{
			name: "register_procedural", data: readFixtureBytes(t, "register-procedural-steps.xml"), raw: RegisterDocument{}, root: "register-document",
			skipRoundTrip: "RegisterDocument embeds MilestoneDates, a custom-UnmarshalXML map encoding/xml cannot re-marshal; losslessness covered by element coverage, key fields and register_full_test.go",
			unmodeledPrefixes: []string{
				"@date-produced", "@dtd-version", "@produced-by", "@lang", "@id",
				"/procedural-step @", // procedural-step bookkeeping attrs (id/phase captured)
				// secondary step detail not surfaced in the typed ProceduralStep:
				"/time-limit",
				"/procedural-step-date @step-date-type",
				"/procedural-step-text @step-text-type",
			},
		},
		{
			name: "register_unip", data: readFixtureBytes(t, "register-unip.xml"), raw: RegisterDocument{}, root: "register-document",
			skipRoundTrip: "RegisterDocument embeds MilestoneDates, a custom-UnmarshalXML map encoding/xml cannot re-marshal; losslessness covered by element coverage, key fields and register_full_test.go",
			unmodeledPrefixes: []string{
				"/addressbook",
				"@change-date", "@change-gazette-num",
				"@date-produced", "@dtd-version", "@produced-by",
				"@app-type", "@designation", "@cdsid", "@sequence", "@id", "@lang",
				"/nationality", "/residence",
				"/designation-pct",
				"/invention-title @",
				"/bibliographic-data @",
				"/dates-rights-effective/first-examination-report-despatched",
				"/dates-rights-effective/request-for-examination",
				"/application-reference",
				"/classifications-ipcr",
				"/language-of-filing",
				"/opposition-data/opposition-not-filed/date",
				"/references-cited/citation @",
				"/patcit @url", "/patcit/text",
				"/search-report-information @",
				"/date-search-report-mailed",
				"/search-report-publication/document-id @lang",
				"/term-of-grant/lapsed-in-country/country",
				"/term-of-grant/lapsed-in-country/date",
				"/publication-reference @change-gazette-num",
				"/publication-reference/document-id @lang",
				"/publication-reference/document-id/date",
				"/citation/patcit/document-id/country",
				"/citation/patcit/document-id/doc-number",
				"/category",
				"/unitary-patent-data", // unitary-patent constituent (typed result is generic register record)
			},
		},
	}
}

// assertRoundTrip re-marshals the decoded raw struct and asserts that, for every
// element/attribute path the struct models, the re-marshaled values reproduce the
// fixture's values. It proves the modeled projection is lossless: a value that
// decoded but did not survive a re-marshal (a wrong/ambiguous tag) is caught.
//
// Values are compared as multisets per modeled path, whitespace-normalised, with
// namespace prefixes ignored. Paths the struct does NOT model are out of scope
// here (element coverage already governs them).
func assertRoundTrip(t *testing.T, c completenessCase) {
	t.Helper()
	if c.skipRoundTrip != "" {
		t.Skipf("round-trip not applicable: %s", c.skipRoundTrip)
	}

	rt := reflect.TypeOf(c.raw)
	out := reflect.New(rt).Interface()

	// Decode from the right root: comprehensive parsers decode a sub-element.
	data := c.data
	if c.root != "" {
		sub := extractFirstElement(string(c.data), c.root)
		if sub == "" {
			t.Fatalf("%s: could not locate <%s> in fixture", c.name, c.root)
		}
		data = []byte(sub)
	}
	if err := xml.Unmarshal(data, out); err != nil {
		t.Fatalf("%s: unmarshal: %v", c.name, err)
	}
	remarshaled, err := xml.Marshal(out)
	if err != nil {
		t.Fatalf("%s: re-marshal: %v", c.name, err)
	}

	// Compare values keyed by ROOT-RELATIVE path (the leading root segment is
	// dropped) so the comparison is unaffected by the root element's name. A raw
	// struct with no XMLName re-marshals under its Go type name (e.g.
	// <ExchangeDocument>), which differs from the fixture's <exchange-document>;
	// stripping the root makes both sides line up on the modeled subtree.
	schemaEls, _ := schemaPaths(rt, c.root)
	schemaRel := stripRoots(schemaEls)
	fixVals := stripRootVals(pathValues(c.data, c.root))
	rtVals := stripRootVals(pathValues(remarshaled, ""))

	var diffs []string
	for path, want := range fixVals {
		if !schemaRel[path] {
			continue // not modeled: governed by element coverage, not round-trip
		}
		got := rtVals[path]
		if !equalMultiset(want, got) {
			diffs = append(diffs, path+": fixture="+strings.Join(want, "|")+" remarshaled="+strings.Join(got, "|"))
		}
	}
	sort.Strings(diffs)
	if len(diffs) > 0 {
		// Cap the report so a structural mismatch does not flood the log.
		if len(diffs) > 20 {
			diffs = append(diffs[:20], "...")
		}
		t.Errorf("%s: %d modeled element path(s) did not round-trip losslessly:\n  %s",
			c.name, len(diffs), strings.Join(diffs, "\n  "))
	}
}

// pathValues walks XML and returns, per rooted element path (local names), the
// multiset of non-empty trimmed character values directly under that element.
func pathValues(data []byte, startLocal string) map[string][]string {
	out := map[string][]string{}
	dec := xml.NewDecoder(strings.NewReader(string(data)))
	var stack []string
	var text strings.Builder
	capturing := startLocal == ""
	depth := 0
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch se := tok.(type) {
		case xml.StartElement:
			local := se.Name.Local
			if !capturing {
				if local == startLocal {
					capturing = true
					stack = nil
					depth = 1
				} else {
					continue
				}
			} else {
				depth++
			}
			text.Reset()
			stack = append(stack, "/"+local)
		case xml.CharData:
			if capturing {
				text.Write(se)
			}
		case xml.EndElement:
			if !capturing {
				continue
			}
			path := strings.Join(stack, "")
			if v := strings.TrimSpace(text.String()); v != "" {
				out[path] = append(out[path], v)
			}
			text.Reset()
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			depth--
			if startLocal != "" && depth == 0 {
				return out
			}
		}
	}
	return out
}

// extractFirstElement returns the serialized XML of the first element with the
// given local name (with its subtree), or "" if not found.
func extractFirstElement(data, local string) string {
	dec := xml.NewDecoder(strings.NewReader(data))
	for {
		tok, err := dec.Token()
		if err != nil {
			return ""
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != local {
			continue
		}
		var buf strings.Builder
		enc := xml.NewEncoder(&buf)
		if err := enc.EncodeToken(se); err != nil {
			return ""
		}
		depth := 1
		for depth > 0 {
			tk, err := dec.Token()
			if err != nil {
				return ""
			}
			switch tk.(type) {
			case xml.StartElement:
				depth++
			case xml.EndElement:
				depth--
			}
			if err := enc.EncodeToken(tk); err != nil {
				return ""
			}
		}
		if err := enc.Flush(); err != nil {
			return ""
		}
		return buf.String()
	}
}

// stripRoot drops the first "/segment" of a rooted path, leaving the path
// relative to the root element (e.g. "/a/b/c" -> "/b/c", "/a" -> "").
func stripRoot(p string) string {
	if len(p) == 0 || p[0] != '/' {
		return p
	}
	i := strings.IndexByte(p[1:], '/')
	if i < 0 {
		return ""
	}
	return p[1+i:]
}

func stripRoots(in map[string]bool) map[string]bool {
	out := make(map[string]bool, len(in))
	for p := range in {
		out[stripRoot(p)] = true
	}
	return out
}

func stripRootVals(in map[string][]string) map[string][]string {
	out := make(map[string][]string, len(in))
	for p, v := range in {
		rp := stripRoot(p)
		out[rp] = append(out[rp], v...)
	}
	return out
}

func equalMultiset(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	ac := append([]string(nil), a...)
	bc := append([]string(nil), b...)
	sort.Strings(ac)
	sort.Strings(bc)
	for i := range ac {
		if ac[i] != bc[i] {
			return false
		}
	}
	return true
}

// TestKeyFields runs layer (c): targeted value assertions through the public
// Parse* functions, proving each value parsed into the right typed field. The
// narrow biblio/abstract/claims/image parsers also have dedicated value tests in
// xml_test.go; the cases here cover the remaining primary endpoints plus the
// comprehensive parsers, and the newly added fixtures.
func TestKeyFields(t *testing.T) {
	t.Run("number_conversion", func(t *testing.T) {
		d, err := ParseNumberConversion(string(readFixtureBytes(t, "number-conversion.xml")))
		if err != nil {
			t.Fatal(err)
		}
		assertEq(t, "InputFormat", d.InputFormat, "docdb")
		assertEq(t, "OutputFormat", d.OutputFormat, "epodoc")
		assertEq(t, "DocNumber", d.DocNumber, "EP2400812")
		assertEq(t, "Kind", d.Kind, "A1")
		assertEq(t, "Date", d.Date, "20111228")
	})

	t.Run("equivalents", func(t *testing.T) {
		d, err := ParseEquivalents(string(readFixtureBytes(t, "equivalents.xml")))
		if err != nil {
			t.Fatal(err)
		}
		assertEq(t, "PatentNumber", d.PatentNumber, "EP2400812")
		assertLen(t, "Equivalents", len(d.Equivalents), 3)
		assertEq(t, "Equivalents[0].Country", d.Equivalents[0].Country, "US")
		assertEq(t, "Equivalents[0].DocNumber", d.Equivalents[0].DocNumber, "2012057518")
	})

	t.Run("classification_schema", func(t *testing.T) {
		d, err := ParseClassificationSchema(string(readFixtureBytes(t, "classification-schema.xml")))
		if err != nil {
			t.Fatal(err)
		}
		assertEq(t, "Symbol", d.Symbol, "H04W")
		assertEq(t, "Title", d.Title, "WIRELESS COMMUNICATION NETWORKS")
		assertEq(t, "SchemeType", d.SchemeType, "cpc")
		if d.Level != 5 {
			t.Errorf("Level = %d, want 5", d.Level)
		}
	})

	t.Run("register_events", func(t *testing.T) {
		d, err := ParseRegisterEvents(string(readFixtureBytes(t, "register-events.xml")))
		if err != nil {
			t.Fatal(err)
		}
		assertEq(t, "PatentNumber", d.PatentNumber, "EP2400812")
		assertEq(t, "Query", d.Query, "publication=EP2400812")
		assertLen(t, "Events", len(d.Events), 55)
		assertLen(t, "Statuses", len(d.Statuses), 4)
		assertEq(t, "Events[0].Date", d.Events[0].Date, "20111228")
		assertEq(t, "Events[0].EventCode", d.Events[0].EventCode, "0009012")
		assertEq(t, "Events[0].Category", d.Events[0].Category, "publication")
	})

	t.Run("fulltext", func(t *testing.T) {
		d, err := ParseFulltext(string(readFixtureBytes(t, "description.xml")))
		if err != nil {
			t.Fatal(err)
		}
		assertEq(t, "Country", d.Country, "EP")
		assertEq(t, "DocNumber", d.DocNumber, "2400812")
		assertEq(t, "Kind", d.Kind, "B1")
		if d.Description == nil {
			t.Error("Description constituent not parsed")
		}
	})

	t.Run("search", func(t *testing.T) {
		d, err := ParseSearch(string(readFixtureBytes(t, "search.xml")))
		if err != nil {
			t.Fatal(err)
		}
		if d.TotalCount != 1523 {
			t.Errorf("TotalCount = %d, want 1523", d.TotalCount)
		}
		assertLen(t, "Results", len(d.Results), 2)
		assertEq(t, "Results[0].Country", d.Results[0].Country, "EP")
		assertEq(t, "Results[0].DocNumber", d.Results[0].DocNumber, "2400812")
		assertEq(t, "Results[0].Title", d.Results[0].Title, "Battery Management System")
		assertEq(t, "Results[0].FamilyID", d.Results[0].FamilyID, "43088294")
	})

	t.Run("family", func(t *testing.T) {
		d, err := ParseFamily(string(readFixtureBytes(t, "family.xml")))
		if err != nil {
			t.Fatal(err)
		}
		assertEq(t, "PatentNumber", d.PatentNumber, "EP2400812")
		assertLen(t, "Members", len(d.Members), 3)
		if got := strings.Join(d.Countries, ","); got != "EP,US,WO" {
			t.Errorf("Countries = %q, want EP,US,WO", got)
		}
	})

	t.Run("family_biblio", func(t *testing.T) {
		d, err := ParseFamily(string(readFixtureBytes(t, "family-biblio.xml")))
		if err != nil {
			t.Fatal(err)
		}
		assertLen(t, "Members", len(d.Members), 6)
		assertEq(t, "FamilyID", d.FamilyID, "23743220")
		withTitle := 0
		for _, m := range d.Members {
			if m.Title != "" {
				withTitle++
			}
		}
		if withTitle == 0 {
			t.Error("no family member carried a biblio title")
		}
	})

	t.Run("legal", func(t *testing.T) {
		d, err := ParseLegal(string(readFixtureBytes(t, "legal.xml")))
		if err != nil {
			t.Fatal(err)
		}
		assertEq(t, "PatentNumber", d.PatentNumber, "EP2400812")
		if len(d.LegalEvents) < 60 {
			t.Errorf("LegalEvents = %d, want >= 60", len(d.LegalEvents))
		}
		e0 := d.LegalEvents[0]
		assertEq(t, "LegalEvents[0].Code", e0.Code, "AK")
		assertEq(t, "LegalEvents[0].Country", e0.Country, "EP")
		// L507EP is well beyond the old fixed L050EP cap: proves dynamic capture.
		if e0.Fields["L507EP"] == "" {
			t.Error("dynamic L-code L507EP not captured")
		}
	})

	t.Run("exchange_documents", func(t *testing.T) {
		docs, err := ParseExchangeDocuments(string(readFixtureBytes(t, "biblio.xml")))
		if err != nil {
			t.Fatal(err)
		}
		assertLen(t, "docs", len(docs), 1)
		d := docs[0]
		assertEq(t, "Country", d.Country, "EP")
		assertEq(t, "DocNumber", d.DocNumber, "2400812")
		assertEq(t, "PublicationNumber", d.PublicationNumber(), "2400812")
		// Multi-category citation is now captured losslessly (regression guard).
		var multi *Citation
		for i := range d.Biblio.Citations {
			if len(d.Biblio.Citations[i].Categories) > 1 {
				multi = &d.Biblio.Citations[i]
				break
			}
		}
		if multi == nil {
			t.Fatal("expected a citation with multiple categories")
		}
		if got := strings.Join(multi.Categories, ","); got != "Y,A" {
			t.Errorf("multi-category Categories = %q, want Y,A", got)
		}
		if got := strings.Join(multi.RelClaimsAll, ","); got != "1-12,15,13,14" {
			t.Errorf("multi-category RelClaimsAll = %q, want 1-12,15,13,14", got)
		}
	})

	t.Run("register", func(t *testing.T) {
		docs, err := ParseRegister(string(readFixtureBytes(t, "register-biblio.xml")))
		if err != nil {
			t.Fatal(err)
		}
		assertLen(t, "docs", len(docs), 1)
		d := docs[0]
		assertLen(t, "Statuses", len(d.Statuses), 4)
		if len(d.Biblio.TermsOfGrant) == 0 {
			t.Error("no term-of-grant captured")
		}
		if len(d.Biblio.MilestoneDates) == 0 {
			t.Error("no milestone dates captured")
		}
		if len(d.Biblio.SearchReports) == 0 {
			t.Error("no search reports captured")
		}
	})

	t.Run("register_unip", func(t *testing.T) {
		docs, err := ParseRegister(string(readFixtureBytes(t, "register-unip.xml")))
		if err != nil {
			t.Fatal(err)
		}
		if len(docs) == 0 {
			t.Fatal("no register-document parsed from unip fixture")
		}
		if len(docs[0].Statuses) == 0 {
			t.Error("no statuses captured from unip register record")
		}
	})

	t.Run("image_inquiry", func(t *testing.T) {
		d, err := ParseImageInquiry(string(readFixtureBytes(t, "image-inquiry.xml")))
		if err != nil {
			t.Fatal(err)
		}
		assertLen(t, "DocumentInstances", len(d.DocumentInstances), 3)
	})
}

func assertEq(t *testing.T, field, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %q, want %q", field, got, want)
	}
}

func assertLen(t *testing.T, field string, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("len(%s) = %d, want %d", field, got, want)
	}
}

// TestRawAndMediaContracts documents the non-parsed endpoint families and pins
// their minimal contract.
//
//   - *Raw endpoints return the raw OPS XML string verbatim (no parsing), so the
//     only meaningful library-side guarantee is that the bytes are well-formed,
//     non-empty XML. Decoding every committed XML fixture as a generic token
//     stream proves exactly that for the recorded responses these endpoints
//     return; the per-endpoint live calls are covered by the integration suite.
//   - Image / media endpoints (GetImage, GetImagePOST, GetClassificationMedia)
//     return raw bytes (TIFF / PDF / PNG); the contract is non-empty bytes with a
//     recognisable format magic, asserted live in the integration suite. There is
//     no committed binary fixture (images are large and licensed), so the magic
//     check is documented here and exercised in TestIntegrationGetImage et al.
func TestRawAndMediaContracts(t *testing.T) {
	entries, err := fixtureFS.ReadDir("testdata")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	xmlCount := 0
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".xml") {
			continue
		}
		xmlCount++
		t.Run(e.Name(), func(t *testing.T) {
			data := readFixtureBytes(t, e.Name())
			if len(data) == 0 {
				t.Fatal("empty fixture")
			}
			dec := xml.NewDecoder(strings.NewReader(string(data)))
			for {
				_, err := dec.Token()
				if err != nil {
					if err.Error() == "EOF" {
						break
					}
					t.Fatalf("not well-formed XML: %v", err)
				}
			}
		})
	}
	if xmlCount < 13 {
		t.Errorf("expected >=13 committed XML fixtures, found %d", xmlCount)
	}
}
