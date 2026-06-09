package epo_ops

import (
	"context"
	"encoding/xml"
	"io"
	"strings"
)

// This file provides a comprehensive parser for the OPS <exchange-document> element,
// covering the full bibliographic-data subtree plus abstracts (the data an INID-level
// client needs). It is modelled on the vendored OPS RWS schema (schema/ops/) and validated
// against recorded responses. The document body (claims, description, MathML, chemistry) is
// out of scope here - those have dedicated parsers (ParseClaims, ParseDescription).
//
// OPS REST distinguishes number formats by the document-id-type attribute on each
// <document-id> (docdb | epodoc | original); the bulk DOCDB product instead puts data-format
// on the reference element. The types below tolerate both.

// DocumentID is one <document-id> within a reference, citation or priority claim.
type DocumentID struct {
	Type      string `xml:"document-id-type,attr"`
	Country   string `xml:"country"`
	DocNumber string `xml:"doc-number"`
	Kind      string `xml:"kind"`
	Name      string `xml:"name"`
	Date      string `xml:"date"`
}

// Reference is a publication-reference or application-reference: one element carrying one
// document-id per available format.
type Reference struct {
	Sequence    string       `xml:"sequence,attr"`
	DataFormat  string       `xml:"data-format,attr"`
	DocID       string       `xml:"doc-id,attr"`
	DocumentIDs []DocumentID `xml:"document-id"`
}

// ByType returns the doc-number for a given format (document-id-type), tolerating the bulk
// DOCDB serialization where the format lives on the reference's data-format attribute.
func (r Reference) ByType(format string) string {
	for _, d := range r.DocumentIDs {
		if d.Type == format && d.DocNumber != "" {
			return d.DocNumber
		}
	}
	if r.DataFormat == format && len(r.DocumentIDs) > 0 {
		return r.DocumentIDs[0].DocNumber
	}
	return ""
}

func pickRef(refs []Reference, prefer ...string) string {
	for _, f := range prefer {
		for _, r := range refs {
			if v := r.ByType(f); v != "" {
				return v
			}
		}
	}
	return ""
}

// Applicant / Inventor / Agent are repeated once per data-format (docdb, epodoc).
type Applicant struct {
	Sequence   string `xml:"sequence,attr"`
	DataFormat string `xml:"data-format,attr"`
	Name       string `xml:"applicant-name>name"`
	Residence  string `xml:"residence>country"`
}

// Inventor is one inventor party (repeated per data-format).
type Inventor struct {
	Sequence   string `xml:"sequence,attr"`
	DataFormat string `xml:"data-format,attr"`
	Name       string `xml:"inventor-name>name"`
	Residence  string `xml:"residence>country"`
}

// Agent is one representative/agent.
type Agent struct {
	Sequence string `xml:"sequence,attr"`
	RepType  string `xml:"rep-type,attr"`
	Name     string `xml:"addressbook>name"`
}

// PatentClassification is one CPC (or other unified-scheme) classification.
type PatentClassification struct {
	Sequence string `xml:"sequence,attr"`
	Scheme   struct {
		Office string `xml:"office,attr"`
		Name   string `xml:"scheme,attr"`
	} `xml:"classification-scheme"`
	Section          string `xml:"section"`
	Class            string `xml:"class"`
	Subclass         string `xml:"subclass"`
	MainGroup        string `xml:"main-group"`
	Subgroup         string `xml:"subgroup"`
	Value            string `xml:"classification-value"`
	GeneratingOffice string `xml:"generating-office"`
}

// Symbol renders the classification as e.g. "H04W84/20", or "" if unpopulated.
func (p PatentClassification) Symbol() string {
	if p.Section == "" {
		return ""
	}
	s := p.Section + p.Class + p.Subclass + p.MainGroup
	if p.Subgroup != "" {
		s += "/" + p.Subgroup
	}
	return s
}

// NationalClassification is a classification-national entry.
type NationalClassification struct {
	Country string   `xml:"country"`
	Main    string   `xml:"main-classification"`
	Further []string `xml:"further-classification"`
	Text    []string `xml:"text"`
}

// Priority is a priority-claims/priority-claim entry.
type Priority struct {
	Sequence    string       `xml:"sequence,attr"`
	Kind        string       `xml:"kind,attr"`
	DataFormat  string       `xml:"data-format,attr"`
	Active      string       `xml:"priority-active-indicator"`
	DocumentIDs []DocumentID `xml:"document-id"`
}

// Patcit is a cited patent document.
type Patcit struct {
	Num         string       `xml:"num,attr"`
	DnumType    string       `xml:"dnum-type,attr"`
	Text        string       `xml:"text"`
	DocumentIDs []DocumentID `xml:"document-id"`
}

// Number assembles the cited patent number for the first matching format.
func (p Patcit) Number(prefer ...string) string {
	if len(prefer) == 0 {
		prefer = []string{"docdb", "epodoc"}
	}
	for _, f := range prefer {
		for _, d := range p.DocumentIDs {
			if d.Type == f {
				return d.Country + d.DocNumber + d.Kind
			}
		}
	}
	return ""
}

// Nplcit is a cited non-patent-literature document.
type Nplcit struct {
	Num  string `xml:"num,attr"`
	Text string `xml:"text"`
}

// Citation is one references-cited/citation: a cited document with its search category
// (X / Y / A ...), the claims it was applied to, and the relevant passages.
type Citation struct {
	Sequence    string   `xml:"sequence,attr"`
	CitedPhase  string   `xml:"cited-phase,attr"`
	CitedBy     string   `xml:"cited-by,attr"`
	Category    string   `xml:"category"`
	RelClaims   string   `xml:"rel-claims"`
	RelPassages []string `xml:"rel-passage>passage"`
	Patcit      *Patcit  `xml:"patcit"`
	Nplcit      *Nplcit  `xml:"nplcit"`
}

// DesignationOfStates captures designated contracting/extension/validation states.
type DesignationOfStates struct {
	EPCContracting []string `xml:"designation-epc>contracting-states>country"`
	EPCExtension   []string `xml:"designation-epc>extension-states>country"`
	PCTRegional    []string `xml:"designation-pct>regional>country"`
	PCTNational    []string `xml:"designation-pct>national>country"`
	Contracting    []string `xml:"contracting-states>country"`
}

// Abstract is one <abstract> constituent.
type Abstract struct {
	Lang       string   `xml:"lang,attr"`
	Paragraphs []string `xml:"p"`
}

// Text joins the abstract paragraphs into a single trimmed string.
func (a Abstract) Text() string {
	return strings.TrimSpace(strings.Join(a.Paragraphs, "\n"))
}

// Title is one invention-title in a given language.
type Title struct {
	Lang string `xml:"lang,attr"`
	Text string `xml:",chardata"`
}

// BibliographicData is the full bibliographic-data subtree.
type BibliographicData struct {
	PublicationRefs       []Reference              `xml:"publication-reference"`
	ApplicationRefs       []Reference              `xml:"application-reference"`
	Titles                []Title                  `xml:"invention-title"`
	Applicants            []Applicant              `xml:"parties>applicants>applicant"`
	Inventors             []Inventor               `xml:"parties>inventors>inventor"`
	Agents                []Agent                  `xml:"parties>agents>agent"`
	IPCR                  []string                 `xml:"classifications-ipcr>classification-ipcr>text"`
	IPC                   []string                 `xml:"classification-ipc>text"`
	National              []NationalClassification `xml:"classification-national"`
	CPC                   []PatentClassification   `xml:"patent-classifications>patent-classification"`
	Priorities            []Priority               `xml:"priority-claims>priority-claim"`
	Citations             []Citation               `xml:"references-cited>citation"`
	Designations          DesignationOfStates      `xml:"designation-of-states"`
	LanguageOfFiling      string                   `xml:"language-of-filing"`
	LanguageOfPublication string                   `xml:"language-of-publication"`
}

// ExchangeDocument is one <exchange-document> with its bibliographic data and abstracts.
type ExchangeDocument struct {
	DocID     string            `xml:"doc-id,attr"`
	System    string            `xml:"system,attr"`
	FamilyID  string            `xml:"family-id,attr"`
	Country   string            `xml:"country,attr"`
	DocNumber string            `xml:"doc-number,attr"`
	Kind      string            `xml:"kind,attr"`
	Status    string            `xml:"status,attr"`
	DatePubl  string            `xml:"date-publ,attr"`
	Biblio    BibliographicData `xml:"bibliographic-data"`
	Abstracts []Abstract        `xml:"abstract"`
}

// PublicationNumber returns the publication number for the first preferred format
// (default docdb then epodoc).
func (e ExchangeDocument) PublicationNumber(prefer ...string) string {
	if len(prefer) == 0 {
		prefer = []string{"docdb", "epodoc"}
	}
	return pickRef(e.Biblio.PublicationRefs, prefer...)
}

// ApplicationNumber returns the application number, preferring the originating-office
// ("original") number national offices look up by, then docdb.
func (e ExchangeDocument) ApplicationNumber(prefer ...string) string {
	if len(prefer) == 0 {
		prefer = []string{"original", "docdb"}
	}
	return pickRef(e.Biblio.ApplicationRefs, prefer...)
}

// Title returns the invention title, preferring the given languages (default "en").
func (e ExchangeDocument) Title(langs ...string) string {
	if len(langs) == 0 {
		langs = []string{"en"}
	}
	for _, l := range langs {
		for _, t := range e.Biblio.Titles {
			if strings.EqualFold(t.Lang, l) && strings.TrimSpace(t.Text) != "" {
				return strings.TrimSpace(t.Text)
			}
		}
	}
	for _, t := range e.Biblio.Titles {
		if strings.TrimSpace(t.Text) != "" {
			return strings.TrimSpace(t.Text)
		}
	}
	return ""
}

// GetExchangeDocuments retrieves bibliographic data and parses the full exchange-document(s).
// It is the comprehensive counterpart to GetBiblio (which returns the lighter BiblioData):
// it captures application/publication references in every format, parties incl. agents,
// classifications, priorities, citations and designations. Works for any published-data or
// family response.
func (c *Client) GetExchangeDocuments(ctx context.Context, refType, format, number string) ([]ExchangeDocument, error) {
	raw, err := c.GetBiblioRaw(ctx, refType, format, number)
	if err != nil {
		return nil, err
	}
	return ParseExchangeDocuments(raw)
}

// ParseExchangeDocuments extracts every <exchange-document> element from an OPS response,
// regardless of nesting (published-data, family, search). It is the comprehensive
// counterpart to the narrow per-endpoint parsers. A response with no exchange-document yields
// an empty slice and a nil error; only malformed XML returns an error.
func ParseExchangeDocuments(xmlData string) ([]ExchangeDocument, error) {
	dec := xml.NewDecoder(strings.NewReader(xmlData))
	var docs []ExchangeDocument
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, &XMLParseError{
				Parser:    "ParseExchangeDocuments",
				Element:   "exchange-document",
				XMLSample: truncateXML(xmlData, 200),
				Cause:     err,
			}
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "exchange-document" {
			continue
		}
		var d ExchangeDocument
		if err := dec.DecodeElement(&d, &se); err != nil {
			return nil, &XMLParseError{
				Parser:    "ParseExchangeDocuments",
				Element:   "exchange-document",
				XMLSample: truncateXML(xmlData, 200),
				Cause:     err,
			}
		}
		docs = append(docs, d)
	}
	return docs, nil
}
