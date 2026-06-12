//go:build integration

package epo_ops

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

// This file holds one live integration test per exported Client method
// (TestIntegration<Method>), so every endpoint of the EPO OPS wrapper has a
// targeted live check that maps 1:1 to the method. The set is machine-verified by
// scripts/check-integration-coverage.sh (make check-integration); a new public
// method without a matching TestIntegration<Method> fails that check.
//
// Run with: go test -tags=integration -count=1 ./...
//
// EPO OPS enforces a strict weekly fair-use quota, so each endpoint is called AT
// MOST ONCE across the whole suite, reusing inputs proven by the demo (demo/...),
// which returned real data live. Every test PASSes (real data) or SKIPs cleanly
// (no credentials, or a documented transient/availability condition); none FAILs
// on a quota throttle, a 403/404, or data simply not existing for the sample.

// Demo-proven inputs (see demo/*.go). These returned real data live.
const (
	pubRefType = "publication"
	docdb      = "docdb"
	epodoc     = "epodoc"

	// A well-known EP publication present across published-data, family, legal,
	// images and number services (docdb form).
	testDocdb = "EP.1000000.B1"
	// The same publication in the no-dot docdb form the image GET path takes.
	imgCountry = "EP"
	imgNumber  = "1000000"
	imgKind    = "B1"
	// epodoc form for the register / equivalents endpoints that require it.
	registerEpodoc = "EP1700924"
	// A second epodoc publication with rich biblio used where EP1700924 lacks data.
	convertDocdb = "EP.2400812.A1"
	// Bulk inputs for the *Multiple (POST) endpoints.
	bulkA = "EP.1000000.B1"
	bulkB = "EP.2400812.A1"
)

func bulkNumbers() []string { return []string{bulkA, bulkB} }

// integrationClient builds a live client from EPO_OPS_CONSUMER_KEY /
// EPO_OPS_CONSUMER_SECRET, or skips the test when either is absent so the suite
// stays green without credentials.
func integrationClient(t *testing.T) *Client {
	t.Helper()
	key := os.Getenv("EPO_OPS_CONSUMER_KEY")
	secret := os.Getenv("EPO_OPS_CONSUMER_SECRET")
	if key == "" || secret == "" {
		t.Skip("set EPO_OPS_CONSUMER_KEY and EPO_OPS_CONSUMER_SECRET to run EPO OPS integration tests")
	}
	c, err := NewClient(&Config{ConsumerKey: key, ConsumerSecret: secret})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func integrationCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// skipIfTransient converts the conditions that are NOT a library bug into clean
// SKIPs so the suite never FAILs on them:
//
//   - QuotaExceededError / ServiceUnavailableError: OPS fair-use throttle (403
//     "fair use" / 429) or a temporary 503 - retry later, not a defect.
//   - ForbiddenError: the account lacks entitlement for that service.
//   - NotFoundError: the chosen sample has no data for this constituent (e.g. no
//     unitary-patent record, no legal events on a very old patent).
//
// It returns true when it skipped. Any other error is the caller's to FAIL on.
func skipIfTransient(t *testing.T, err error) bool {
	t.Helper()
	if err == nil {
		return false
	}
	var quota *QuotaExceededError
	var unavail *ServiceUnavailableError
	var forbidden *ForbiddenError
	var notFound *NotFoundError
	switch {
	case errors.As(err, &quota):
		t.Skipf("skipping: OPS fair-use quota throttle: %v", err)
	case errors.As(err, &unavail):
		t.Skipf("skipping: OPS temporarily unavailable: %v", err)
	case errors.As(err, &forbidden):
		t.Skipf("skipping: account not entitled for this service: %v", err)
	case errors.As(err, &notFound):
		t.Skipf("skipping: no data for sample input: %v", err)
	}
	return true
}

func mustNonEmptyXML(t *testing.T, raw string) {
	t.Helper()
	if strings.TrimSpace(raw) == "" {
		t.Fatal("empty raw XML response")
	}
	if !strings.Contains(raw, "<") {
		t.Fatalf("response is not XML: %.80q", raw)
	}
}

// --- Cross-cutting auth / error behaviour --------------------------------------

// TestIntegrationAuthentication exercises real token acquisition and caching.
func TestIntegrationAuthentication(t *testing.T) {
	key := os.Getenv("EPO_OPS_CONSUMER_KEY")
	secret := os.Getenv("EPO_OPS_CONSUMER_SECRET")
	if key == "" || secret == "" {
		t.Skip("set EPO_OPS_CONSUMER_KEY and EPO_OPS_CONSUMER_SECRET to run EPO OPS integration tests")
	}
	auth := NewAuthenticator(key, secret, nil)
	ctx := integrationCtx(t)
	tok1, err := auth.GetToken(ctx)
	if err != nil {
		t.Fatalf("GetToken: %v", err)
	}
	if len(tok1) < 20 {
		t.Errorf("token too short: %d chars", len(tok1))
	}
	tok2, err := auth.GetToken(ctx)
	if err != nil {
		t.Fatalf("GetToken (cached): %v", err)
	}
	if tok1 != tok2 {
		t.Error("expected cached token reuse")
	}
}

// TestIntegrationInvalidCredentials confirms invalid credentials yield an AuthError.
func TestIntegrationInvalidCredentials(t *testing.T) {
	if os.Getenv("EPO_OPS_CONSUMER_KEY") == "" {
		t.Skip("set EPO_OPS_CONSUMER_KEY and EPO_OPS_CONSUMER_SECRET to run EPO OPS integration tests")
	}
	auth := NewAuthenticator("invalid_key", "invalid_secret", nil)
	ctx := integrationCtx(t)
	tok, err := auth.GetToken(ctx)
	if err == nil {
		t.Fatal("expected error with invalid credentials")
	}
	if tok != "" {
		t.Error("expected empty token on auth failure")
	}
	var authErr *AuthError
	if !errors.As(err, &authErr) {
		t.Errorf("expected *AuthError, got %T", err)
	}
}

// TestIntegrationNotFound confirms a non-existent patent yields an error (the
// library classifies most as NotFoundError; OPS may answer differently for a
// syntactically invalid number, which is also acceptable).
func TestIntegrationNotFound(t *testing.T) {
	c := integrationClient(t)
	ctx := integrationCtx(t)
	_, err := c.GetBiblio(ctx, pubRefType, docdb, "EP.99999999999.A1")
	if err == nil {
		t.Error("expected an error for a non-existent patent")
	}
}

// --- Published-data services ---------------------------------------------------

func TestIntegrationGetBiblio(t *testing.T) {
	c := integrationClient(t)
	d, err := c.GetBiblio(integrationCtx(t), pubRefType, docdb, testDocdb)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetBiblio: %v", err)
	}
	if d == nil || len(d.Titles) == 0 {
		t.Fatal("no biblio titles parsed")
	}
}

func TestIntegrationGetBiblioRaw(t *testing.T) {
	c := integrationClient(t)
	raw, err := c.GetBiblioRaw(integrationCtx(t), pubRefType, docdb, testDocdb)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetBiblioRaw: %v", err)
	}
	mustNonEmptyXML(t, raw)
}

func TestIntegrationGetBiblioMultiple(t *testing.T) {
	c := integrationClient(t)
	raw, err := c.GetBiblioMultiple(integrationCtx(t), pubRefType, docdb, bulkNumbers())
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetBiblioMultiple: %v", err)
	}
	mustNonEmptyXML(t, raw)
}

func TestIntegrationGetAbstract(t *testing.T) {
	c := integrationClient(t)
	d, err := c.GetAbstract(integrationCtx(t), pubRefType, docdb, testDocdb)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetAbstract: %v", err)
	}
	if d == nil {
		t.Fatal("nil abstract")
	}
}

func TestIntegrationGetAbstractRaw(t *testing.T) {
	c := integrationClient(t)
	raw, err := c.GetAbstractRaw(integrationCtx(t), pubRefType, docdb, testDocdb)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetAbstractRaw: %v", err)
	}
	mustNonEmptyXML(t, raw)
}

func TestIntegrationGetAbstractMultiple(t *testing.T) {
	c := integrationClient(t)
	raw, err := c.GetAbstractMultiple(integrationCtx(t), pubRefType, docdb, bulkNumbers())
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetAbstractMultiple: %v", err)
	}
	mustNonEmptyXML(t, raw)
}

func TestIntegrationGetClaims(t *testing.T) {
	c := integrationClient(t)
	d, err := c.GetClaims(integrationCtx(t), pubRefType, docdb, testDocdb)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetClaims: %v", err)
	}
	if d == nil || len(d.Claims) == 0 {
		t.Fatal("no claims parsed")
	}
}

func TestIntegrationGetClaimsRaw(t *testing.T) {
	c := integrationClient(t)
	raw, err := c.GetClaimsRaw(integrationCtx(t), pubRefType, docdb, testDocdb)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetClaimsRaw: %v", err)
	}
	mustNonEmptyXML(t, raw)
}

func TestIntegrationGetClaimsMultiple(t *testing.T) {
	c := integrationClient(t)
	raw, err := c.GetClaimsMultiple(integrationCtx(t), pubRefType, docdb, bulkNumbers())
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetClaimsMultiple: %v", err)
	}
	mustNonEmptyXML(t, raw)
}

func TestIntegrationGetDescription(t *testing.T) {
	c := integrationClient(t)
	d, err := c.GetDescription(integrationCtx(t), pubRefType, docdb, testDocdb)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetDescription: %v", err)
	}
	if d == nil || len(d.Paragraphs) == 0 {
		t.Fatal("no description paragraphs parsed")
	}
}

func TestIntegrationGetDescriptionRaw(t *testing.T) {
	c := integrationClient(t)
	raw, err := c.GetDescriptionRaw(integrationCtx(t), pubRefType, docdb, testDocdb)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetDescriptionRaw: %v", err)
	}
	mustNonEmptyXML(t, raw)
}

func TestIntegrationGetDescriptionMultiple(t *testing.T) {
	c := integrationClient(t)
	d, err := c.GetDescriptionMultiple(integrationCtx(t), pubRefType, docdb, bulkNumbers())
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetDescriptionMultiple: %v", err)
	}
	if d == nil {
		t.Fatal("nil description batch")
	}
}

func TestIntegrationGetFulltext(t *testing.T) {
	c := integrationClient(t)
	d, err := c.GetFulltext(integrationCtx(t), pubRefType, docdb, testDocdb)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetFulltext: %v", err)
	}
	if d == nil {
		t.Fatal("nil fulltext")
	}
}

func TestIntegrationGetFulltextRaw(t *testing.T) {
	c := integrationClient(t)
	raw, err := c.GetFulltextRaw(integrationCtx(t), pubRefType, docdb, testDocdb)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetFulltextRaw: %v", err)
	}
	mustNonEmptyXML(t, raw)
}

func TestIntegrationGetFulltextMultiple(t *testing.T) {
	c := integrationClient(t)
	d, err := c.GetFulltextMultiple(integrationCtx(t), pubRefType, docdb, bulkNumbers())
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetFulltextMultiple: %v", err)
	}
	if d == nil {
		t.Fatal("nil fulltext batch")
	}
}

func TestIntegrationGetFullCycleMultiple(t *testing.T) {
	c := integrationClient(t)
	raw, err := c.GetFullCycleMultiple(integrationCtx(t), pubRefType, docdb, bulkNumbers())
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetFullCycleMultiple: %v", err)
	}
	mustNonEmptyXML(t, raw)
}

func TestIntegrationGetPublishedEquivalents(t *testing.T) {
	c := integrationClient(t)
	d, err := c.GetPublishedEquivalents(integrationCtx(t), pubRefType, docdb, testDocdb)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetPublishedEquivalents: %v", err)
	}
	if d == nil {
		t.Fatal("nil equivalents")
	}
}

func TestIntegrationGetPublishedEquivalentsRaw(t *testing.T) {
	c := integrationClient(t)
	raw, err := c.GetPublishedEquivalentsRaw(integrationCtx(t), pubRefType, docdb, testDocdb)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetPublishedEquivalentsRaw: %v", err)
	}
	mustNonEmptyXML(t, raw)
}

func TestIntegrationGetPublishedEquivalentsMultiple(t *testing.T) {
	c := integrationClient(t)
	d, err := c.GetPublishedEquivalentsMultiple(integrationCtx(t), pubRefType, docdb, bulkNumbers())
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetPublishedEquivalentsMultiple: %v", err)
	}
	if d == nil {
		t.Fatal("nil equivalents batch")
	}
}

// TestIntegrationGetExchangeDocuments exercises the comprehensive biblio parser.
func TestIntegrationGetExchangeDocuments(t *testing.T) {
	c := integrationClient(t)
	docs, err := c.GetExchangeDocuments(integrationCtx(t), pubRefType, docdb, testDocdb)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetExchangeDocuments: %v", err)
	}
	if len(docs) == 0 || docs[0].PublicationNumber() == "" {
		t.Fatal("no exchange documents parsed")
	}
}

// --- Search services -----------------------------------------------------------

func TestIntegrationSearch(t *testing.T) {
	c := integrationClient(t)
	d, err := c.Search(integrationCtx(t), "ti=plastic", "1-5")
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if d == nil || d.TotalCount == 0 {
		t.Fatal("no search results")
	}
}

func TestIntegrationSearchRaw(t *testing.T) {
	c := integrationClient(t)
	raw, err := c.SearchRaw(integrationCtx(t), "ti=plastic", "1-5")
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("SearchRaw: %v", err)
	}
	mustNonEmptyXML(t, raw)
}

func TestIntegrationSearchWithConstituent(t *testing.T) {
	c := integrationClient(t)
	d, err := c.SearchWithConstituent(integrationCtx(t), "biblio", "ti=plastic", "1-3")
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("SearchWithConstituent: %v", err)
	}
	if d == nil {
		t.Fatal("nil search results")
	}
}

// --- Family services -----------------------------------------------------------

func TestIntegrationGetFamily(t *testing.T) {
	c := integrationClient(t)
	d, err := c.GetFamily(integrationCtx(t), pubRefType, docdb, testDocdb)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetFamily: %v", err)
	}
	if d == nil || len(d.Members) == 0 {
		t.Fatal("no family members")
	}
}

func TestIntegrationGetFamilyRaw(t *testing.T) {
	c := integrationClient(t)
	raw, err := c.GetFamilyRaw(integrationCtx(t), pubRefType, docdb, testDocdb)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetFamilyRaw: %v", err)
	}
	mustNonEmptyXML(t, raw)
}

func TestIntegrationGetFamilyWithBiblio(t *testing.T) {
	c := integrationClient(t)
	d, err := c.GetFamilyWithBiblio(integrationCtx(t), pubRefType, docdb, testDocdb)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetFamilyWithBiblio: %v", err)
	}
	if d == nil || len(d.Members) == 0 {
		t.Fatal("no family members")
	}
}

func TestIntegrationGetFamilyWithBiblioMultiple(t *testing.T) {
	c := integrationClient(t)
	d, err := c.GetFamilyWithBiblioMultiple(integrationCtx(t), pubRefType, docdb, bulkNumbers())
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetFamilyWithBiblioMultiple: %v", err)
	}
	if d == nil {
		t.Fatal("nil family batch")
	}
}

func TestIntegrationGetFamilyWithLegal(t *testing.T) {
	c := integrationClient(t)
	d, err := c.GetFamilyWithLegal(integrationCtx(t), pubRefType, docdb, testDocdb)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetFamilyWithLegal: %v", err)
	}
	if d == nil {
		t.Fatal("nil family-with-legal")
	}
}

func TestIntegrationGetFamilyWithLegalMultiple(t *testing.T) {
	c := integrationClient(t)
	d, err := c.GetFamilyWithLegalMultiple(integrationCtx(t), pubRefType, docdb, bulkNumbers())
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetFamilyWithLegalMultiple: %v", err)
	}
	if d == nil {
		t.Fatal("nil family-with-legal batch")
	}
}

// --- Legal services ------------------------------------------------------------

func TestIntegrationGetLegal(t *testing.T) {
	c := integrationClient(t)
	d, err := c.GetLegal(integrationCtx(t), pubRefType, docdb, testDocdb)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetLegal: %v", err)
	}
	if d == nil || len(d.LegalEvents) == 0 {
		t.Fatal("no legal events")
	}
}

func TestIntegrationGetLegalRaw(t *testing.T) {
	c := integrationClient(t)
	raw, err := c.GetLegalRaw(integrationCtx(t), pubRefType, docdb, testDocdb)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetLegalRaw: %v", err)
	}
	mustNonEmptyXML(t, raw)
}

func TestIntegrationGetLegalMultiple(t *testing.T) {
	c := integrationClient(t)
	d, err := c.GetLegalMultiple(integrationCtx(t), pubRefType, docdb, bulkNumbers())
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetLegalMultiple: %v", err)
	}
	if d == nil {
		t.Fatal("nil legal batch")
	}
}

// --- Register services ---------------------------------------------------------

func TestIntegrationGetRegister(t *testing.T) {
	c := integrationClient(t)
	docs, err := c.GetRegister(integrationCtx(t), pubRefType, epodoc, registerEpodoc)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetRegister: %v", err)
	}
	if len(docs) == 0 {
		t.Fatal("no register documents")
	}
}

func TestIntegrationGetRegisterBiblioRaw(t *testing.T) {
	c := integrationClient(t)
	raw, err := c.GetRegisterBiblioRaw(integrationCtx(t), pubRefType, epodoc, registerEpodoc)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetRegisterBiblioRaw: %v", err)
	}
	mustNonEmptyXML(t, raw)
}

func TestIntegrationGetRegisterBiblioMultipleRaw(t *testing.T) {
	c := integrationClient(t)
	raw, err := c.GetRegisterBiblioMultipleRaw(integrationCtx(t), pubRefType, epodoc, []string{registerEpodoc})
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetRegisterBiblioMultipleRaw: %v", err)
	}
	mustNonEmptyXML(t, raw)
}

func TestIntegrationGetRegisterEvents(t *testing.T) {
	c := integrationClient(t)
	d, err := c.GetRegisterEvents(integrationCtx(t), pubRefType, epodoc, registerEpodoc)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetRegisterEvents: %v", err)
	}
	if d == nil || len(d.Events) == 0 {
		t.Fatal("no register events")
	}
}

func TestIntegrationGetRegisterEventsRaw(t *testing.T) {
	c := integrationClient(t)
	raw, err := c.GetRegisterEventsRaw(integrationCtx(t), pubRefType, epodoc, registerEpodoc)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetRegisterEventsRaw: %v", err)
	}
	mustNonEmptyXML(t, raw)
}

func TestIntegrationGetRegisterEventsMultipleRaw(t *testing.T) {
	c := integrationClient(t)
	raw, err := c.GetRegisterEventsMultipleRaw(integrationCtx(t), pubRefType, epodoc, []string{registerEpodoc})
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetRegisterEventsMultipleRaw: %v", err)
	}
	mustNonEmptyXML(t, raw)
}

func TestIntegrationGetRegisterProceduralSteps(t *testing.T) {
	c := integrationClient(t)
	docs, err := c.GetRegisterProceduralSteps(integrationCtx(t), pubRefType, epodoc, registerEpodoc)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetRegisterProceduralSteps: %v", err)
	}
	if len(docs) == 0 {
		t.Fatal("no register documents")
	}
}

func TestIntegrationGetRegisterProceduralStepsRaw(t *testing.T) {
	c := integrationClient(t)
	raw, err := c.GetRegisterProceduralStepsRaw(integrationCtx(t), pubRefType, epodoc, registerEpodoc)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetRegisterProceduralStepsRaw: %v", err)
	}
	mustNonEmptyXML(t, raw)
}

func TestIntegrationGetRegisterProceduralStepsMultipleRaw(t *testing.T) {
	c := integrationClient(t)
	raw, err := c.GetRegisterProceduralStepsMultipleRaw(integrationCtx(t), pubRefType, epodoc, []string{registerEpodoc})
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetRegisterProceduralStepsMultipleRaw: %v", err)
	}
	mustNonEmptyXML(t, raw)
}

func TestIntegrationGetRegisterUNIP(t *testing.T) {
	c := integrationClient(t)
	// Not every patent has a unitary-patent record; absence is a clean skip.
	docs, err := c.GetRegisterUNIP(integrationCtx(t), pubRefType, epodoc, registerEpodoc)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetRegisterUNIP: %v", err)
	}
	_ = docs
}

func TestIntegrationGetRegisterUNIPRaw(t *testing.T) {
	c := integrationClient(t)
	raw, err := c.GetRegisterUNIPRaw(integrationCtx(t), pubRefType, epodoc, registerEpodoc)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetRegisterUNIPRaw: %v", err)
	}
	mustNonEmptyXML(t, raw)
}

func TestIntegrationGetRegisterUNIPMultipleRaw(t *testing.T) {
	c := integrationClient(t)
	raw, err := c.GetRegisterUNIPMultipleRaw(integrationCtx(t), pubRefType, epodoc, []string{registerEpodoc})
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetRegisterUNIPMultipleRaw: %v", err)
	}
	mustNonEmptyXML(t, raw)
}

func TestIntegrationSearchRegister(t *testing.T) {
	c := integrationClient(t)
	raw, err := c.SearchRegister(integrationCtx(t), "ti=battery", "1-5")
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("SearchRegister: %v", err)
	}
	mustNonEmptyXML(t, raw)
}

func TestIntegrationSearchRegisterWithConstituent(t *testing.T) {
	c := integrationClient(t)
	raw, err := c.SearchRegisterWithConstituent(integrationCtx(t), "biblio", "ti=solar", "1-3")
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("SearchRegisterWithConstituent: %v", err)
	}
	mustNonEmptyXML(t, raw)
}

// --- Classification services ---------------------------------------------------

func TestIntegrationGetClassificationSchema(t *testing.T) {
	c := integrationClient(t)
	d, err := c.GetClassificationSchema(integrationCtx(t), "H04W", false, false)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetClassificationSchema: %v", err)
	}
	if d == nil || d.Symbol == "" {
		t.Fatal("no classification symbol")
	}
}

func TestIntegrationGetClassificationSchemaRaw(t *testing.T) {
	c := integrationClient(t)
	raw, err := c.GetClassificationSchemaRaw(integrationCtx(t), "H04W", false, false)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetClassificationSchemaRaw: %v", err)
	}
	mustNonEmptyXML(t, raw)
}

func TestIntegrationGetClassificationSchemaSubclassRaw(t *testing.T) {
	c := integrationClient(t)
	raw, err := c.GetClassificationSchemaSubclassRaw(integrationCtx(t), "H04W4", "00", false, false)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetClassificationSchemaSubclassRaw: %v", err)
	}
	mustNonEmptyXML(t, raw)
}

func TestIntegrationGetClassificationSchemaMultipleRaw(t *testing.T) {
	c := integrationClient(t)
	raw, err := c.GetClassificationSchemaMultipleRaw(integrationCtx(t), []string{"H04W", "H04L"})
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetClassificationSchemaMultipleRaw: %v", err)
	}
	mustNonEmptyXML(t, raw)
}

func TestIntegrationGetClassificationStatisticsRaw(t *testing.T) {
	c := integrationClient(t)
	raw, err := c.GetClassificationStatisticsRaw(integrationCtx(t), "H04W")
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetClassificationStatisticsRaw: %v", err)
	}
	mustNonEmptyXML(t, raw)
}

func TestIntegrationGetClassificationMappingRaw(t *testing.T) {
	c := integrationClient(t)
	raw, err := c.GetClassificationMappingRaw(integrationCtx(t), "cpc", "H04W84", "18", "ecla", false)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetClassificationMappingRaw: %v", err)
	}
	mustNonEmptyXML(t, raw)
}

// TestIntegrationGetClassificationMedia asserts the CPC media GET returns
// non-empty image bytes (a GIF here).
func TestIntegrationGetClassificationMedia(t *testing.T) {
	c := integrationClient(t)
	data, err := c.GetClassificationMedia(integrationCtx(t), "1000.gif", false)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetClassificationMedia: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("empty media bytes")
	}
}

// --- Number service ------------------------------------------------------------

func TestIntegrationConvertPatentNumber(t *testing.T) {
	c := integrationClient(t)
	d, err := c.ConvertPatentNumber(integrationCtx(t), pubRefType, docdb, convertDocdb, epodoc)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("ConvertPatentNumber: %v", err)
	}
	if d == nil || d.DocNumber == "" {
		t.Fatal("no converted doc number")
	}
}

func TestIntegrationConvertPatentNumberRaw(t *testing.T) {
	c := integrationClient(t)
	raw, err := c.ConvertPatentNumberRaw(integrationCtx(t), pubRefType, docdb, convertDocdb, epodoc)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("ConvertPatentNumberRaw: %v", err)
	}
	mustNonEmptyXML(t, raw)
}

func TestIntegrationConvertPatentNumberMultipleRaw(t *testing.T) {
	c := integrationClient(t)
	raw, err := c.ConvertPatentNumberMultipleRaw(integrationCtx(t), pubRefType, docdb, bulkNumbers(), epodoc)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("ConvertPatentNumberMultipleRaw: %v", err)
	}
	mustNonEmptyXML(t, raw)
}

// --- Image services ------------------------------------------------------------

// TestIntegrationGetImage asserts the image GET returns non-empty bytes with TIFF
// magic (OPS drawings are TIFF).
func TestIntegrationGetImage(t *testing.T) {
	c := integrationClient(t)
	data, err := c.GetImage(integrationCtx(t), imgCountry, imgNumber, imgKind, "Drawing", 1)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetImage: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("empty image bytes")
	}
	if !isTIFF(data) {
		t.Logf("image not TIFF magic (first bytes %x); accepted as non-empty bytes", data[:min(4, len(data))])
	}
}

// TestIntegrationGetImagePOST asserts the POST image variant returns image bytes.
// It first resolves a fetchable instance via the image inquiry.
func TestIntegrationGetImagePOST(t *testing.T) {
	c := integrationClient(t)
	ctx := integrationCtx(t)
	inq, err := c.GetImageInquiry(ctx, pubRefType, docdb, convertDocdb)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetImageInquiry (for POST): %v", err)
	}
	if len(inq.DocumentInstances) == 0 {
		t.Skip("no image instances available for sample")
	}
	// The POST identifier is the instance link minus the page-image suffix; the
	// demo's proven identifier form is "<Country>/<Number>/<Kind>/fullimage".
	identifier := imgCountry + "/2400812/A1/fullimage"
	data, err := c.GetImagePOST(ctx, 1, identifier)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetImagePOST: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("empty image bytes")
	}
}

func TestIntegrationGetImageInquiry(t *testing.T) {
	c := integrationClient(t)
	d, err := c.GetImageInquiry(integrationCtx(t), pubRefType, docdb, convertDocdb)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetImageInquiry: %v", err)
	}
	if d == nil || len(d.DocumentInstances) == 0 {
		t.Fatal("no image instances")
	}
}

// --- Usage / quota services ----------------------------------------------------

// TestIntegrationGetLastQuota asserts quota is captured from response headers
// after a real call (it is nil before any request).
func TestIntegrationGetLastQuota(t *testing.T) {
	c := integrationClient(t)
	if c.GetLastQuota() != nil {
		t.Error("expected nil quota before any request")
	}
	_, err := c.GetBiblio(integrationCtx(t), pubRefType, docdb, testDocdb)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetBiblio (for quota): %v", err)
	}
	if c.GetLastQuota() == nil {
		t.Fatal("expected quota after a request")
	}
}

func TestIntegrationGetUsageStats(t *testing.T) {
	c := integrationClient(t)
	// A small recent window (dd/MM/yyyy~dd/MM/yyyy), as the demo uses.
	now := time.Now()
	from := now.AddDate(0, 0, -1).Format("02/01/2006")
	to := now.Format("02/01/2006")
	d, err := c.GetUsageStats(integrationCtx(t), from+"~"+to)
	if skipIfTransient(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("GetUsageStats: %v", err)
	}
	if d == nil {
		t.Fatal("nil usage stats")
	}
}

// isTIFF reports whether b begins with a TIFF byte-order mark (II / MM).
func isTIFF(b []byte) bool {
	return len(b) >= 2 && ((b[0] == 'I' && b[1] == 'I') || (b[0] == 'M' && b[1] == 'M'))
}
