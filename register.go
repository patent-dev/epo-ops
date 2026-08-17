package epo_ops

import (
	"context"
	"encoding/xml"
	"io"
	"strings"
)

// This file provides a comprehensive parser for the EP Register response. The OPS register
// endpoint returns far more than legal events - the full register record across its biblio,
// procedural-steps and unitary-patent (unip) constituents: bibliographic data, EP patent
// statuses, procedural steps, term of grant and opposition data. ParseRegisterEvents covers
// only events+statuses; ParseRegister captures the whole record. The bibliographic data
// reuses the shared exchange types (BibliographicData).

// EPStatus is one ep-patent-status entry (the register's own status timeline).
type EPStatus struct {
	ChangeDate string `xml:"change-date,attr"`
	Code       string `xml:"status-code,attr"`
	Text       string `xml:",chardata"`
}

// UnitaryStatus is one unitary-patent-status entry from the unitary-patent (UPP)
// constituent: the timeline of the request for unitary effect (filed, registered,
// rejected, ...). Present only on the GetRegisterUNIP record.
type UnitaryStatus struct {
	ChangeDate string `xml:"change-date,attr"`
	Code       string `xml:"status-code,attr"`
	Text       string `xml:",chardata"`
}

// ProceduralStep is one procedural-data/procedural-step (the prosecution step log).
type ProceduralStep struct {
	ID        string     `xml:"id,attr"`
	Phase     string     `xml:"procedure-step-phase,attr"`
	Code      string     `xml:"procedural-step-code"`
	Texts     []StepText `xml:"procedural-step-text"`
	Dates     []StepDate `xml:"procedural-step-date"`
	TimeLimit *TimeLimit `xml:"time-limit"`
}

// StepText is one procedural-step-text with its step-text-type (e.g. STEP_DESCRIPTION,
// "Kind of amendment").
type StepText struct {
	Type string `xml:"step-text-type,attr"`
	Text string `xml:",chardata"`
}

// StepDate is one procedural-step-date with its step-date-type (e.g. DATE_OF_DISPATCH,
// DATE_OF_REPLY, DATE_OF_REQUEST).
type StepDate struct {
	Type string `xml:"step-date-type,attr"`
	Date string `xml:"date"`
}

// TimeLimit is the reply time limit stamped on a procedural step (e.g. 04 months).
type TimeLimit struct {
	Unit  string `xml:"time-limit-unit,attr"`
	Value string `xml:",chardata"`
}

// Description returns the STEP_DESCRIPTION text, falling back to the first text.
func (s ProceduralStep) Description() string {
	for _, t := range s.Texts {
		if t.Type == "STEP_DESCRIPTION" {
			return strings.TrimSpace(t.Text)
		}
	}
	if len(s.Texts) > 0 {
		return strings.TrimSpace(s.Texts[0].Text)
	}
	return ""
}

// DateByType returns the date for the given step-date-type, or "" if absent.
func (s ProceduralStep) DateByType(t string) string {
	for _, d := range s.Dates {
		if d.Type == t {
			return d.Date
		}
	}
	return ""
}

// PrimaryDate returns the first procedural-step-date (dispatch/request/...), or "".
func (s ProceduralStep) PrimaryDate() string {
	if len(s.Dates) > 0 {
		return s.Dates[0].Date
	}
	return ""
}

// LapsedCountry is one designated state that lapsed, with the effective date.
type LapsedCountry struct {
	Country string `xml:"country"`
	Date    string `xml:"date"`
}

// TermOfGrant is a term-of-grant entry (lapse / term changes per country).
type TermOfGrant struct {
	ChangeDate      string          `xml:"change-date,attr"`
	GazetteNum      string          `xml:"change-gazette-num,attr"`
	LapsedCountries []LapsedCountry `xml:"lapsed-in-country"`
}

// OppositionData captures opposition status (presence of an opposition-not-filed marker).
type OppositionData struct {
	ChangeDate  string    `xml:"change-date,attr"`
	GazetteNum  string    `xml:"change-gazette-num,attr"`
	NotFiledTag *struct{} `xml:"opposition-not-filed"`
}

// NotFiled reports whether the register states no opposition was filed.
func (o *OppositionData) NotFiled() bool { return o != nil && o.NotFiledTag != nil }

// SearchReport references a search-report publication (search-reports-information entry).
type SearchReport struct {
	Country   string `xml:"search-report-publication>document-id>country"`
	DocNumber string `xml:"search-report-publication>document-id>doc-number"`
	Kind      string `xml:"search-report-publication>document-id>kind"`
	Date      string `xml:"search-report-publication>document-id>date"`
}

// MilestoneDates maps register milestone names (e.g. request-for-examination,
// first-examination-report-despatched) to their dates, captured from dates-rights-effective.
// A map keeps the parser robust to the open-ended set of milestone elements.
type MilestoneDates map[string]string

// UnmarshalXML records each child milestone element under its local name with its date.
func (m *MilestoneDates) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	*m = MilestoneDates{}
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			var v struct {
				Date string `xml:"date"`
				Text string `xml:",chardata"`
			}
			if err := d.DecodeElement(&v, &t); err != nil {
				return err
			}
			val := v.Date
			if val == "" {
				val = strings.TrimSpace(v.Text)
			}
			if val != "" {
				(*m)[t.Name.Local] = val
			}
		case xml.EndElement:
			if t.Name.Local == start.Name.Local {
				return nil
			}
		}
	}
}

// RegisterBiblio is the register's bibliographic-data: the shared exchange fields (embedded)
// plus register-specific children (term of grant, opposition, milestone dates, search reports).
type RegisterBiblio struct {
	BibliographicData
	TermsOfGrant   []TermOfGrant   `xml:"term-of-grant"`
	Opposition     *OppositionData `xml:"opposition-data"`
	MilestoneDates MilestoneDates  `xml:"dates-rights-effective"`
	SearchReports  []SearchReport  `xml:"search-reports-information>search-report-information"`
}

// RegisterDocument is one EP Register record. Biblio carries the bibliographic data
// (references, parties, title, classifications, citations, designations, term of grant); the
// remaining fields are register-specific (statuses, procedural steps).
type RegisterDocument struct {
	ID        string `xml:"id,attr"`
	Status    string `xml:"status,attr"`
	Lang      string `xml:"lang,attr"`
	Country   string `xml:"country,attr"`
	DocNumber string `xml:"doc-number,attr"`
	Kind      string `xml:"kind,attr"`

	Statuses        []EPStatus       `xml:"ep-patent-statuses>ep-patent-status"`
	Biblio          RegisterBiblio   `xml:"bibliographic-data"`
	ProceduralSteps []ProceduralStep `xml:"procedural-data>procedural-step"`

	// UnitaryStatuses is the unitary-patent (UPP) status timeline, populated only
	// on the GetRegisterUNIP record (empty for non-unitary patents).
	UnitaryStatuses []UnitaryStatus `xml:"unitary-patent>unitary-patent-statuses>unitary-patent-status"`
}

// GetRegister retrieves the full EP Register record (bibliographic data, EP patent statuses,
// term of grant, opposition) for a patent. Use GetRegisterEvents for the lighter events-only
// view.
func (c *Client) GetRegister(ctx context.Context, refType, format, number string) ([]RegisterDocument, error) {
	raw, err := c.GetRegisterBiblioRaw(ctx, refType, format, number)
	if err != nil {
		return nil, err
	}
	return ParseRegister(raw)
}

// GetRegisterProceduralSteps retrieves the EP Register procedural-step log (typed).
func (c *Client) GetRegisterProceduralSteps(ctx context.Context, refType, format, number string) ([]RegisterDocument, error) {
	raw, err := c.GetRegisterProceduralStepsRaw(ctx, refType, format, number)
	if err != nil {
		return nil, err
	}
	return ParseRegister(raw)
}

// GetRegisterUNIP retrieves the Unitary Patent register record (typed).
func (c *Client) GetRegisterUNIP(ctx context.Context, refType, format, number string) ([]RegisterDocument, error) {
	raw, err := c.GetRegisterUNIPRaw(ctx, refType, format, number)
	if err != nil {
		return nil, err
	}
	return ParseRegister(raw)
}

// ParseRegister extracts every <register-document> from a register response (biblio, events,
// procedural-steps or unip constituent), capturing the full record rather than only events.
// A response with no register-document yields an empty slice and a nil error; only malformed
// XML returns an error.
func ParseRegister(xmlData string) ([]RegisterDocument, error) {
	dec := xml.NewDecoder(strings.NewReader(xmlData))
	var docs []RegisterDocument
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, &XMLParseError{
				Parser:    "ParseRegister",
				Element:   "register-document",
				XMLSample: truncateXML(xmlData, 200),
				Cause:     err,
			}
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "register-document" {
			continue
		}
		var d RegisterDocument
		if err := dec.DecodeElement(&d, &se); err != nil {
			return nil, &XMLParseError{
				Parser:    "ParseRegister",
				Element:   "register-document",
				XMLSample: truncateXML(xmlData, 200),
				Cause:     err,
			}
		}
		docs = append(docs, d)
	}
	return docs, nil
}
