package epo_ops

import (
	_ "embed"
	"encoding/xml"
	"fmt"
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

// imageInquiryXML is the internal structure for unmarshaling image inquiry XML
type imageInquiryXML struct {
	XMLName         xml.Name `xml:"world-patent-data"`
	DocumentInquiry struct {
		InquiryResult struct {
			DocumentInstances []struct {
				Desc          string `xml:"desc,attr"`
				NumberOfPages int    `xml:"number-of-pages,attr"`
				DocType       string `xml:"doc-type,attr"`
				Link          string `xml:"link,attr"`
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
		return nil, fmt.Errorf("failed to parse image inquiry XML: %w", err)
	}

	result := &ImageInquiry{
		DocumentInstances: make([]DocumentInstance, 0, len(raw.DocumentInquiry.InquiryResult.DocumentInstances)),
	}

	for _, inst := range raw.DocumentInquiry.InquiryResult.DocumentInstances {
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
			Link:          inst.Link,
			NumberOfPages: inst.NumberOfPages,
			Formats:       formats,
			DocType:       inst.DocType,
		})
	}

	return result, nil
}
