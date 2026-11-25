package epo_ops

import (
	"encoding/xml"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// reachability_test.go proves, on the recorded responses, that the public parse results lose
// nothing: every text value and every attribute value OPS sent can be read back from the
// struct a parser returns.
//
// It complements TestElementCoverage (decode_fixtures_test.go), which checks xml tags by
// reflection and so cannot see into the custom UnmarshalXML methods the register and legal
// parsers rely on, and which looks at the first document of a fixture only. This test looks at
// the values themselves, in EVERY document of a fixture: it walks the XML below the parser's
// root element, collects each element's own text and each attribute value, and requires each
// one to equal some string held by the parsed result. A value that exists only inside an
// Unparsed catch-all does not count, and TestNothingUnparsed requires those to stay empty.

// fixtureValue is one text or attribute value of a recorded response with where it stands.
type fixtureValue struct {
	where string // element path, plus " @attr" for an attribute
	value string
}

// collectFixtureValues returns every non-empty element text and attribute value found in the
// subtrees rooted at an element named root (every such subtree), or in the whole document
// when root is empty. Namespace declarations are not data and are skipped.
func collectFixtureValues(t *testing.T, data []byte, root string) []fixtureValue {
	t.Helper()
	dec := xml.NewDecoder(strings.NewReader(string(data)))
	var out []fixtureValue
	var stack []string
	var texts []*strings.Builder
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch tk := tok.(type) {
		case xml.StartElement:
			if len(stack) == 0 && root != "" && tk.Name.Local != root {
				continue
			}
			stack = append(stack, tk.Name.Local)
			texts = append(texts, &strings.Builder{})
			path := "/" + strings.Join(stack, "/")
			for _, a := range tk.Attr {
				if a.Name.Space == "xmlns" || a.Name.Local == "xmlns" {
					continue
				}
				if v := strings.TrimSpace(a.Value); v != "" {
					out = append(out, fixtureValue{path + " @" + a.Name.Local, v})
				}
			}
		case xml.CharData:
			if len(texts) > 0 {
				texts[len(texts)-1].Write(tk)
			}
		case xml.EndElement:
			if len(stack) == 0 {
				continue
			}
			if v := strings.TrimSpace(texts[len(texts)-1].String()); v != "" {
				out = append(out, fixtureValue{"/" + strings.Join(stack, "/"), v})
			}
			stack = stack[:len(stack)-1]
			texts = texts[:len(texts)-1]
		}
	}
	if len(out) == 0 {
		t.Fatalf("no values collected below <%s>", root)
	}
	return out
}

var rawElementType = reflect.TypeOf(RawElement{})

// collectStructStrings gathers every string a parse result holds (trimmed), with numbers and
// booleans in their decimal / true-false form. The Unparsed catch-alls are left out on purpose.
func collectStructStrings(v reflect.Value, into map[string]bool) {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			collectStructStrings(v.Elem(), into)
		}
	case reflect.Struct:
		if v.Type() == rawElementType {
			return
		}
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() {
				collectStructStrings(v.Field(i), into)
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			collectStructStrings(v.Index(i), into)
		}
	case reflect.Map:
		for _, k := range v.MapKeys() {
			collectStructStrings(k, into)
			collectStructStrings(v.MapIndex(k), into)
		}
	case reflect.String:
		into[strings.TrimSpace(v.String())] = true
	case reflect.Int, reflect.Int64:
		into[strconv.FormatInt(v.Int(), 10)] = true
	case reflect.Bool:
		into[strconv.FormatBool(v.Bool())] = true
	}
}

// collectUnparsed returns the names of all elements that ended up in an Unparsed catch-all.
func collectUnparsed(v reflect.Value, into *[]string) {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			collectUnparsed(v.Elem(), into)
		}
	case reflect.Struct:
		if v.Type() == rawElementType {
			*into = append(*into, v.Interface().(RawElement).XMLName.Local)
			return
		}
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() {
				collectUnparsed(v.Field(i), into)
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			collectUnparsed(v.Index(i), into)
		}
	}
}

type reachabilityCase struct {
	parser  string
	fixture string
	root    string // parser's root element; "" = whole response
	parse   func(string) (any, error)
	// notCarried lists places whose values the result does not hold, each for a stated reason.
	notCarried map[string]string
}

func reachabilityCases() []reachabilityCase {
	exchange := func(s string) (any, error) { return ParseExchangeDocuments(s) }
	biblioDocs := func(s string) (any, error) { return ParseBiblioDocuments(s) }
	register := func(s string) (any, error) { return ParseRegister(s) }
	family := func(s string) (any, error) { return ParseFamily(s) }
	legal := func(s string) (any, error) { return ParseLegal(s) }
	search := func(s string) (any, error) { return ParseSearch(s) }
	inquiry := func(s string) (any, error) { return ParseImageInquiry(s) }

	var cases []reachabilityCase
	for _, f := range []string{"biblio.xml", "biblio-kindless-ep2400812.xml", "family-biblio.xml", "search.xml", "abstract.xml"} {
		cases = append(cases,
			reachabilityCase{parser: "ParseExchangeDocuments", fixture: f, root: "exchange-document", parse: exchange},
			reachabilityCase{parser: "ParseBiblioDocuments", fixture: f, root: "exchange-document", parse: biblioDocs},
		)
	}
	for _, f := range []string{
		"register-biblio.xml", "register-biblio-ep2771468.xml", "register-biblio-ep4303979.xml",
		"register-biblio-ep4306318.xml", "register-biblio-ep4654614.xml", "register-events.xml",
		"register-procedural-steps.xml", "register-unip.xml", "register-unip-ep3970521.xml",
	} {
		cases = append(cases, reachabilityCase{parser: "ParseRegister", fixture: f, root: "register-document", parse: register})
	}
	for _, f := range []string{"family.xml", "family-biblio.xml", "legal.xml"} {
		cases = append(cases, reachabilityCase{parser: "ParseFamily", fixture: f, parse: family})
	}
	cases = append(cases,
		reachabilityCase{parser: "ParseLegal", fixture: "legal.xml", parse: legal, notCarried: map[string]string{
			"/world-patent-data/patent-family @legal": "always \"true\" for a legal response; it restates the request",
		}},
		reachabilityCase{parser: "ParseSearch", fixture: "search.xml", parse: search},
		reachabilityCase{parser: "ParseImageInquiry", fixture: "image-inquiry.xml", parse: inquiry},
	)
	return cases
}

func TestEveryValueReachable(t *testing.T) {
	for _, c := range reachabilityCases() {
		t.Run(c.parser+"/"+c.fixture, func(t *testing.T) {
			data := readFixtureBytes(t, c.fixture)
			result, err := c.parse(string(data))
			if err != nil {
				t.Fatal(err)
			}
			have := map[string]bool{}
			collectStructStrings(reflect.ValueOf(result), have)

			missing := map[string]bool{}
			used := map[string]bool{}
			for _, fv := range collectFixtureValues(t, data, c.root) {
				if have[fv.value] {
					continue
				}
				if _, ok := c.notCarried[fv.where]; ok {
					used[fv.where] = true
					continue
				}
				missing[fv.where+" = "+strconv.Quote(fv.value)] = true
			}
			if len(missing) > 0 {
				list := make([]string, 0, len(missing))
				for m := range missing {
					list = append(list, m)
				}
				sort.Strings(list)
				if len(list) > 40 {
					list = append(list[:40], "...")
				}
				t.Errorf("%d value(s) of the response cannot be read from the result of %s:\n  %s", len(missing), c.parser, strings.Join(list, "\n  "))
			}
			for where := range c.notCarried {
				if !used[where] {
					t.Errorf("notCarried entry %q is no longer needed; remove it", where)
				}
			}
		})
	}
}

// TestNothingUnparsed requires the typed fields to model every element of the recorded
// responses: the Unparsed catch-alls exist for elements OPS may send that no recording has
// shown yet, and must stay empty on the recordings.
func TestNothingUnparsed(t *testing.T) {
	for _, c := range reachabilityCases() {
		t.Run(c.parser+"/"+c.fixture, func(t *testing.T) {
			result, err := c.parse(string(readFixtureBytes(t, c.fixture)))
			if err != nil {
				t.Fatal(err)
			}
			var names []string
			collectUnparsed(reflect.ValueOf(result), &names)
			if len(names) > 0 {
				sort.Strings(names)
				t.Errorf("elements left in Unparsed: %s", strings.Join(names, ", "))
			}
		})
	}
}
