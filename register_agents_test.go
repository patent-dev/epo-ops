package epo_ops

import (
	"os"
	"reflect"
	"testing"
)

// The fixtures are unedited register biblio responses. EP4303979 and EP4306318 changed
// representative, so their older blocks must not count as current; EP4306318's newest block
// carries the gazette value "N/P" instead of a bulletin number.
func TestParseRegister_RepresentativeBlocks(t *testing.T) {
	tests := []struct {
		fixture string
		current []Agent
		gazette []string
	}{
		{
			fixture: "testdata/register-biblio-ep4303979.xml",
			current: []Agent{{Sequence: "1", RepType: "agent", Name: "Kraus & Lederer PartGmbB", Country: "DE"}},
			gazette: []string{"2025/48", "2024/02"},
		},
		{
			fixture: "testdata/register-biblio-ep4654614.xml",
			current: []Agent{{Sequence: "1", RepType: "agent", Name: "Kraus & Lederer PartGmbB", Country: "DE"}},
			gazette: []string{"2025/48"},
		},
		{
			fixture: "testdata/register-biblio-ep4306318.xml",
			current: []Agent{{Sequence: "1", RepType: "agent", Name: "Kraus & Lederer PartGmbB, et al", Country: "DE"}},
			gazette: []string{"N/P", "2024/03"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			raw, err := os.ReadFile(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}
			docs, err := ParseRegister(string(raw))
			if err != nil {
				t.Fatal(err)
			}
			if len(docs) != 1 {
				t.Fatalf("got %d documents, want 1", len(docs))
			}
			b := docs[0].Biblio
			if got := b.CurrentAgents(); !reflect.DeepEqual(got, tt.current) {
				t.Errorf("CurrentAgents = %+v, want %+v", got, tt.current)
			}
			var gazette []string
			var total int
			for _, g := range b.AgentGroups {
				gazette = append(gazette, g.GazetteNum)
				total += len(g.Agents)
			}
			if !reflect.DeepEqual(gazette, tt.gazette) {
				t.Errorf("gazettes = %v, want %v", gazette, tt.gazette)
			}
			if len(b.Agents) != total {
				t.Errorf("flat Agents has %d entries, want all %d", len(b.Agents), total)
			}
		})
	}
}

func TestCurrentAgents_NoRepresentative(t *testing.T) {
	if got := (RegisterBiblio{}).CurrentAgents(); got != nil {
		t.Errorf("CurrentAgents = %+v, want nil", got)
	}
}
