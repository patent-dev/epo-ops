package epo_ops

import (
	"encoding/xml"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// This is the XML analogue of a strict-decode (DisallowUnknownFields) JSON test.
//
// encoding/xml silently drops any element or attribute that no struct field maps
// to, so an incomplete struct loses real data without any error. EPO OPS returns
// large XML documents and the per-endpoint parsers deliberately project only the
// subset each typed result needs (e.g. biblioXML keeps publication-reference,
// titles, parties, IPC/CPC but not the full bibliographic-data tree). That is a
// legitimate design choice, but it is exactly the kind of silent drop that hides
// regressions: if OPS adds a child to a subtree the parser DOES model, or a typo
// removes a field, the data vanishes with no error.
//
// These tests pin the contract. For each recorded fixture they collect every
// element path and attribute present in the real XML, then assert that the raw
// unmarshal struct captures every one of them via an `xml:"..."` tag -- EXCEPT
// the paths each case explicitly lists in `unmodeled`. The `unmodeled` set is the
// documented, reviewed boundary of what a parser intentionally ignores; anything
// outside both the struct AND the allowlist fails the test, so a newly appearing
// (or newly dropped) element cannot slip through unnoticed.
//
// Matching follows encoding/xml semantics: by local element name (namespace
// prefixes such as ops:/ns2: are ignored, exactly as the decoder ignores them),
// rooted at the element the parser decodes from.

// schemaPaths reflects over a raw XML struct type and returns the set of element
// paths and attribute paths it can decode. Element paths look like
// "/Root/Child/Leaf"; attribute paths look like "/Root/Child @attrName".
//
// rootOverride, when non-empty, is used as the root element name instead of the
// struct's XMLName tag. Parsers that decode from a sub-element (ParseExchangeDocuments
// decodes <exchange-document>, ParseRegister decodes <register-document>) pass it.
func schemaPaths(t reflect.Type, rootOverride string) (elements, attrs map[string]bool) {
	elements = map[string]bool{}
	attrs = map[string]bool{}
	seen := map[reflect.Type]bool{}

	root := ""
	t = derefType(t)
	if rootOverride != "" {
		root = "/" + rootOverride
	} else if t.Kind() == reflect.Struct {
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.Name == "XMLName" {
				if name := tagName(f.Tag.Get("xml")); name != "" {
					root = "/" + name
				}
			}
		}
	}
	elements[root] = true
	collectStruct(t, root, elements, attrs, seen)
	return elements, attrs
}

func collectStruct(t reflect.Type, prefix string, elements, attrs map[string]bool, seen map[reflect.Type]bool) {
	t = derefType(t)
	if t.Kind() != reflect.Struct {
		return
	}
	// Guard against recursive types (classificationItemXML nests itself).
	if seen[t] {
		return
	}
	seen[t] = true
	defer delete(seen, t)

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" { // unexported
			continue
		}
		tag := f.Tag.Get("xml")
		if f.Name == "XMLName" {
			continue
		}
		// Anonymous embedded struct fields are promoted by encoding/xml: their
		// fields decode at the SAME level as the embedding struct, with no path
		// segment of their own (e.g. RegisterBiblio embeds BibliographicData).
		if f.Anonymous && tag == "" {
			collectStruct(f.Type, prefix, elements, attrs, seen)
			continue
		}
		name, opts := parseXMLTag(tag)
		if hasOpt(opts, "attr") {
			attrName := name
			if attrName == "" {
				attrName = f.Name
			}
			attrs[prefix+" @"+attrName] = true
			continue
		}
		if hasOpt(opts, "chardata") || hasOpt(opts, "innerxml") || hasOpt(opts, "comment") || hasOpt(opts, "cdata") {
			continue
		}
		if name == "-" {
			continue
		}
		if name == "" {
			name = f.Name
		}
		// Expand ">"-separated nested element paths.
		segs := strings.Split(name, ">")
		p := prefix
		for _, s := range segs {
			p += "/" + s
			elements[p] = true
		}
		collectStruct(f.Type, p, elements, attrs, seen)
	}
}

func derefType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice || t.Kind() == reflect.Array || t.Kind() == reflect.Map {
		t = t.Elem()
	}
	return t
}

func tagName(tag string) string {
	n, _ := parseXMLTag(tag)
	return n
}

func parseXMLTag(tag string) (name string, opts []string) {
	if tag == "" {
		return "", nil
	}
	parts := strings.Split(tag, ",")
	name = strings.TrimSpace(parts[0])
	// Strip any namespace prefix in the tag (e.g. "ns name") to local name.
	if idx := strings.LastIndex(name, " "); idx >= 0 {
		name = name[idx+1:]
	}
	return name, parts[1:]
}

func hasOpt(opts []string, want string) bool {
	for _, o := range opts {
		if o == want {
			return true
		}
	}
	return false
}

// fixturePaths walks the actual fixture XML and returns the element paths and
// attribute paths present, rooted at the given start element (local name) and
// using local element names. When startLocal is empty the document root is used.
func fixturePaths(data []byte, startLocal string) (elements, attrs map[string]bool) {
	elements = map[string]bool{}
	attrs = map[string]bool{}

	dec := xml.NewDecoder(strings.NewReader(string(data)))
	var stack []string
	capturing := startLocal == ""
	depth := 0 // depth at which capture started, to know when to stop
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
			cur := strings.Join(stack, "") + "/" + local
			stack = append(stack, "/"+local)
			elements[cur] = true
			for _, a := range se.Attr {
				if a.Name.Space == "xmlns" || a.Name.Local == "xmlns" {
					continue
				}
				attrs[cur+" @"+a.Name.Local] = true
			}
		case xml.EndElement:
			if !capturing {
				continue
			}
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			depth--
			if startLocal != "" && depth == 0 {
				// Finished the first matching subtree; one is enough.
				return elements, attrs
			}
		}
	}
	return elements, attrs
}

// completenessCase pins one fixture against its raw unmarshal struct.
type completenessCase struct {
	name string
	data []byte
	raw  any // zero value of the raw unmarshal struct
	// root overrides the decode root element local name for parsers that decode
	// from a sub-element rather than the document root (empty = use XMLName).
	root string
	// unmodeled lists exact element/attribute paths present in the fixture that the
	// parser intentionally does NOT model. Each entry is the documented boundary of
	// a deliberate projection; the test fails if the fixture carries any path that
	// is neither in the struct nor here (or under an unmodeledPrefix).
	unmodeled []string
	// skipRoundTrip documents that the raw struct cannot be re-marshaled by
	// encoding/xml because it captures open-ended data through a custom
	// UnmarshalXML into a map (legalEventXML.Fields, MilestoneDates): reflection
	// marshaling rejects map types and there is no symmetric MarshalXML. Lossless
	// parsing for these is proven by element coverage + key fields + the dedicated
	// dynamic-field tests (xml_dynamic_test.go) instead.
	skipRoundTrip string
	// unmodeledPrefixes marks whole intentionally-unmodeled subtrees by path prefix
	// (e.g. a parser that captures party NAMES but not the full postal addressbook).
	// Any fixture path that begins with one of these prefixes is treated as a
	// documented projection boundary. Use sparingly and comment each one in the
	// case so the boundary stays reviewable.
	unmodeledPrefixes []string
}

func assertCompleteness(t *testing.T, c completenessCase) {
	t.Helper()
	schemaEls, schemaAttrs := schemaPaths(reflect.TypeOf(c.raw), c.root)
	fixEls, fixAttrs := fixturePaths(c.data, c.root)

	allow := map[string]bool{}
	for _, p := range c.unmodeled {
		allow[p] = true
	}
	usedAllow := map[string]bool{}

	allowedByPrefix := func(p string) bool {
		for _, pre := range c.unmodeledPrefixes {
			if strings.Contains(p, pre) {
				return true
			}
		}
		return false
	}

	var missingEls, missingAttrs []string
	for p := range fixEls {
		if schemaEls[p] {
			continue
		}
		if allow[p] {
			usedAllow[p] = true
			continue
		}
		if allowedByPrefix(p) {
			continue
		}
		missingEls = append(missingEls, p)
	}
	for a := range fixAttrs {
		if schemaAttrs[a] {
			continue
		}
		if allow[a] {
			usedAllow[a] = true
			continue
		}
		if allowedByPrefix(a) {
			continue
		}
		missingAttrs = append(missingAttrs, a)
	}
	sort.Strings(missingEls)
	sort.Strings(missingAttrs)

	if len(missingEls) > 0 {
		t.Errorf("%s: %d element path(s) present in the response are NOT captured by %s and NOT in the documented unmodeled set (data silently dropped):\n  %s",
			c.name, len(missingEls), reflect.TypeOf(c.raw).Name(), strings.Join(missingEls, "\n  "))
	}
	if len(missingAttrs) > 0 {
		t.Errorf("%s: %d attribute(s) present are NOT captured by %s and NOT in the documented unmodeled set:\n  %s",
			c.name, len(missingAttrs), reflect.TypeOf(c.raw).Name(), strings.Join(missingAttrs, "\n  "))
	}

	// A stale allowlist entry (no longer present in the fixture) is a smell: the
	// boundary drifted. Flag it so the list stays honest.
	var stale []string
	for _, p := range c.unmodeled {
		if !usedAllow[p] && !schemaEls[p] && !schemaAttrs[p] {
			stale = append(stale, p)
		}
	}
	sort.Strings(stale)
	if len(stale) > 0 {
		t.Errorf("%s: %d unmodeled allowlist entr(y/ies) no longer present in the fixture (remove them):\n  %s",
			c.name, len(stale), strings.Join(stale, "\n  "))
	}
}
