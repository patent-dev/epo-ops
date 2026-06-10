package epo_ops

import "testing"

// TestParseImageInquiry_LinkAttribute covers the real OPS shape, where the image
// link is a "link" attribute on document-instance (not a nested
// document-instance-link/@href element).
func TestParseImageInquiry_LinkAttribute(t *testing.T) {
	xmlData := `<ops:world-patent-data xmlns="http://www.epo.org/exchange" xmlns:ops="http://ops.epo.org">
  <ops:document-inquiry>
    <ops:inquiry-result>
      <ops:document-instance system="ops.epo.org" number-of-pages="11" desc="FullDocument" link="published-data/images/EP/1000000/B1/fullimage">
        <ops:document-format-options>
          <ops:document-format>application/tiff</ops:document-format>
          <ops:document-format>application/pdf</ops:document-format>
        </ops:document-format-options>
      </ops:document-instance>
      <ops:document-instance system="ops.epo.org" number-of-pages="6" desc="Drawing" link="published-data/images/EP/1000000/B1/thumbnail"/>
    </ops:inquiry-result>
  </ops:document-inquiry>
</ops:world-patent-data>`

	data, err := ParseImageInquiry(xmlData)
	if err != nil {
		t.Fatalf("ParseImageInquiry: %v", err)
	}
	if len(data.DocumentInstances) != 2 {
		t.Fatalf("got %d instances, want 2", len(data.DocumentInstances))
	}

	full := data.DocumentInstances[0]
	if full.Description != "FullDocument" {
		t.Errorf("Description = %q, want FullDocument", full.Description)
	}
	if full.Link != "published-data/images/EP/1000000/B1/fullimage" {
		t.Errorf("Link = %q, want the link attribute value", full.Link)
	}
	if full.NumberOfPages != 11 {
		t.Errorf("NumberOfPages = %d, want 11", full.NumberOfPages)
	}
	if len(full.Formats) != 2 {
		t.Errorf("Formats = %v, want 2 entries", full.Formats)
	}

	if data.DocumentInstances[1].Link != "published-data/images/EP/1000000/B1/thumbnail" {
		t.Errorf("Drawing link = %q", data.DocumentInstances[1].Link)
	}
}

// TestParseImageInquiry_SkipsLinklessInstance ensures one instance with no link
// does not fail the whole inquiry.
func TestParseImageInquiry_SkipsLinklessInstance(t *testing.T) {
	xmlData := `<ops:world-patent-data xmlns="http://www.epo.org/exchange" xmlns:ops="http://ops.epo.org">
  <ops:document-inquiry>
    <ops:inquiry-result>
      <ops:document-instance desc="Broken"/>
      <ops:document-instance desc="Drawing" number-of-pages="3" link="published-data/images/EP/1000000/B1/thumbnail"/>
    </ops:inquiry-result>
  </ops:document-inquiry>
</ops:world-patent-data>`

	data, err := ParseImageInquiry(xmlData)
	if err != nil {
		t.Fatalf("ParseImageInquiry: %v", err)
	}
	if len(data.DocumentInstances) != 1 || data.DocumentInstances[0].Description != "Drawing" {
		t.Fatalf("expected only the linked Drawing instance, got %+v", data.DocumentInstances)
	}
}
