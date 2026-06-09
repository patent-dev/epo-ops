package epo_ops

import (
	_ "embed"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"
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
	Title          string   // Invention title (populated from biblio data, English preferred)
	Applicants     []string // Applicant names (populated from biblio data, epodoc format)
	ApplicationRef ApplicationReference
	PriorityClaims []PriorityClaim
}

// ApplicationReference represents the application reference for a family member
type ApplicationReference struct {
	Country           string
	DocNumber         string // docdb-standardised application number
	Kind              string
	Date              string
	DocID             string
	OriginalDocNumber string // application number as supplied by the originating office (document-id-type="original"); the key national offices look up by
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
	Countries    []string // Unique country codes from all members, sorted alphabetically
	Members      []FamilyMember
}

// LegalEvent represents a single legal event
type LegalEvent struct {
	Code        string
	Description string
	Influence   string
	DateMigr    string
	Country     string            // Country code from L001EP field
	Date        string            // Gazette date from L007EP field
	EventCode   string            // Legal event code from L008EP field
	Fields      map[string]string // All L-code field values
}

// LegalData represents parsed legal event data
type LegalData struct {
	XMLName      xml.Name `xml:"world-patent-data"`
	PatentNumber string
	FamilyID     string
	LegalEvents  []LegalEvent
}

// RegisterEvent represents a single dossier event from the EPO Register
type RegisterEvent struct {
	ID          string // Event identifier (e.g., "EVT_434577438")
	Date        string // Event date (YYYYMMDD)
	EventCode   string // Event code (e.g., "0009250", "EPIDOSNIGR1")
	Description string // Human-readable description
	Category    string // Categorized: filing, publication, examination, grant, opposition, appeal, other
	GazetteNum  string // Gazette number (e.g., "2022/32"), empty if not available
	GazetteDate string // Gazette date (YYYYMMDD), empty if not available
}

// PatentStatus represents a patent status entry from the EPO Register
type PatentStatus struct {
	Date        string // Status change date (YYYYMMDD)
	Code        string // Status code (e.g., "7", "8")
	Description string // Human-readable description
}

// RegisterEventsData represents parsed register events data
type RegisterEventsData struct {
	PatentNumber string
	Query        string
	Statuses     []PatentStatus
	Events       []RegisterEvent
}

// NumberConversionData represents parsed patent number conversion data
type NumberConversionData struct {
	InputFormat  string
	OutputFormat string
	Country      string
	DocNumber    string
	Kind         string
	Date         string
}

// ClassificationChild represents a child classification item
type ClassificationChild struct {
	Symbol      string
	Title       string
	Level       int
	HasChildren bool // True when the symbol has descendants that are not included in this response.
}

// ClassificationData represents parsed CPC classification schema data
type ClassificationData struct {
	Symbol      string
	Title       string
	Level       int
	SchemeType  string
	HasChildren bool // True when the symbol has descendants that are not included in this response.
	Children    []ClassificationChild
	Ancestors   []ClassificationChild
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
				DocID       string `xml:"doc-id,attr"`
				DocumentIDs []struct {
					Type      string `xml:"document-id-type,attr"`
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
			ExchangeDocuments []struct {
				BiblioData struct {
					Titles []struct {
						Lang  string `xml:"lang,attr"`
						Title string `xml:",chardata"`
					} `xml:"invention-title"`
					Parties struct {
						Applicants struct {
							Applicants []struct {
								DataFormat string `xml:"data-format,attr"`
								Name       struct {
									Name string `xml:"name"`
								} `xml:"applicant-name"`
							} `xml:"applicant"`
						} `xml:"applicants"`
					} `xml:"parties"`
				} `xml:"bibliographic-data"`
			} `xml:"exchange-document"`
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

	data := &FamilyData{
		Members: []FamilyMember{}, // Always non-nil, even if empty
	}

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

	// Parse attributes
	data.Legal = raw.PatentFamily.Legal == "true"
	if raw.PatentFamily.TotalResultCount != "" {
		_, _ = fmt.Sscanf(raw.PatentFamily.TotalResultCount, "%d", &data.TotalCount)
	}

	// Parse family members
	countrySet := make(map[string]bool)
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

		// Parse application reference: docdb is the standardised number; "original" is
		// the originating-office number national offices (e.g. JP/TW) look up by.
		appRef := ApplicationReference{DocID: member.ApplicationRef.DocID}
		for _, d := range member.ApplicationRef.DocumentIDs {
			switch d.Type {
			case "docdb", "":
				appRef.Country, appRef.DocNumber, appRef.Kind, appRef.Date = d.Country, d.DocNumber, d.Kind, d.Date
			case "original":
				appRef.OriginalDocNumber = d.DocNumber
			}
		}
		// Fallback: if no docdb document-id was present, use the first available one so a
		// member that only carries epodoc still keeps an application number.
		if appRef.DocNumber == "" && len(member.ApplicationRef.DocumentIDs) > 0 {
			d := member.ApplicationRef.DocumentIDs[0]
			appRef.Country, appRef.DocNumber, appRef.Kind, appRef.Date = d.Country, d.DocNumber, d.Kind, d.Date
		}
		familyMember.ApplicationRef = appRef

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

		// Extract biblio data from exchange-document (present in GetFamilyWithBiblio responses)
		for _, exDoc := range member.ExchangeDocuments {
			// Title: prefer English, fallback to first available
			if familyMember.Title == "" {
				for _, t := range exDoc.BiblioData.Titles {
					title := strings.TrimSpace(t.Title)
					if title == "" {
						continue
					}
					if strings.EqualFold(t.Lang, "en") {
						familyMember.Title = title
						break
					}
					if familyMember.Title == "" {
						familyMember.Title = title
					}
				}
			}

			// Applicants: prefer epodoc format, deduplicate
			if len(familyMember.Applicants) == 0 {
				seen := make(map[string]bool)
				for _, a := range exDoc.BiblioData.Parties.Applicants.Applicants {
					if !strings.EqualFold(a.DataFormat, "epodoc") {
						continue
					}
					name := strings.TrimSpace(a.Name.Name)
					if name != "" && !seen[name] {
						seen[name] = true
						familyMember.Applicants = append(familyMember.Applicants, name)
					}
				}
			}
		}

		if familyMember.FamilyID != "" {
			data.FamilyID = familyMember.FamilyID
		}

		if familyMember.Country != "" {
			countrySet[familyMember.Country] = true
		}

		data.Members = append(data.Members, familyMember)
	}

	// Compute sorted unique countries
	if len(countrySet) > 0 {
		countries := make([]string, 0, len(countrySet))
		for c := range countrySet {
			countries = append(countries, c)
		}
		sort.Strings(countries)
		data.Countries = countries
	} else {
		data.Countries = []string{}
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
	Code     string            `xml:"code,attr"`
	Desc     string            `xml:"desc,attr"`
	Infl     string            `xml:"infl,attr"`
	DateMigr string            `xml:"dateMigr,attr"`
	Pre      []string          `xml:"pre"`
	Fields   map[string]string // every L-code child element (L001EP..L5xxEP), unbounded
}

// UnmarshalXML captures every child element of <legal> dynamically, so any L-code is kept
// regardless of range. OPS legal status uses codes well beyond L050 (real responses reach
// L5xxEP); the previous fixed field list silently dropped everything above L050EP.
func (e *legalEventXML) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	e.Fields = make(map[string]string)
	for _, a := range start.Attr {
		switch a.Name.Local {
		case "code":
			e.Code = a.Value
		case "desc":
			e.Desc = a.Value
		case "infl":
			e.Infl = a.Value
		case "dateMigr":
			e.DateMigr = a.Value
		}
	}
	return e.walkLegal(d, start)
}

// walkLegal records the direct text of every descendant element under its local name. OPS
// nests some L-codes inside others (e.g. L500EP contains L525EP), so a flat walk over direct
// children misses them; recursing captures every L-code at any depth. <pre> raw lines are
// collected separately.
func (e *legalEventXML) walkLegal(d *xml.Decoder, start xml.StartElement) error {
	var text strings.Builder
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.CharData:
			text.Write(t)
		case xml.StartElement:
			if t.Name.Local == "pre" {
				var v string
				if err := d.DecodeElement(&v, &t); err != nil {
					return err
				}
				e.Pre = append(e.Pre, v)
				continue
			}
			if err := e.walkLegal(d, t); err != nil {
				return err
			}
		case xml.EndElement:
			// Record the element's direct text under its local name. If the same L-code
			// appears more than once in one event the last value wins (a map cannot hold
			// repeats); this matches the prior fixed-field behaviour. Repeated raw lines
			// are preserved separately in Pre.
			if v := strings.TrimSpace(text.String()); v != "" && start.Name.Local != "legal" {
				e.Fields[start.Name.Local] = v
			}
			return nil
		}
	}
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

	data := &LegalData{
		LegalEvents: []LegalEvent{}, // Always non-nil, even if empty
	}

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
			fields := legal.Fields
			event := LegalEvent{
				Code:        strings.TrimSpace(legal.Code),
				Description: strings.TrimSpace(legal.Desc),
				Influence:   strings.TrimSpace(legal.Infl),
				DateMigr:    strings.TrimSpace(legal.DateMigr),
				Country:     strings.TrimSpace(fields["L001EP"]),
				Date:        strings.TrimSpace(fields["L007EP"]),
				EventCode:   strings.TrimSpace(fields["L008EP"]),
				Fields:      fields,
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

// searchExchangeDocs holds exchange-document elements from search results (with constituents)
type searchExchangeDocs struct {
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
}

// searchPubRef holds publication-reference elements from search results (without constituents)
type searchPubRef struct {
	System   string `xml:"system,attr"`
	FamilyID string `xml:"family-id,attr"`
	DocID    struct {
		Country   string `xml:"country"`
		DocNumber string `xml:"doc-number"`
		Kind      string `xml:"kind"`
	} `xml:"document-id"`
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
		SearchResult struct {
			ExchangeDocuments     searchExchangeDocs `xml:"exchange-documents"`
			PublicationReferences []searchPubRef     `xml:"publication-reference"`
		} `xml:"search-result"`
	} `xml:"biblio-search"`
}

// ParseSearch parses search result XML into structured data.
// Handles both formats: with constituents (exchange-documents) and without (publication-reference).
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
		_, _ = fmt.Sscanf(raw.BiblioSearch.TotalResultCount, "%d", &data.TotalCount)
	}
	if raw.BiblioSearch.Range.Begin != "" {
		_, _ = fmt.Sscanf(raw.BiblioSearch.Range.Begin, "%d", &data.RangeBegin)
	}
	if raw.BiblioSearch.Range.End != "" {
		_, _ = fmt.Sscanf(raw.BiblioSearch.Range.End, "%d", &data.RangeEnd)
	}

	// Parse results from exchange-documents (search with constituents)
	for _, doc := range raw.BiblioSearch.SearchResult.ExchangeDocuments.Documents {
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

	// Fallback: parse publication-reference elements (search without constituents)
	if len(data.Results) == 0 {
		for _, ref := range raw.BiblioSearch.SearchResult.PublicationReferences {
			data.Results = append(data.Results, SearchResult{
				System:    ref.System,
				FamilyID:  ref.FamilyID,
				Country:   ref.DocID.Country,
				DocNumber: ref.DocID.DocNumber,
				Kind:      ref.DocID.Kind,
			})
		}
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

// --- Register Events XML Structs ---

// registerEventsXML is the internal XML struct for register events responses
type registerEventsXML struct {
	XMLName        xml.Name `xml:"world-patent-data"`
	RegisterSearch struct {
		TotalCount string `xml:"total-result-count,attr"`
		Query      struct {
			Text string `xml:",chardata"`
		} `xml:"query"`
		RegisterDocuments struct {
			Documents []registerDocumentXML `xml:"register-document"`
		} `xml:"register-documents"`
	} `xml:"register-search"`
}

type registerDocumentXML struct {
	Status   string `xml:"status,attr"`
	Statuses struct {
		Entries []struct {
			ChangeDate string `xml:"change-date,attr"`
			StatusCode string `xml:"status-code,attr"`
			Text       string `xml:",chardata"`
		} `xml:"ep-patent-status"`
	} `xml:"ep-patent-statuses"`
	EventsData []struct {
		DossierEvent struct {
			ID        string `xml:"id,attr"`
			EventType string `xml:"event-type,attr"`
			EventDate struct {
				Date string `xml:"date"`
			} `xml:"event-date"`
			EventCode string `xml:"event-code"`
			EventText struct {
				Type string `xml:"event-text-type,attr"`
				Text string `xml:",chardata"`
			} `xml:"event-text"`
			GazetteRef struct {
				GazetteNum string `xml:"gazette-num"`
				Date       string `xml:"date"`
			} `xml:"gazette-reference"`
		} `xml:"dossier-event"`
	} `xml:"events-data"`
}

// categorizeRegisterEvent maps an event code to a category string.
// The description parameter is used as a fallback when the event code
// doesn't match a known pattern.
func categorizeRegisterEvent(eventCode, description string) string {
	upper := strings.ToUpper(eventCode)

	// Grant-related
	if strings.Contains(upper, "IGR") || eventCode == "0009210" {
		return "grant"
	}
	// Examination-related
	if strings.Contains(upper, "EXR") || eventCode == "0009171" || eventCode == "0009185" {
		return "examination"
	}
	// Publication-related
	if eventCode == "0009012" {
		return "publication"
	}
	// No-opposition (0009261 = "No opposition filed within time limit")
	if eventCode == "0009261" {
		return "no_opposition"
	}
	// Opposition-related
	if strings.Contains(upper, "OPPO") {
		return "opposition"
	}
	// Appeal-related
	if strings.Contains(upper, "APPEAL") {
		return "appeal"
	}
	// Filing-related
	if eventCode == "0009100" {
		return "filing"
	}

	// Description-based fallback for unrecognized codes
	upperDesc := strings.ToUpper(description)
	if strings.Contains(upperDesc, "APPEAL") {
		return "appeal"
	}
	if strings.Contains(upperDesc, "NO OPPOSITION") {
		return "no_opposition"
	}
	if strings.Contains(upperDesc, "OPPOSITION") {
		return "opposition"
	}

	return "other"
}

// ParseRegisterEvents parses EPO Register events XML into a RegisterEventsData struct.
func ParseRegisterEvents(xmlData string) (*RegisterEventsData, error) {
	var raw registerEventsXML
	if err := xml.Unmarshal([]byte(xmlData), &raw); err != nil {
		return nil, &XMLParseError{
			Parser:    "ParseRegisterEvents",
			Element:   "world-patent-data",
			XMLSample: truncateXML(xmlData, 200),
			Cause:     err,
		}
	}

	data := &RegisterEventsData{
		Query:    strings.TrimSpace(raw.RegisterSearch.Query.Text),
		Statuses: []PatentStatus{},
		Events:   []RegisterEvent{},
	}

	// Extract patent number from query (e.g., "publication=EP2400812" -> "EP2400812")
	if q := data.Query; q != "" {
		if idx := strings.Index(q, "="); idx >= 0 {
			data.PatentNumber = strings.TrimSpace(q[idx+1:])
		}
	}

	docs := raw.RegisterSearch.RegisterDocuments.Documents
	if len(docs) == 0 {
		return data, nil
	}

	// Use the first register document
	doc := docs[0]

	// Extract patent statuses
	for _, s := range doc.Statuses.Entries {
		data.Statuses = append(data.Statuses, PatentStatus{
			Date:        strings.TrimSpace(s.ChangeDate),
			Code:        strings.TrimSpace(s.StatusCode),
			Description: strings.TrimSpace(s.Text),
		})
	}

	// Extract dossier events
	for _, ed := range doc.EventsData {
		evt := ed.DossierEvent
		code := strings.TrimSpace(evt.EventCode)
		desc := strings.TrimSpace(evt.EventText.Text)
		regEvent := RegisterEvent{
			ID:          strings.TrimSpace(evt.ID),
			Date:        strings.TrimSpace(evt.EventDate.Date),
			EventCode:   code,
			Description: desc,
			Category:    categorizeRegisterEvent(code, desc),
			GazetteNum:  strings.TrimSpace(evt.GazetteRef.GazetteNum),
			GazetteDate: strings.TrimSpace(evt.GazetteRef.Date),
		}
		data.Events = append(data.Events, regEvent)
	}

	// Sort events chronologically (oldest first)
	sort.Slice(data.Events, func(i, j int) bool {
		return data.Events[i].Date < data.Events[j].Date
	})

	return data, nil
}

// --- Number Conversion XML Structs ---

type numberConversionXML struct {
	XMLName         xml.Name `xml:"world-patent-data"`
	Standardization struct {
		InputFormat  string `xml:"inputFormat,attr"`
		OutputFormat string `xml:"outputFormat,attr"`
		Output       struct {
			PublicationRef struct {
				DocumentID struct {
					Type      string `xml:"document-id-type,attr"`
					Country   string `xml:"country"`
					DocNumber string `xml:"doc-number"`
					Kind      string `xml:"kind"`
					Date      string `xml:"date"`
				} `xml:"document-id"`
			} `xml:"publication-reference"`
		} `xml:"output"`
	} `xml:"standardization"`
}

// ParseNumberConversion parses EPO number conversion XML into a NumberConversionData struct.
func ParseNumberConversion(xmlData string) (*NumberConversionData, error) {
	var raw numberConversionXML
	if err := xml.Unmarshal([]byte(xmlData), &raw); err != nil {
		return nil, &XMLParseError{
			Parser:    "ParseNumberConversion",
			Element:   "world-patent-data",
			XMLSample: truncateXML(xmlData, 200),
			Cause:     err,
		}
	}

	std := raw.Standardization
	docID := std.Output.PublicationRef.DocumentID

	if docID.DocNumber == "" {
		return nil, &DataValidationError{
			Parser:       "ParseNumberConversion",
			MissingField: "doc-number",
			Message:      "output document-id missing or incomplete",
		}
	}

	return &NumberConversionData{
		InputFormat:  strings.TrimSpace(std.InputFormat),
		OutputFormat: strings.TrimSpace(std.OutputFormat),
		Country:      strings.TrimSpace(docID.Country),
		DocNumber:    strings.TrimSpace(docID.DocNumber),
		Kind:         strings.TrimSpace(docID.Kind),
		Date:         strings.TrimSpace(docID.Date),
	}, nil
}

// --- Classification Schema XML Structs ---

type classificationSchemaXML struct {
	XMLName              xml.Name `xml:"world-patent-data"`
	ClassificationScheme struct {
		CPC struct {
			ClassScheme struct {
				SchemeType string                  `xml:"scheme-type,attr"`
				Items      []classificationItemXML `xml:"classification-item"`
			} `xml:"class-scheme"`
		} `xml:"cpc"`
	} `xml:"classification-scheme"`
}

type classificationItemXML struct {
	Level       int    `xml:"level,attr"`
	HasChildren bool   `xml:"has-children,attr"`
	Symbol      string `xml:"classification-symbol"`
	Title       struct {
		TitleParts []struct {
			Text string `xml:"text"`
		} `xml:"title-part"`
	} `xml:"class-title"`
	Children []classificationItemXML `xml:"classification-item"`
}

// ParseClassificationSchema parses EPO classification schema XML into a ClassificationData struct.
func ParseClassificationSchema(xmlData string) (*ClassificationData, error) {
	var raw classificationSchemaXML
	if err := xml.Unmarshal([]byte(xmlData), &raw); err != nil {
		return nil, &XMLParseError{
			Parser:    "ParseClassificationSchema",
			Element:   "world-patent-data",
			XMLSample: truncateXML(xmlData, 200),
			Cause:     err,
		}
	}

	scheme := raw.ClassificationScheme.CPC.ClassScheme
	items := scheme.Items

	if len(items) == 0 {
		return &ClassificationData{
			Children: []ClassificationChild{},
		}, nil
	}

	// When ancestors=true, the XML returns a nested hierarchy where ancestor items
	// wrap the target symbol. Walk down the tree collecting ancestors until we reach
	// the deepest item (the one the user requested).
	// Heuristic: ancestor items have exactly 1 child (the next level in the linear chain).
	// The target item has 0 children (leaf) or 2+ children (branching point).
	// This works because the EPO API returns a single-path chain from root to target.
	item := items[0]
	var ancestors []ClassificationChild

	for len(item.Children) == 1 {
		ancestorTitle := ""
		if len(item.Title.TitleParts) > 0 {
			ancestorTitle = strings.TrimSpace(item.Title.TitleParts[0].Text)
		}
		ancestors = append(ancestors, ClassificationChild{
			Symbol:      strings.TrimSpace(item.Symbol),
			Title:       ancestorTitle,
			Level:       item.Level,
			HasChildren: item.HasChildren,
		})
		item = item.Children[0]
	}

	title := ""
	if len(item.Title.TitleParts) > 0 {
		title = strings.TrimSpace(item.Title.TitleParts[0].Text)
	}

	children := []ClassificationChild{}
	for _, child := range item.Children {
		childTitle := ""
		if len(child.Title.TitleParts) > 0 {
			childTitle = strings.TrimSpace(child.Title.TitleParts[0].Text)
		}
		children = append(children, ClassificationChild{
			Symbol:      strings.TrimSpace(child.Symbol),
			Title:       childTitle,
			Level:       child.Level,
			HasChildren: child.HasChildren,
		})
	}

	return &ClassificationData{
		Symbol:      strings.TrimSpace(item.Symbol),
		Title:       title,
		Level:       item.Level,
		SchemeType:  strings.TrimSpace(scheme.SchemeType),
		HasChildren: item.HasChildren,
		Children:    children,
		Ancestors:   ancestors,
	}, nil
}
