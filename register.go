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
	// Results are the procedural-step-result values (e.g. "Negative" on a unitary-effect
	// decision), AffectedStates the procedural-step-affected-states text and Gazette the
	// bulletin the step was published in.
	Results        []string          `xml:"procedural-step-result"`
	AffectedStates string            `xml:"procedural-step-affected-states"`
	Gazette        *GazetteReference `xml:"gazette-reference"`
}

// GazetteReference is a gazette-reference: the bulletin number and its date.
type GazetteReference struct {
	Num  string `xml:"gazette-num"`
	Date string `xml:"date"`
}

// DossierEvent is one events-data/dossier-event exactly as the register states it. The same
// element appears at the top of a register document (EP procedure) and inside its
// unitary-patent block.
type DossierEvent struct {
	ID        string            `xml:"id,attr"`
	EventType string            `xml:"event-type,attr"`
	Date      string            `xml:"event-date>date"`
	Code      string            `xml:"event-code"`
	Texts     []EventText       `xml:"event-text"`
	Gazette   *GazetteReference `xml:"gazette-reference"`
}

// EventText is one event-text with its event-text-type (e.g. DESCRIPTION).
type EventText struct {
	Type string `xml:"event-text-type,attr"`
	Text string `xml:",chardata"`
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

// OppositionData is one opposition-data block: either the marker that no opposition was
// filed, or the opponents.
type OppositionData struct {
	ChangeDate  string    `xml:"change-date,attr"`
	GazetteNum  string    `xml:"change-gazette-num,attr"`
	NotFiledTag *struct{} `xml:"opposition-not-filed"`
	// NotFiledDate is the date inside opposition-not-filed. The tag is "-" because the element
	// is already claimed by NotFiledTag; UnmarshalXML fills it.
	NotFiledDate string `xml:"-"`
	// Opponents are the parties that filed an opposition, as listed in this block.
	Opponents []Opponent `xml:"opponent"`
}

// Opponent is one opponent with its representative, status and filing date.
type Opponent struct {
	Sequence    string          `xml:"sequence,attr"`
	Addressbook Addressbook     `xml:"addressbook"`
	Agent       *Representative `xml:"agent"`
	// Status is the status attribute of opponent-status (e.g. "ADMISSIBLE"); the tag is "-"
	// because a field tag cannot address an attribute of a child element.
	Status     string `xml:"-"`
	StatusDate string `xml:"opponent-status>date"`
	FiledDate  string `xml:"date-opposition-filed>date"`
}

// UnmarshalXML decodes an opponent. The status attribute sits on the opponent-status child,
// which a field tag cannot address, hence the explicit decode.
func (o *Opponent) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var aux struct {
		Sequence    string          `xml:"sequence,attr"`
		Addressbook Addressbook     `xml:"addressbook"`
		Agent       *Representative `xml:"agent"`
		Status      struct {
			Status string `xml:"status,attr"`
			Date   string `xml:"date"`
		} `xml:"opponent-status"`
		FiledDate string `xml:"date-opposition-filed>date"`
	}
	if err := d.DecodeElement(&aux, &start); err != nil {
		return err
	}
	*o = Opponent{
		Sequence:    aux.Sequence,
		Addressbook: aux.Addressbook,
		Agent:       aux.Agent,
		Status:      aux.Status.Status,
		StatusDate:  aux.Status.Date,
		FiledDate:   aux.FiledDate,
	}
	return nil
}

// UnmarshalXML decodes one opposition-data block including the date of its
// opposition-not-filed marker.
func (o *OppositionData) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var aux struct {
		ChangeDate string `xml:"change-date,attr"`
		GazetteNum string `xml:"change-gazette-num,attr"`
		NotFiled   *struct {
			Date string `xml:"date"`
		} `xml:"opposition-not-filed"`
		Opponents []Opponent `xml:"opponent"`
	}
	if err := d.DecodeElement(&aux, &start); err != nil {
		return err
	}
	*o = OppositionData{ChangeDate: aux.ChangeDate, GazetteNum: aux.GazetteNum, Opponents: aux.Opponents}
	if aux.NotFiled != nil {
		o.NotFiledTag = &struct{}{}
		o.NotFiledDate = aux.NotFiled.Date
	}
	return nil
}

// NotFiled reports whether the register states no opposition was filed.
func (o *OppositionData) NotFiled() bool { return o != nil && o.NotFiledTag != nil }

// SearchReport references a search-report publication (search-reports-information entry).
type SearchReport struct {
	Country   string `xml:"search-report-publication>document-id>country"`
	DocNumber string `xml:"search-report-publication>document-id>doc-number"`
	Kind      string `xml:"search-report-publication>document-id>kind"`
	Date      string `xml:"search-report-publication>document-id>date"`
	// ID, Office, SearchType and DeclarationOfNoSearch are the entry's attributes (e.g. "EP",
	// "national-supplementary", "not-determined"); ChangeDate and ChangeGazetteNum its stamps.
	ID                    string `xml:"id,attr"`
	Office                string `xml:"office,attr"`
	SearchType            string `xml:"search-type,attr"`
	DeclarationOfNoSearch string `xml:"declaration-of-no-search,attr"`
	ChangeDate            string `xml:"change-date,attr"`
	ChangeGazetteNum      string `xml:"change-gazette-num,attr"`
	// MailedDate is date-search-report-mailed, present on entries without a publication too.
	MailedDate string `xml:"date-search-report-mailed>date"`
	// PublicationLang is the lang attribute of the publication's document-id. The tag is "-"
	// because the document-id is already claimed by the fields above; UnmarshalXML fills it.
	PublicationLang string `xml:"-"`
}

// UnmarshalXML decodes a search-report-information entry including the language of its
// publication. Country, DocNumber, Kind and Date hold what their field tags would give.
func (r *SearchReport) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var aux struct {
		ID                    string `xml:"id,attr"`
		Office                string `xml:"office,attr"`
		SearchType            string `xml:"search-type,attr"`
		DeclarationOfNoSearch string `xml:"declaration-of-no-search,attr"`
		ChangeDate            string `xml:"change-date,attr"`
		ChangeGazetteNum      string `xml:"change-gazette-num,attr"`
		MailedDate            string `xml:"date-search-report-mailed>date"`
		Publications          []struct {
			Lang      *string `xml:"lang,attr"`
			Country   *string `xml:"country"`
			DocNumber *string `xml:"doc-number"`
			Kind      *string `xml:"kind"`
			Date      *string `xml:"date"`
		} `xml:"search-report-publication>document-id"`
	}
	if err := d.DecodeElement(&aux, &start); err != nil {
		return err
	}
	r.ID = aux.ID
	r.Office = aux.Office
	r.SearchType = aux.SearchType
	r.DeclarationOfNoSearch = aux.DeclarationOfNoSearch
	r.ChangeDate = aux.ChangeDate
	r.ChangeGazetteNum = aux.ChangeGazetteNum
	r.MailedDate = aux.MailedDate
	for _, p := range aux.Publications {
		for dst, src := range map[*string]*string{&r.PublicationLang: p.Lang, &r.Country: p.Country, &r.DocNumber: p.DocNumber, &r.Kind: p.Kind, &r.Date: p.Date} {
			if src != nil {
				*dst = *src
			}
		}
	}
	return nil
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

// Milestone is one child of dates-rights-effective exactly as the register states it: the
// element name (e.g. "request-for-examination"), its date, and the stamps on it. MilestoneDates
// is the name-to-date view of the same elements.
type Milestone struct {
	Name             string
	Date             string
	Text             string
	ChangeDate       string
	ChangeGazetteNum string
}

// milestoneList decodes dates-rights-effective into its children in document order.
type milestoneList []Milestone

func (m *milestoneList) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	*m = milestoneList{}
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			var v struct {
				ChangeDate       string `xml:"change-date,attr"`
				ChangeGazetteNum string `xml:"change-gazette-num,attr"`
				Date             string `xml:"date"`
				Text             string `xml:",chardata"`
			}
			if err := d.DecodeElement(&v, &t); err != nil {
				return err
			}
			*m = append(*m, Milestone{
				Name:             t.Name.Local,
				Date:             v.Date,
				Text:             strings.TrimSpace(v.Text),
				ChangeDate:       v.ChangeDate,
				ChangeGazetteNum: v.ChangeGazetteNum,
			})
		case xml.EndElement:
			if t.Name.Local == start.Name.Local {
				return nil
			}
		}
	}
}

// RegisterBiblio is the register's bibliographic-data: the shared exchange fields (embedded)
// plus register-specific children (term of grant, opposition, milestone dates, search reports).
//
// The register keeps history: it repeats a block (applicants, inventors, representatives,
// classifications, priorities, designated states, opposition data) once per change, newest
// first, each stamped with a change date and a gazette number. The embedded flat lists
// (Applicants, Inventors, Agents, IPCR, Priorities, Designations) hold the entries of ALL
// blocks, as they always did; the ...Groups fields keep the blocks apart.
//
// Decoding goes through UnmarshalXML, so xml.Unmarshal and ParseRegister give the same result.
// A field tagged "-" is filled there because its element is already claimed by another field.
type RegisterBiblio struct {
	BibliographicData
	// AgentGroups keeps the register's representative blocks apart, newest first. The register
	// repeats parties>agents for every change, so the flat Agents list mixes current and former
	// representatives; ParseRegister still fills it with all of them.
	AgentGroups    []AgentGroup    `xml:"parties>agents"`
	TermsOfGrant   []TermOfGrant   `xml:"term-of-grant"`
	Opposition     *OppositionData `xml:"opposition-data"`
	MilestoneDates MilestoneDates  `xml:"dates-rights-effective"`
	SearchReports  []SearchReport  `xml:"search-reports-information>search-report-information"`

	// Country, ID, Lang and Status are the attributes of the bibliographic-data element
	// (e.g. "EP", "EP21198049P", "en", "No opposition filed within time limit").
	Country string `xml:"country,attr"`
	ID      string `xml:"id,attr"`
	Lang    string `xml:"lang,attr"`
	Status  string `xml:"status,attr"`

	// ApplicantGroups and InventorGroups keep the applicant and inventor blocks apart, in
	// register order (newest first).
	ApplicantGroups []ApplicantGroup `xml:"-"`
	InventorGroups  []InventorGroup  `xml:"-"`
	// IPCRGroups and CPCGroups are the register's classification blocks. The register writes a
	// block's symbols as one comma-separated text.
	IPCRGroups []RegisterClassificationGroup `xml:"-"`
	CPCGroups  []RegisterClassificationGroup `xml:"classifications-cpc"`
	// PriorityGroups keeps the priority-claims blocks apart.
	PriorityGroups []PriorityGroup `xml:"-"`
	// DesignationGroups keeps the designation-of-states blocks apart; the embedded Designations
	// holds the states of all blocks in one list.
	DesignationGroups []DesignationGroup `xml:"-"`
	// Oppositions holds every opposition-data block, in register order. Opposition is the
	// summary it always was: the stamps of the last block and whether any block carries the
	// opposition-not-filed marker.
	Oppositions []OppositionData `xml:"-"`
	// Milestones are the children of dates-rights-effective in full (MilestoneDates keeps
	// name and date only).
	Milestones []Milestone `xml:"-"`
	// RelatedDocuments are the related-documents entries (divisional parents and children,
	// and any other relation the register states).
	RelatedDocuments []RelatedDocument `xml:"-"`
	// Revocations are the date-of-revocation entries.
	Revocations []RegisterDate `xml:"date-of-revocation"`
	// OfficeSpecific is the office-specific-bib-data (EP appeal data).
	OfficeSpecific []OfficeSpecificData `xml:"office-specific-bib-data"`
}

// ApplicantGroup is one dated block of applicants in the EP Register.
type ApplicantGroup struct {
	ChangeDate string      `xml:"change-date,attr"`
	GazetteNum string      `xml:"change-gazette-num,attr"`
	Applicants []Applicant `xml:"applicant"`
}

// InventorGroup is one dated block of inventors in the EP Register.
type InventorGroup struct {
	ChangeDate string     `xml:"change-date,attr"`
	GazetteNum string     `xml:"change-gazette-num,attr"`
	Inventors  []Inventor `xml:"inventor"`
}

// RegisterClassification is one classification-ipcr or classification-cpc entry of the
// register. Text is the register's own string, several symbols separated by commas.
type RegisterClassification struct {
	ID       string `xml:"id,attr"`
	Sequence string `xml:"sequence,attr"`
	Scheme   string `xml:"scheme,attr"`
	Status   string `xml:"status,attr"`
	Text     string `xml:"text"`
}

// RegisterClassificationGroup is one dated classifications-ipcr or classifications-cpc block.
type RegisterClassificationGroup struct {
	ChangeDate      string                   `xml:"change-date,attr"`
	GazetteNum      string                   `xml:"change-gazette-num,attr"`
	Classifications []RegisterClassification `xml:",any"`
}

// PriorityGroup is one dated priority-claims block.
type PriorityGroup struct {
	ChangeDate string     `xml:"change-date,attr"`
	GazetteNum string     `xml:"change-gazette-num,attr"`
	Priorities []Priority `xml:"priority-claim"`
}

// DesignationGroup is one dated designation-of-states block.
type DesignationGroup struct {
	ChangeDate string `xml:"change-date,attr"`
	GazetteNum string `xml:"change-gazette-num,attr"`
	DesignationOfStates
}

// RelatedDocument is one entry of related-documents. Type is the element name the register
// uses for the relation (e.g. "division"). A divisional application lists its parent under
// Parents; a parent application lists each divisional under Children. The register sends an
// empty document-id on the side that does not apply, and it is kept as sent.
type RelatedDocument struct {
	Type       string
	ChangeDate string
	GazetteNum string
	Parents    []DocumentID
	Children   []DocumentID
}

// relatedDocuments decodes related-documents into one entry per relation element.
type relatedDocuments []RelatedDocument

func (r *relatedDocuments) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			var v struct {
				ChangeDate string       `xml:"change-date,attr"`
				GazetteNum string       `xml:"change-gazette-num,attr"`
				Parents    []DocumentID `xml:"relation>parent-doc>document-id"`
				Children   []DocumentID `xml:"relation>child-doc>document-id"`
			}
			if err := d.DecodeElement(&v, &t); err != nil {
				return err
			}
			*r = append(*r, RelatedDocument{
				Type:       t.Name.Local,
				ChangeDate: v.ChangeDate,
				GazetteNum: v.GazetteNum,
				Parents:    v.Parents,
				Children:   v.Children,
			})
		case xml.EndElement:
			if t.Name.Local == start.Name.Local {
				return nil
			}
		}
	}
}

// RegisterDate is a dated register entry such as date-of-revocation.
type RegisterDate struct {
	ChangeDate string `xml:"change-date,attr"`
	GazetteNum string `xml:"change-gazette-num,attr"`
	Date       string `xml:"date"`
}

// OfficeSpecificData is one office-specific-bib-data block.
type OfficeSpecificData struct {
	File      string     `xml:"file,attr"`
	ID        string     `xml:"id,attr"`
	Lang      string     `xml:"lang,attr"`
	Office    string     `xml:"office,attr"`
	OfficeDTD string     `xml:"office-dtd,attr"`
	Status    string     `xml:"status,attr"`
	Appeals   []EPAppeal `xml:"ep-appeal-data"`
	// Unparsed holds every other child (the register schema also allows revocation,
	// limitation and petition-for-review procedures here).
	Unparsed []RawElement `xml:",any"`
}

// EPAppeal is one ep-appeal-data entry: an appeal with its number, phase, dates and result.
type EPAppeal struct {
	Number                 string `xml:"appeal-num,attr"`
	Phase                  string `xml:"appeal-phase,attr"`
	Date                   string `xml:"date"`
	StatementOfGroundsDate string `xml:"ep-date-statement-of-grounds-filed>date"`
	Result                 string `xml:"-"`
	ResultDate             string `xml:"-"`
	// Unparsed holds every other child (interlocutory revision, fee payment and decision
	// dispatch dates in the register schema).
	Unparsed []RawElement `xml:"-"`
}

// UnmarshalXML decodes an appeal. The result's date is an attribute of the ep-appeal-result
// child, which a field tag cannot address, hence the explicit decode.
func (a *EPAppeal) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var aux struct {
		Number                 string `xml:"appeal-num,attr"`
		Phase                  string `xml:"appeal-phase,attr"`
		Date                   string `xml:"date"`
		StatementOfGroundsDate string `xml:"ep-date-statement-of-grounds-filed>date"`
		Result                 struct {
			Date string `xml:"date,attr"`
			Text string `xml:",chardata"`
		} `xml:"ep-appeal-result"`
		Unparsed []RawElement `xml:",any"`
	}
	if err := d.DecodeElement(&aux, &start); err != nil {
		return err
	}
	*a = EPAppeal{
		Number:                 aux.Number,
		Phase:                  aux.Phase,
		Date:                   aux.Date,
		StatementOfGroundsDate: aux.StatementOfGroundsDate,
		Result:                 strings.TrimSpace(aux.Result.Text),
		ResultDate:             aux.Result.Date,
		Unparsed:               aux.Unparsed,
	}
	return nil
}

// AgentGroup is one dated block of representatives in the EP Register. GazetteNum is the
// bulletin that published the change; the register also uses non-dates such as "N/P" here.
type AgentGroup struct {
	ChangeDate string  `xml:"change-date,attr"`
	GazetteNum string  `xml:"change-gazette-num,attr"`
	Agents     []Agent `xml:"agent"`
	// Representatives holds the same representatives as Agents, index for index, each with its
	// complete addressbook (address lines, cdsid). The tag is "-" because the agent element is
	// already claimed by Agents; UnmarshalXML fills it.
	Representatives []Representative `xml:"-"`
}

// UnmarshalXML decodes a representative block once and fills both views of it.
func (g *AgentGroup) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var aux struct {
		ChangeDate string           `xml:"change-date,attr"`
		GazetteNum string           `xml:"change-gazette-num,attr"`
		Agents     []Representative `xml:"agent"`
	}
	if err := d.DecodeElement(&aux, &start); err != nil {
		return err
	}
	*g = AgentGroup{ChangeDate: aux.ChangeDate, GazetteNum: aux.GazetteNum, Representatives: aux.Agents}
	for _, r := range aux.Agents {
		g.Agents = append(g.Agents, Agent{Sequence: r.Sequence, RepType: r.RepType, Name: r.Addressbook.Name, Country: r.Addressbook.Country})
	}
	return nil
}

// CurrentAgents returns the representatives of record: the first block, because the register
// lists its blocks newest first. It is empty when the register names no representative.
func (b RegisterBiblio) CurrentAgents() []Agent {
	if len(b.AgentGroups) == 0 {
		return nil
	}
	return b.AgentGroups[0].Agents
}

// CurrentOpposition returns the newest opposition-data block (the register lists its blocks
// newest first), or nil when the register has none.
func (b RegisterBiblio) CurrentOpposition() *OppositionData {
	if len(b.Oppositions) == 0 {
		return nil
	}
	return &b.Oppositions[0]
}

// CurrentRepresentatives is CurrentAgents with complete addressbooks.
func (b RegisterBiblio) CurrentRepresentatives() []Representative {
	if len(b.AgentGroups) == 0 {
		return nil
	}
	return b.AgentGroups[0].Representatives
}

// registerBiblioXML is the decode shape of the register's bibliographic-data. Its block
// fields sit above the embedded BibliographicData and so take the repeated elements
// (parties>applicants, classifications-ipcr, ...) away from the embedded flat fields, which
// UnmarshalXML then refills from the blocks.
type registerBiblioXML struct {
	BibliographicData
	Country           string                        `xml:"country,attr"`
	ID                string                        `xml:"id,attr"`
	Lang              string                        `xml:"lang,attr"`
	Status            string                        `xml:"status,attr"`
	ApplicantGroups   []ApplicantGroup              `xml:"parties>applicants"`
	InventorGroups    []InventorGroup               `xml:"parties>inventors"`
	AgentGroups       []AgentGroup                  `xml:"parties>agents"`
	IPCRGroups        []RegisterClassificationGroup `xml:"classifications-ipcr"`
	CPCGroups         []RegisterClassificationGroup `xml:"classifications-cpc"`
	PriorityGroups    []PriorityGroup               `xml:"priority-claims"`
	DesignationGroups []DesignationGroup            `xml:"designation-of-states"`
	TermsOfGrant      []TermOfGrant                 `xml:"term-of-grant"`
	Oppositions       []OppositionData              `xml:"opposition-data"`
	Milestones        *milestoneList                `xml:"dates-rights-effective"`
	SearchReports     []SearchReport                `xml:"search-reports-information>search-report-information"`
	RelatedDocuments  relatedDocuments              `xml:"related-documents"`
	Revocations       []RegisterDate                `xml:"date-of-revocation"`
	OfficeSpecific    []OfficeSpecificData          `xml:"office-specific-bib-data"`
}

// UnmarshalXML decodes the register's bibliographic-data, keeping each repeated block apart
// and filling the flat lists from all blocks.
func (b *RegisterBiblio) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var x registerBiblioXML
	if err := d.DecodeElement(&x, &start); err != nil {
		return err
	}
	*b = RegisterBiblio{
		BibliographicData: x.BibliographicData,
		AgentGroups:       x.AgentGroups,
		TermsOfGrant:      x.TermsOfGrant,
		SearchReports:     x.SearchReports,
		Country:           x.Country,
		ID:                x.ID,
		Lang:              x.Lang,
		Status:            x.Status,
		ApplicantGroups:   x.ApplicantGroups,
		InventorGroups:    x.InventorGroups,
		IPCRGroups:        x.IPCRGroups,
		CPCGroups:         x.CPCGroups,
		PriorityGroups:    x.PriorityGroups,
		DesignationGroups: x.DesignationGroups,
		Oppositions:       x.Oppositions,
		RelatedDocuments:  x.RelatedDocuments,
		Revocations:       x.Revocations,
		OfficeSpecific:    x.OfficeSpecific,
	}
	for _, g := range x.ApplicantGroups {
		b.Applicants = append(b.Applicants, g.Applicants...)
	}
	for _, g := range x.InventorGroups {
		b.Inventors = append(b.Inventors, g.Inventors...)
	}
	for _, g := range x.AgentGroups {
		b.Agents = append(b.Agents, g.Agents...)
	}
	for _, g := range x.IPCRGroups {
		for _, c := range g.Classifications {
			b.IPCR = append(b.IPCR, c.Text)
		}
	}
	for _, g := range x.PriorityGroups {
		b.Priorities = append(b.Priorities, g.Priorities...)
	}
	for _, g := range x.DesignationGroups {
		b.Designations.EPCContracting = append(b.Designations.EPCContracting, g.EPCContracting...)
		b.Designations.EPCExtension = append(b.Designations.EPCExtension, g.EPCExtension...)
		b.Designations.PCTRegional = append(b.Designations.PCTRegional, g.PCTRegional...)
		b.Designations.PCTNational = append(b.Designations.PCTNational, g.PCTNational...)
		b.Designations.Contracting = append(b.Designations.Contracting, g.Contracting...)
		b.Designations.PCTRegion = append(b.Designations.PCTRegion, g.PCTRegion...)
	}
	// Opposition summarises the blocks the way a single struct decoded from repeated elements
	// did: each block overwrites the stamps it carries, the marker stays once seen.
	for _, o := range x.Oppositions {
		if b.Opposition == nil {
			b.Opposition = &OppositionData{}
		}
		if o.ChangeDate != "" {
			b.Opposition.ChangeDate = o.ChangeDate
		}
		if o.GazetteNum != "" {
			b.Opposition.GazetteNum = o.GazetteNum
		}
		if o.NotFiledTag != nil {
			b.Opposition.NotFiledTag = o.NotFiledTag
		}
	}
	if len(x.Oppositions) == 1 {
		one := x.Oppositions[0]
		b.Opposition = &one
	}
	if x.Milestones != nil {
		b.Milestones = *x.Milestones
		b.MilestoneDates = MilestoneDates{}
		for _, m := range b.Milestones {
			val := m.Date
			if val == "" {
				val = m.Text
			}
			if val != "" {
				b.MilestoneDates[m.Name] = val
			}
		}
	}
	return nil
}

// RegisterDocument is one EP Register record. Biblio carries the bibliographic data
// (references, parties, title, classifications, citations, designations, term of grant); the
// remaining fields are register-specific (statuses, procedural steps, events).
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

	// DateProduced, DTDVersion and ProducedBy are the record's production attributes.
	DateProduced string `xml:"date-produced,attr"`
	DTDVersion   string `xml:"dtd-version,attr"`
	ProducedBy   string `xml:"produced-by,attr"`

	// Events are the dossier events of the events constituent, in register order.
	// ParseRegisterEvents gives the same events sorted and categorised.
	Events []DossierEvent `xml:"events-data>dossier-event"`

	// UnitaryProceduralSteps and UnitaryEvents are the procedural steps and events of the
	// unitary-patent block (request for unitary effect and the decision on it).
	UnitaryProceduralSteps []ProceduralStep `xml:"unitary-patent>procedural-data>procedural-step"`
	UnitaryEvents          []DossierEvent   `xml:"unitary-patent>events-data>dossier-event"`

	// Unparsed holds every direct child of the register-document that no field above models.
	Unparsed []RawElement `xml:",any"`
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
