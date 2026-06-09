package epo_ops

import (
	"os"
	"testing"
)

// Real INPADOC legal status uses L-codes well beyond L050EP (up to L5xxEP); the parser must
// keep all of them, not silently cap at a fixed range.
func TestParseLegal_UnboundedLCodes(t *testing.T) {
	b, err := os.ReadFile("testdata/legal.xml")
	if err != nil {
		t.Fatal(err)
	}
	data, err := ParseLegal(string(b))
	if err != nil {
		t.Fatalf("ParseLegal: %v", err)
	}
	maxSeen := ""
	hasHigh := false
	for _, ev := range data.LegalEvents {
		for code := range ev.Fields {
			if code > maxSeen {
				maxSeen = code
			}
			if code >= "L100EP" {
				hasHigh = true
			}
		}
	}
	if !hasHigh {
		t.Errorf("no L-code >= L100EP captured (highest %q); high codes are being dropped", maxSeen)
	}
	t.Logf("highest L-code captured: %s", maxSeen)
}
