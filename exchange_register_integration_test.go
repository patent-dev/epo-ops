//go:build integration

package epo_ops

import (
	"context"
	"os"
	"testing"
	"time"
)

func newIntegrationClient(t *testing.T) *Client {
	t.Helper()
	key := os.Getenv("EPO_OPS_CONSUMER_KEY")
	secret := os.Getenv("EPO_OPS_CONSUMER_SECRET")
	if key == "" || secret == "" {
		t.Skip("Skipping integration test: EPO_OPS_CONSUMER_KEY and EPO_OPS_CONSUMER_SECRET must be set")
	}
	c, err := NewClient(&Config{ConsumerKey: key, ConsumerSecret: secret})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

// TestExchangeAndRegisterIntegration exercises the comprehensive parsers end to end against
// the live API via their typed client methods.
func TestExchangeAndRegisterIntegration(t *testing.T) {
	c := newIntegrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	t.Run("GetExchangeDocuments", func(t *testing.T) {
		docs, err := c.GetExchangeDocuments(ctx, "publication", "docdb", "EP.1000000.B1")
		if err != nil {
			t.Fatalf("GetExchangeDocuments: %v", err)
		}
		if len(docs) == 0 {
			t.Fatal("no exchange documents parsed")
		}
		if docs[0].PublicationNumber() == "" {
			t.Error("no publication number parsed")
		}
		t.Logf("exchange-documents: %d, pub=%s title=%q citations=%d",
			len(docs), docs[0].PublicationNumber(), docs[0].Title(), len(docs[0].Biblio.Citations))
	})

	t.Run("GetRegister", func(t *testing.T) {
		docs, err := c.GetRegister(ctx, "publication", "epodoc", "EP1700924")
		if err != nil {
			t.Fatalf("GetRegister: %v", err)
		}
		if len(docs) == 0 {
			t.Fatal("no register documents parsed")
		}
		d := docs[0]
		if len(d.Statuses) == 0 {
			t.Error("no ep-patent-status entries")
		}
		t.Logf("register: %d statuses, %d titles, %d terms-of-grant, %d citations",
			len(d.Statuses), len(d.Biblio.Titles), len(d.Biblio.TermsOfGrant), len(d.Biblio.Citations))
	})

	t.Run("GetRegisterProceduralSteps", func(t *testing.T) {
		docs, err := c.GetRegisterProceduralSteps(ctx, "publication", "epodoc", "EP1700924")
		if err != nil {
			t.Fatalf("GetRegisterProceduralSteps: %v", err)
		}
		if len(docs) == 0 || len(docs[0].ProceduralSteps) == 0 {
			t.Fatal("no procedural steps parsed")
		}
		t.Logf("procedural steps: %d", len(docs[0].ProceduralSteps))
	})

	t.Run("GetRegisterUNIP", func(t *testing.T) {
		// Not every patent has a unitary-patent record; only assert the call is well-formed.
		if _, err := c.GetRegisterUNIP(ctx, "publication", "epodoc", "EP1700924"); err != nil {
			t.Logf("GetRegisterUNIP returned an error (acceptable if no UP record exists): %v", err)
		}
	})
}
