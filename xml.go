package epo_ops

import (
	_ "embed"
	"encoding/xml"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// Embed XSD schemas into the binary at compile time
// When users import this library, these files are compiled into their binary
// No filesystem access needed at runtime

//go:embed resources/exchange-documents.xsd
var exchangeDocumentsXSD string

//go:embed resources/fulltext-documents.xsd
var fulltextDocumentsXSD string

//go:embed resources/ops_legal.xsd
var opsLegalXSD string

//go:embed resources/ops.xsd
var opsXSD string

//go:embed resources/CPCSchema.xsd
var cpcSchemaXSD string

// GetEmbeddedXSD returns the embedded XSD schema content by name.
// This allows users to access schemas for custom validation if needed.
//
// Available schemas: "exchange-documents", "fulltext-documents", "ops_legal", "ops", "cpc"
func GetEmbeddedXSD(name string) (string, bool) {
	schemas := map[string]string{
		"exchange-documents": exchangeDocumentsXSD,
		"fulltext-documents": fulltextDocumentsXSD,
		"ops_legal":          opsLegalXSD,
		"ops":                opsXSD,
		"cpc":                cpcSchemaXSD,
	}
	schema, ok := schemas[name]
	return schema, ok
}

// XML Parsing Structs and Functions

// AbstractData represents parsed patent abstract
type AbstractData struct {
	XMLName      xml.Name `xml:"world-patent-data"`
	PatentNumber string
	Country      string
	DocNumber    string
	Kind         string
	Language     string
	Text         string
}

// BiblioData represents parsed bibliographic data
type BiblioData struct {
	XMLName         xml.Name `xml:"world-patent-data"`
	PatentNumber    string
	Country         string
	DocNumber       string
	Kind            string
	PublicationDate string
	FamilyID        string
	Titles          map[string]string // lang -> title
	Applicants      []Party
	Inventors       []Party
	IPCClasses      []string
	CPCClasses      []CPCClass
}

// ClaimsData represents parsed patent claims
type ClaimsData struct {
	XMLName      xml.Name `xml:"world-patent-data"`
	PatentNumber string
	Country      string
	DocNumber    string
	Kind         string
	Language     string
	Claims       []Claim
}

// Party represents an applicant or inventor
type Party struct {
	Name    string
	Country string
}

// CPCClass represents a Cooperative Patent Classification
type CPCClass struct {
	Section   string
	Class     string
	Subclass  string
	MainGroup string
	Subgroup  string
	Full      string // Combined representation (e.g., "H04W 84/20")
}

// Claim represents a single patent claim
type Claim struct {
	Number int
	Text   string
}

// FamilyMember represents a single member of a patent family
type FamilyMember struct {
	FamilyID       string
	Country        string
	DocNumber      string
	Kind           string
	Date           string
	ApplicationRef ApplicationReference
	PriorityClaims []PriorityClaim
}

// ApplicationReference represents the application reference for a family member
type ApplicationReference struct {
	Country   string
	DocNumber string
	Kind      string
	Date      string
	DocID     string
}

// PriorityClaim represents a priority claim for a family member
type PriorityClaim struct {
	Country   string
	DocNumber string
	Kind      string
	Date      string
	Sequence  string
	Active    string
}

// FamilyData represents parsed patent family data
type FamilyData struct {
	XMLName      xml.Name `xml:"world-patent-data"`
	PatentNumber string
	FamilyID     string
	TotalCount   int
	Legal        bool
	Members      []FamilyMember
}

// LegalEvent represents a single legal event
type LegalEvent struct {
	Code        string
	Description string
	Influence   string
	DateMigr    string
	Fields      map[string]string
}

// LegalData represents parsed legal event data
type LegalData struct {
	XMLName      xml.Name `xml:"world-patent-data"`
	PatentNumber string
	FamilyID     string
	LegalEvents  []LegalEvent
}

// Paragraph represents a description paragraph
type Paragraph struct {
	ID   string
	Num  string
	Text string
}

// DescriptionData represents parsed description data
type DescriptionData struct {
	XMLName      xml.Name `xml:"world-patent-data"`
	PatentNumber string
	Country      string
	DocNumber    string
	Kind         string
	Language     string
	Paragraphs   []Paragraph
}

// FulltextData represents complete fulltext document data
type FulltextData struct {
	XMLName     xml.Name `xml:"world-patent-data"`
	Country     string
	DocNumber   string
	Kind        string
	Language    string
	Status      string
	Biblio      *BiblioData
	Abstract    *AbstractData
	Description *DescriptionData
	Claims      *ClaimsData
}

// SearchResult represents a single search result
type SearchResult struct {
	System    string
	FamilyID  string
	Country   string
	DocNumber string
	Kind      string
	Title     string
}

// SearchResultData represents search results with pagination
type SearchResultData struct {
	XMLName    xml.Name `xml:"world-patent-data"`
	Query      string
	TotalCount int
	RangeBegin int
	RangeEnd   int
	Results    []SearchResult
}

// EquivalentPatent represents an equivalent patent
type EquivalentPatent struct {
	Country   string
	DocNumber string
	Kind      string
}

// EquivalentsData represents published equivalents inquiry results
type EquivalentsData struct {
	XMLName      xml.Name `xml:"world-patent-data"`
	PatentNumber string
	Equivalents  []EquivalentPatent
}

// Internal structs for XML unmarshaling
type abstractXML struct {
	XMLName          xml.Name `xml:"world-patent-data"`
	ExchangeDocument struct {
		Country   string `xml:"country,attr"`
		DocNumber string `xml:"doc-number,attr"`
		Kind      string `xml:"kind,attr"`
		Abstract  struct {
			Lang string `xml:"lang,attr"`
			P    string `xml:"p"`
		} `xml:"abstract"`
	} `xml:"exchange-documents>exchange-document"`
}

type biblioXML struct {
	XMLName          xml.Name `xml:"world-patent-data"`
	ExchangeDocument struct {
		Country    string `xml:"country,attr"`
		DocNumber  string `xml:"doc-number,attr"`
		Kind       string `xml:"kind,attr"`
		FamilyID   string `xml:"family-id,attr"`
		BiblioData struct {
			PublicationRef struct {
				DocumentID []struct {
					Type      string `xml:"document-id-type,attr"`
					Country   string `xml:"country"`
					DocNumber string `xml:"doc-number"`
					Kind      string `xml:"kind"`
					Date      string `xml:"date"`
				} `xml:"document-id"`
			} `xml:"publication-reference"`
			InventionTitles []struct {
				Lang string `xml:"lang,attr"`
				Text string `xml:",chardata"`
			} `xml:"invention-title"`
			Parties struct {
				Applicants []struct {
					Sequence      string `xml:"sequence,attr"`
					DataFormat    string `xml:"data-format,attr"`
					ApplicantName struct {
						Name string `xml:"name"`
					} `xml:"applicant-name"`
				} `xml:"applicants>applicant"`
				Inventors []struct {
					Sequence     string `xml:"sequence,attr"`
					DataFormat   string `xml:"data-format,attr"`
					InventorName struct {
						Name string `xml:"name"`
					} `xml:"inventor-name"`
				} `xml:"inventors>inventor"`
			} `xml:"parties"`
			ClassificationsIPCR []struct {
				Text string `xml:"text"`
			} `xml:"classifications-ipcr>classification-ipcr"`
			PatentClassifications []struct {
				Section   string `xml:"section"`
				Class     string `xml:"class"`
				Subclass  string `xml:"subclass"`
				MainGroup string `xml:"main-group"`
				Subgroup  string `xml:"subgroup"`
			} `xml:"patent-classifications>patent-classification"`
		} `xml:"bibliographic-data"`
	} `xml:"exchange-documents>exchange-document"`
}

type claimsXML struct {
	XMLName           xml.Name `xml:"world-patent-data"`
	FulltextDocuments struct {
		FulltextDocument struct {
			BiblioData struct {
				PublicationRef struct {
					DocumentID struct {
						Country   string `xml:"country"`
						DocNumber string `xml:"doc-number"`
						Kind      string `xml:"kind"`
					} `xml:"document-id"`
				} `xml:"publication-reference"`
			} `xml:"bibliographic-data"`
			Claims struct {
				Lang      string `xml:"lang,attr"`
				ClaimList struct {
					ClaimTexts []struct {
						Text string `xml:",chardata"`
					} `xml:"claim-text"`
				} `xml:"claim"`
			} `xml:"claims"`
		} `xml:"fulltext-document"`
	} `xml:"fulltext-documents"`
}

// ParseAbstract parses abstract XML into structured data
func ParseAbstract(xmlData string) (*AbstractData, error) {
	var raw abstractXML
	if err := xml.Unmarshal([]byte(xmlData), &raw); err != nil {
		return nil, err
	}

	data := &AbstractData{
		Country:   raw.ExchangeDocument.Country,
		DocNumber: raw.ExchangeDocument.DocNumber,
		Kind:      raw.ExchangeDocument.Kind,
		Language:  raw.ExchangeDocument.Abstract.Lang,
		Text:      strings.TrimSpace(raw.ExchangeDocument.Abstract.P),
	}

	// Construct patent number
	if data.Country != "" && data.DocNumber != "" && data.Kind != "" {
		data.PatentNumber = fmt.Sprintf("%s%s%s", data.Country, data.DocNumber, data.Kind)
	}

	return data, nil
}

// ParseBiblio parses bibliographic XML into structured data
func ParseBiblio(xmlData string) (*BiblioData, error) {
	var raw biblioXML
	if err := xml.Unmarshal([]byte(xmlData), &raw); err != nil {
		return nil, err
	}

	data := &BiblioData{
		Country:   raw.ExchangeDocument.Country,
		DocNumber: raw.ExchangeDocument.DocNumber,
		Kind:      raw.ExchangeDocument.Kind,
		FamilyID:  raw.ExchangeDocument.FamilyID,
		Titles:    make(map[string]string),
	}

	// Construct patent number
	if data.Country != "" && data.DocNumber != "" && data.Kind != "" {
		data.PatentNumber = fmt.Sprintf("%s%s%s", data.Country, data.DocNumber, data.Kind)
	}

	// Extract publication date from first docdb document-id
	for _, docID := range raw.ExchangeDocument.BiblioData.PublicationRef.DocumentID {
		if docID.Type == "docdb" && docID.Date != "" {
			data.PublicationDate = docID.Date
			break
		}
	}

	// Extract titles (multilingual)
	for _, title := range raw.ExchangeDocument.BiblioData.InventionTitles {
		if title.Lang != "" && title.Text != "" {
			data.Titles[title.Lang] = strings.TrimSpace(title.Text)
		}
	}

	// Extract applicants (only epodoc format to avoid duplicates)
	for _, applicant := range raw.ExchangeDocument.BiblioData.Parties.Applicants {
		if applicant.DataFormat == "epodoc" && applicant.ApplicantName.Name != "" {
			name := strings.TrimSpace(applicant.ApplicantName.Name)
			// Extract country from name if present (format: "NAME [CC]")
			country := ""
			if idx := strings.LastIndex(name, "["); idx > 0 {
				if idx2 := strings.Index(name[idx:], "]"); idx2 > 0 {
					country = name[idx+1 : idx+idx2]
					name = strings.TrimSpace(name[:idx])
				}
			}
			data.Applicants = append(data.Applicants, Party{
				Name:    name,
				Country: country,
			})
		}
	}

	// Extract inventors (only epodoc format)
	for _, inventor := range raw.ExchangeDocument.BiblioData.Parties.Inventors {
		if inventor.DataFormat == "epodoc" && inventor.InventorName.Name != "" {
			name := strings.TrimSpace(inventor.InventorName.Name)
			country := ""
			if idx := strings.LastIndex(name, "["); idx > 0 {
				if idx2 := strings.Index(name[idx:], "]"); idx2 > 0 {
					country = name[idx+1 : idx+idx2]
					name = strings.TrimSpace(name[:idx])
				}
			}
			data.Inventors = append(data.Inventors, Party{
				Name:    name,
				Country: country,
			})
		}
	}

	// Extract IPC classifications
	for _, ipc := range raw.ExchangeDocument.BiblioData.ClassificationsIPCR {
		if ipc.Text != "" {
			data.IPCClasses = append(data.IPCClasses, strings.TrimSpace(ipc.Text))
		}
	}

	// Extract CPC classifications
	for _, cpc := range raw.ExchangeDocument.BiblioData.PatentClassifications {
		class := CPCClass{
			Section:   cpc.Section,
			Class:     cpc.Class,
			Subclass:  cpc.Subclass,
			MainGroup: cpc.MainGroup,
			Subgroup:  cpc.Subgroup,
		}
		// Build full representation
		class.Full = fmt.Sprintf("%s%s%s %s/%s", class.Section, class.Class, class.Subclass, class.MainGroup, class.Subgroup)
		data.CPCClasses = append(data.CPCClasses, class)
	}

	return data, nil
}

// ParseClaims parses claims XML into structured data
func ParseClaims(xmlData string) (*ClaimsData, error) {
	var raw claimsXML
	if err := xml.Unmarshal([]byte(xmlData), &raw); err != nil {
		return nil, err
	}

	doc := raw.FulltextDocuments.FulltextDocument
	data := &ClaimsData{
		Country:   doc.BiblioData.PublicationRef.DocumentID.Country,
		DocNumber: doc.BiblioData.PublicationRef.DocumentID.DocNumber,
		Kind:      doc.BiblioData.PublicationRef.DocumentID.Kind,
		Language:  doc.Claims.Lang,
	}

	// Construct patent number
	if data.Country != "" && data.DocNumber != "" && data.Kind != "" {
		data.PatentNumber = fmt.Sprintf("%s%s%s", data.Country, data.DocNumber, data.Kind)
	}

	// Extract claims
	for i, claimText := range doc.Claims.ClaimList.ClaimTexts {
		if claimText.Text != "" {
			data.Claims = append(data.Claims, Claim{
				Number: i + 1,
				Text:   strings.TrimSpace(claimText.Text),
			})
		}
	}

	return data, nil
}

// imageInquiryXML is the internal structure for unmarshaling image inquiry XML.
//
// Note on Link field structure:
// The EPO OPS API returns the image link as a nested element, not an attribute:
//
//	<ops:document-instance desc="Drawing" ...>
//	  <ops:document-instance-link href="/rest-services/..."/>
//	</ops:document-instance>
//
// Previous versions of this library incorrectly used:
//
//	Link string `xml:"link,attr"`
//
// The correct structure is:
//
//	Link struct { Href string `xml:"href,attr"` } `xml:"document-instance-link"`
type imageInquiryXML struct {
	XMLName         xml.Name `xml:"world-patent-data"`
	DocumentInquiry struct {
		InquiryResult struct {
			DocumentInstances []struct {
				Desc          string `xml:"desc,attr"`
				NumberOfPages int    `xml:"number-of-pages,attr"`
				DocType       string `xml:"doc-type,attr"`
				Link          struct {
					Href string `xml:"href,attr"`
				} `xml:"document-instance-link"`
				FormatOptions struct {
					Formats []struct {
						Value string `xml:",chardata"`
					} `xml:"document-format"`
				} `xml:"document-format-options"`
			} `xml:"document-instance"`
		} `xml:"inquiry-result"`
	} `xml:"document-inquiry"`
}

// ParseImageInquiry parses image inquiry XML into structured data.
//
// This function processes the XML response from the EPO OPS Published Images Inquiry service
// and extracts information about available document instances (drawings, full document, etc.),
// their page counts, available formats, and download links.
//
// Example XML structure:
//
//	<ops:world-patent-data>
//	  <ops:document-inquiry>
//	    <ops:inquiry-result>
//	      <ops:document-instance desc="Drawing" number-of-pages="5" doc-type="Drawing">
//	        <ops:document-instance-link href="..."/>
//	        <ops:document-format-options>
//	          <ops:document-format>application/pdf</ops:document-format>
//	          <ops:document-format>image/tiff</ops:document-format>
//	        </ops:document-format-options>
//	      </ops:document-instance>
//	    </ops:inquiry-result>
//	  </ops:document-inquiry>
//	</ops:world-patent-data>
func ParseImageInquiry(xmlData string) (*ImageInquiry, error) {
	var raw imageInquiryXML
	if err := xml.Unmarshal([]byte(xmlData), &raw); err != nil {
		return nil, &XMLParseError{
			Parser:    "ParseImageInquiry",
			Element:   "root",
			XMLSample: truncateXML(xmlData, 200),
			Cause:     err,
		}
	}

	result := &ImageInquiry{
		DocumentInstances: make([]DocumentInstance, 0, len(raw.DocumentInquiry.InquiryResult.DocumentInstances)),
	}

	for i, inst := range raw.DocumentInquiry.InquiryResult.DocumentInstances {
		// Validate required fields
		if inst.Link.Href == "" {
			return nil, &DataValidationError{
				Parser:       "ParseImageInquiry",
				MissingField: fmt.Sprintf("DocumentInstances[%d].Link.Href", i),
				Message:      "document-instance-link href is required but was empty",
			}
		}

		// Extract formats
		formats := make([]string, 0, len(inst.FormatOptions.Formats))
		for _, f := range inst.FormatOptions.Formats {
			format := strings.TrimSpace(f.Value)
			if format != "" {
				formats = append(formats, format)
			}
		}

		result.DocumentInstances = append(result.DocumentInstances, DocumentInstance{
			Description:   inst.Desc,
			Link:          inst.Link.Href,
			NumberOfPages: inst.NumberOfPages,
			Formats:       formats,
			DocType:       inst.DocType,
		})
	}

	// Validate we have at least one document instance
	if len(result.DocumentInstances) == 0 {
		return nil, &DataValidationError{
			Parser:       "ParseImageInquiry",
			MissingField: "DocumentInstances",
			Message:      "no document instances found in response",
		}
	}

	return result, nil
}

// Internal structs for Family XML unmarshaling
type familyXML struct {
	XMLName      xml.Name `xml:"world-patent-data"`
	PatentFamily struct {
		Legal            string `xml:"legal,attr"`
		TotalResultCount string `xml:"total-result-count,attr"`
		PublicationRef   struct {
			DocumentID struct {
				Country   string `xml:"country"`
				DocNumber string `xml:"doc-number"`
				Kind      string `xml:"kind"`
			} `xml:"document-id"`
		} `xml:"publication-reference"`
		FamilyMembers []struct {
			FamilyID       string `xml:"family-id,attr"`
			PublicationRef struct {
				DocumentIDs []struct {
					Type      string `xml:"document-id-type,attr"`
					Country   string `xml:"country"`
					DocNumber string `xml:"doc-number"`
					Kind      string `xml:"kind"`
					Date      string `xml:"date"`
				} `xml:"document-id"`
			} `xml:"publication-reference"`
			ApplicationRef struct {
				DocID      string `xml:"doc-id,attr"`
				DocumentID struct {
					Country   string `xml:"country"`
					DocNumber string `xml:"doc-number"`
					Kind      string `xml:"kind"`
					Date      string `xml:"date"`
				} `xml:"document-id"`
			} `xml:"application-reference"`
			PriorityClaims []struct {
				Sequence   string `xml:"sequence,attr"`
				Kind       string `xml:"kind,attr"`
				DocumentID struct {
					Country   string `xml:"country"`
					DocNumber string `xml:"doc-number"`
					Kind      string `xml:"kind"`
					Date      string `xml:"date"`
				} `xml:"document-id"`
				ActiveIndicator string `xml:"priority-active-indicator"`
			} `xml:"priority-claim"`
		} `xml:"family-member"`
	} `xml:"patent-family"`
}

// truncateXML truncates XML for error messages
func truncateXML(xmlData string, maxLen int) string {
	if len(xmlData) <= maxLen {
		return xmlData
	}
	return xmlData[:maxLen] + "..."
}

// ParseFamily parses patent family XML into structured data
func ParseFamily(xmlData string) (*FamilyData, error) {
	var raw familyXML
	if err := xml.Unmarshal([]byte(xmlData), &raw); err != nil {
		return nil, &XMLParseError{
			Parser:    "ParseFamily",
			Element:   "root",
			XMLSample: truncateXML(xmlData, 200),
			Cause:     err,
		}
	}

	data := &FamilyData{}

	// Parse patent number from publication reference
	// Some family responses have a top-level publication-reference, others don't
	pubRef := raw.PatentFamily.PublicationRef.DocumentID
	if pubRef.Country != "" && pubRef.DocNumber != "" {
		data.PatentNumber = pubRef.Country + pubRef.DocNumber
	} else if len(raw.PatentFamily.FamilyMembers) > 0 {
		// If no top-level publication-reference, use first family member
		firstMember := raw.PatentFamily.FamilyMembers[0]
		if len(firstMember.PublicationRef.DocumentIDs) > 0 {
			firstDoc := firstMember.PublicationRef.DocumentIDs[0]
			data.PatentNumber = firstDoc.Country + firstDoc.DocNumber
		}
	}

	// Patent number is optional for some family responses
	// We'll validate we have at least family members below

	// Parse attributes
	data.Legal = raw.PatentFamily.Legal == "true"
	if raw.PatentFamily.TotalResultCount != "" {
		if _, err := fmt.Sscanf(raw.PatentFamily.TotalResultCount, "%d", &data.TotalCount); err != nil {
			// Non-critical: if parsing fails, TotalCount remains 0
		}
	}

	// Parse family members
	for _, member := range raw.PatentFamily.FamilyMembers {
		familyMember := FamilyMember{
			FamilyID: member.FamilyID,
		}

		// Get first publication reference (docdb format)
		if len(member.PublicationRef.DocumentIDs) > 0 {
			pubDoc := member.PublicationRef.DocumentIDs[0]
			familyMember.Country = pubDoc.Country
			familyMember.DocNumber = pubDoc.DocNumber
			familyMember.Kind = pubDoc.Kind
			familyMember.Date = pubDoc.Date
		}

		// Parse application reference
		appDoc := member.ApplicationRef.DocumentID
		familyMember.ApplicationRef = ApplicationReference{
			Country:   appDoc.Country,
			DocNumber: appDoc.DocNumber,
			Kind:      appDoc.Kind,
			Date:      appDoc.Date,
			DocID:     member.ApplicationRef.DocID,
		}

		// Parse priority claims
		for _, pc := range member.PriorityClaims {
			familyMember.PriorityClaims = append(familyMember.PriorityClaims, PriorityClaim{
				Country:   pc.DocumentID.Country,
				DocNumber: pc.DocumentID.DocNumber,
				Kind:      pc.DocumentID.Kind,
				Date:      pc.DocumentID.Date,
				Sequence:  pc.Sequence,
				Active:    pc.ActiveIndicator,
			})
		}

		if familyMember.FamilyID != "" {
			data.FamilyID = familyMember.FamilyID
		}

		data.Members = append(data.Members, familyMember)
	}

	// Validate parsed data
	// Note: FamilyID may be empty in some responses (especially simplified test data)
	// but should be present in real EPO API responses

	if len(data.Members) == 0 {
		return nil, &DataValidationError{
			Parser:       "ParseFamily",
			MissingField: "Members",
			Message:      "family should have at least one member",
		}
	}

	return data, nil
}

// Internal structs for Legal XML unmarshaling
type legalXML struct {
	XMLName      xml.Name `xml:"world-patent-data"`
	PatentFamily struct {
		PublicationRef struct {
			DocumentID struct {
				Country   string `xml:"country"`
				DocNumber string `xml:"doc-number"`
				Kind      string `xml:"kind"`
			} `xml:"document-id"`
		} `xml:"publication-reference"`
		FamilyMembers []struct {
			FamilyID    string          `xml:"family-id,attr"`
			LegalEvents []legalEventXML `xml:"legal"`
		} `xml:"family-member"`
	} `xml:"patent-family"`
}

// legalEventXML represents a single legal event with dynamic L*EP fields.
// The EPO OPS API returns legal event data with L-fields (L001EP, L002EP, etc.)
// that vary by event type and jurisdiction. We define a sufficient range
// (L001EP through L050EP) and use reflection to dynamically extract all non-empty fields.
type legalEventXML struct {
	Code     string   `xml:"code,attr"`
	Desc     string   `xml:"desc,attr"`
	Infl     string   `xml:"infl,attr"`
	DateMigr string   `xml:"dateMigr,attr"`
	Pre      []string `xml:"pre"`
	// L-fields: Extended to L050EP to support future EPO additions
	L001EP string `xml:"L001EP"`
	L002EP string `xml:"L002EP"`
	L003EP string `xml:"L003EP"`
	L004EP string `xml:"L004EP"`
	L005EP string `xml:"L005EP"`
	L006EP string `xml:"L006EP"`
	L007EP string `xml:"L007EP"`
	L008EP string `xml:"L008EP"`
	L009EP string `xml:"L009EP"`
	L010EP string `xml:"L010EP"`
	L011EP string `xml:"L011EP"`
	L012EP string `xml:"L012EP"`
	L013EP string `xml:"L013EP"`
	L014EP string `xml:"L014EP"`
	L015EP string `xml:"L015EP"`
	L016EP string `xml:"L016EP"`
	L017EP string `xml:"L017EP"`
	L018EP string `xml:"L018EP"`
	L019EP string `xml:"L019EP"`
	L020EP string `xml:"L020EP"`
	L021EP string `xml:"L021EP"`
	L022EP string `xml:"L022EP"`
	L023EP string `xml:"L023EP"`
	L024EP string `xml:"L024EP"`
	L025EP string `xml:"L025EP"`
	L026EP string `xml:"L026EP"`
	L027EP string `xml:"L027EP"`
	L028EP string `xml:"L028EP"`
	L029EP string `xml:"L029EP"`
	L030EP string `xml:"L030EP"`
	L031EP string `xml:"L031EP"`
	L032EP string `xml:"L032EP"`
	L033EP string `xml:"L033EP"`
	L034EP string `xml:"L034EP"`
	L035EP string `xml:"L035EP"`
	L036EP string `xml:"L036EP"`
	L037EP string `xml:"L037EP"`
	L038EP string `xml:"L038EP"`
	L039EP string `xml:"L039EP"`
	L040EP string `xml:"L040EP"`
	L041EP string `xml:"L041EP"`
	L042EP string `xml:"L042EP"`
	L043EP string `xml:"L043EP"`
	L044EP string `xml:"L044EP"`
	L045EP string `xml:"L045EP"`
	L046EP string `xml:"L046EP"`
	L047EP string `xml:"L047EP"`
	L048EP string `xml:"L048EP"`
	L049EP string `xml:"L049EP"`
	L050EP string `xml:"L050EP"`
}

// Cache for legal field metadata to avoid repeated reflection
var (
	legalFieldIndices []int     // indices of L*EP fields in legalEventXML
	legalFieldNames   []string  // corresponding field names
	legalFieldsOnce   sync.Once // ensures cache is initialized only once
)

// initLegalFieldsCache initializes the cache of legal field indices and names.
// This is called once via sync.Once to avoid repeated reflection overhead.
func initLegalFieldsCache() {
	t := reflect.TypeOf(legalEventXML{})

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldName := field.Name

		// Check if field name matches L*EP pattern (starts with L, ends with EP)
		if strings.HasPrefix(fieldName, "L") && strings.HasSuffix(fieldName, "EP") {
			// Only cache string fields
			if field.Type.Kind() == reflect.String {
				legalFieldIndices = append(legalFieldIndices, i)
				legalFieldNames = append(legalFieldNames, fieldName)
			}
		}
	}
}

// extractLegalFields uses reflection to dynamically extract all L*EP fields
// from a legalEventXML struct. This automatically handles any number of L-fields
// without requiring hardcoded field names.
//
// Performance: Field metadata is cached on first call using sync.Once, so subsequent
// calls only perform value extraction without type inspection overhead.
func extractLegalFields(legal legalEventXML) map[string]string {
	// Initialize cache on first call
	legalFieldsOnce.Do(initLegalFieldsCache)

	fields := make(map[string]string)
	v := reflect.ValueOf(legal)

	// Use cached indices to extract values
	for i, fieldIdx := range legalFieldIndices {
		fieldValue := v.Field(fieldIdx)
		value := fieldValue.String()
		if value != "" {
			fields[legalFieldNames[i]] = value
		}
	}

	return fields
}

// ParseLegal parses legal event XML into structured data
func ParseLegal(xmlData string) (*LegalData, error) {
	var raw legalXML
	if err := xml.Unmarshal([]byte(xmlData), &raw); err != nil {
		return nil, &XMLParseError{
			Parser:    "ParseLegal",
			Element:   "root",
			XMLSample: truncateXML(xmlData, 200),
			Cause:     err,
		}
	}

	data := &LegalData{}

	// Parse patent number
	pubRef := raw.PatentFamily.PublicationRef.DocumentID
	if pubRef.Country == "" || pubRef.DocNumber == "" {
		return nil, &DataValidationError{
			Parser:       "ParseLegal",
			MissingField: "publication-reference",
			Message:      "country or doc-number is empty",
		}
	}
	data.PatentNumber = pubRef.Country + pubRef.DocNumber

	// Parse legal events from all family members
	for _, member := range raw.PatentFamily.FamilyMembers {
		if member.FamilyID != "" {
			data.FamilyID = member.FamilyID
		}

		for _, legal := range member.LegalEvents {
			event := LegalEvent{
				Code:        legal.Code,
				Description: legal.Desc,
				Influence:   legal.Infl,
				DateMigr:    legal.DateMigr,
				Fields:      extractLegalFields(legal), // Dynamic extraction using reflection
			}

			data.LegalEvents = append(data.LegalEvents, event)
		}
	}

	return data, nil
}

// Internal structs for Description XML unmarshaling
type descriptionXML struct {
	XMLName           xml.Name `xml:"world-patent-data"`
	FulltextDocuments struct {
		FulltextDocument struct {
			Country    string `xml:"country,attr"`
			DocNumber  string `xml:"doc-number,attr"`
			Kind       string `xml:"kind,attr"`
			BiblioData struct {
				PublicationRef struct {
					DocumentID struct {
						Country   string `xml:"country"`
						DocNumber string `xml:"doc-number"`
						Kind      string `xml:"kind"`
					} `xml:"document-id"`
				} `xml:"publication-reference"`
			} `xml:"bibliographic-data"`
			Description struct {
				Lang       string `xml:"lang,attr"`
				Paragraphs []struct {
					ID   string `xml:"id,attr"`
					Num  string `xml:"num,attr"`
					Text string `xml:",chardata"`
				} `xml:"p"`
			} `xml:"description"`
		} `xml:"fulltext-document"`
	} `xml:"fulltext-documents"`
}

// ParseDescription parses description XML into structured data
func ParseDescription(xmlData string) (*DescriptionData, error) {
	var raw descriptionXML
	if err := xml.Unmarshal([]byte(xmlData), &raw); err != nil {
		return nil, &XMLParseError{
			Parser:    "ParseDescription",
			Element:   "root",
			XMLSample: truncateXML(xmlData, 200),
			Cause:     err,
		}
	}

	doc := raw.FulltextDocuments.FulltextDocument
	data := &DescriptionData{
		Country:   doc.Country,
		DocNumber: doc.DocNumber,
		Kind:      doc.Kind,
		Language:  doc.Description.Lang,
	}

	// Use biblio data if available
	if doc.BiblioData.PublicationRef.DocumentID.Country != "" {
		pubRef := doc.BiblioData.PublicationRef.DocumentID
		data.PatentNumber = pubRef.Country + pubRef.DocNumber
	} else {
		data.PatentNumber = doc.Country + doc.DocNumber
	}

	// Parse paragraphs
	for _, p := range doc.Description.Paragraphs {
		data.Paragraphs = append(data.Paragraphs, Paragraph{
			ID:   p.ID,
			Num:  p.Num,
			Text: strings.TrimSpace(p.Text),
		})
	}

	return data, nil
}

// Internal structs for Fulltext XML unmarshaling
type fulltextXML struct {
	XMLName           xml.Name `xml:"world-patent-data"`
	FulltextDocuments struct {
		FulltextDocument struct {
			Country   string `xml:"country,attr"`
			DocNumber string `xml:"doc-number,attr"`
			Kind      string `xml:"kind,attr"`
			Lang      string `xml:"lang,attr"`
			Status    string `xml:"status,attr"`
		} `xml:"fulltext-document"`
	} `xml:"fulltext-documents"`
}

// ParseFulltext parses fulltext XML into structured data by reusing existing parsers
func ParseFulltext(xmlData string) (*FulltextData, error) {
	var raw fulltextXML
	if err := xml.Unmarshal([]byte(xmlData), &raw); err != nil {
		return nil, fmt.Errorf("failed to unmarshal fulltext XML: %w", err)
	}

	doc := raw.FulltextDocuments.FulltextDocument
	data := &FulltextData{
		Country:   doc.Country,
		DocNumber: doc.DocNumber,
		Kind:      doc.Kind,
		Language:  doc.Lang,
		Status:    doc.Status,
	}

	// Try to parse each section separately using existing parsers
	// If a section fails, we continue with the others

	if biblio, err := ParseBiblio(xmlData); err == nil {
		data.Biblio = biblio
	}

	if abstract, err := ParseAbstract(xmlData); err == nil {
		data.Abstract = abstract
	}

	if description, err := ParseDescription(xmlData); err == nil {
		data.Description = description
	}

	if claims, err := ParseClaims(xmlData); err == nil {
		data.Claims = claims
	}

	return data, nil
}

// Internal structs for Search XML unmarshaling
type searchXML struct {
	XMLName      xml.Name `xml:"world-patent-data"`
	BiblioSearch struct {
		TotalResultCount string `xml:"total-result-count,attr"`
		Query            string `xml:"query"`
		Range            struct {
			Begin string `xml:"begin,attr"`
			End   string `xml:"end,attr"`
		} `xml:"range"`
		ExchangeDocuments struct {
			Documents []struct {
				System     string `xml:"system,attr"`
				FamilyID   string `xml:"family-id,attr"`
				Country    string `xml:"country,attr"`
				DocNumber  string `xml:"doc-number,attr"`
				Kind       string `xml:"kind,attr"`
				BiblioData struct {
					InventionTitle []struct {
						Lang string `xml:"lang,attr"`
						Text string `xml:",chardata"`
					} `xml:"invention-title"`
				} `xml:"bibliographic-data"`
			} `xml:"exchange-document"`
		} `xml:"exchange-documents"`
	} `xml:"biblio-search"`
}

// ParseSearch parses search result XML into structured data
func ParseSearch(xmlData string) (*SearchResultData, error) {
	var raw searchXML
	if err := xml.Unmarshal([]byte(xmlData), &raw); err != nil {
		return nil, &XMLParseError{
			Parser:    "ParseSearch",
			Element:   "root",
			XMLSample: truncateXML(xmlData, 200),
			Cause:     err,
		}
	}

	data := &SearchResultData{
		Query: raw.BiblioSearch.Query,
	}

	// Parse counts and ranges
	if raw.BiblioSearch.TotalResultCount != "" {
		if _, err := fmt.Sscanf(raw.BiblioSearch.TotalResultCount, "%d", &data.TotalCount); err != nil {
			// Non-critical: if parsing fails, TotalCount remains 0
		}
	}
	if raw.BiblioSearch.Range.Begin != "" {
		if _, err := fmt.Sscanf(raw.BiblioSearch.Range.Begin, "%d", &data.RangeBegin); err != nil {
			// Non-critical: if parsing fails, RangeBegin remains 0
		}
	}
	if raw.BiblioSearch.Range.End != "" {
		if _, err := fmt.Sscanf(raw.BiblioSearch.Range.End, "%d", &data.RangeEnd); err != nil {
			// Non-critical: if parsing fails, RangeEnd remains 0
		}
	}

	// Parse results
	for _, doc := range raw.BiblioSearch.ExchangeDocuments.Documents {
		result := SearchResult{
			System:    doc.System,
			FamilyID:  doc.FamilyID,
			Country:   doc.Country,
			DocNumber: doc.DocNumber,
			Kind:      doc.Kind,
		}

		// Get title (prefer English, fall back to first available)
		for _, title := range doc.BiblioData.InventionTitle {
			if title.Lang == "en" || result.Title == "" {
				result.Title = title.Text
			}
		}

		data.Results = append(data.Results, result)
	}

	return data, nil
}

// Internal structs for Equivalents XML unmarshaling
type equivalentsXML struct {
	XMLName            xml.Name `xml:"world-patent-data"`
	EquivalentsInquiry struct {
		PublicationRef struct {
			DocumentID struct {
				Country   string `xml:"country"`
				DocNumber string `xml:"doc-number"`
				Kind      string `xml:"kind"`
			} `xml:"document-id"`
		} `xml:"publication-reference"`
		InquiryResults []struct {
			PublicationRef struct {
				DocumentID struct {
					DocNumber string `xml:"doc-number"`
				} `xml:"document-id"`
			} `xml:"publication-reference"`
		} `xml:"inquiry-result"`
	} `xml:"equivalents-inquiry"`
}

// ParseEquivalents parses equivalents inquiry XML into structured data
func ParseEquivalents(xmlData string) (*EquivalentsData, error) {
	var raw equivalentsXML
	if err := xml.Unmarshal([]byte(xmlData), &raw); err != nil {
		return nil, &XMLParseError{
			Parser:    "ParseEquivalents",
			Element:   "root",
			XMLSample: truncateXML(xmlData, 200),
			Cause:     err,
		}
	}

	pubRef := raw.EquivalentsInquiry.PublicationRef.DocumentID
	if pubRef.Country == "" || pubRef.DocNumber == "" {
		return nil, &DataValidationError{
			Parser:       "ParseEquivalents",
			MissingField: "publication-reference",
			Message:      "country or doc-number is empty",
		}
	}

	data := &EquivalentsData{
		PatentNumber: pubRef.Country + pubRef.DocNumber,
	}

	// Parse equivalents
	for _, result := range raw.EquivalentsInquiry.InquiryResults {
		docNum := result.PublicationRef.DocumentID.DocNumber
		if docNum == "" {
			continue
		}

		// Parse country code from document number (e.g., "EP2400812" -> "EP", "2400812")
		country := ""
		number := docNum
		kind := ""

		// Extract 2-letter country code
		if len(docNum) >= 2 {
			country = docNum[:2]
			number = docNum[2:]
		}

		data.Equivalents = append(data.Equivalents, EquivalentPatent{
			Country:   country,
			DocNumber: number,
			Kind:      kind,
		})
	}

	return data, nil
}
