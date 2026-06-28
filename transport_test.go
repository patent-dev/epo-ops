package epo_ops

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type countingTransport struct{ calls int }

func (c *countingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	c.calls++
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"access_token":"t","expires_in":"3600"}`)),
		Header:     make(http.Header),
	}, nil
}

type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestNewClient_TransportGovernsBothPaths(t *testing.T) {
	var tokenHits, apiHits int
	rt := rtFunc(func(r *http.Request) (*http.Response, error) {
		body := `<x/>`
		if strings.Contains(r.URL.Path, "accesstoken") {
			tokenHits++
			body = `{"access_token":"t","expires_in":"3600"}`
		} else {
			apiHits++
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})

	c, err := NewClient(&Config{ConsumerKey: "k", ConsumerSecret: "s", Transport: rt})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	// The biblio body is junk so the call errors, but the API request must still
	// have flowed through the injected transport - that is what we assert.
	_, _ = c.GetBiblio(context.Background(), "publication", "docdb", "EP.1000000.B1")
	if tokenHits == 0 {
		t.Error("token request did not use the injected transport")
	}
	if apiHits == 0 {
		t.Error("API request did not use the injected transport")
	}
}

func TestNewClient_UsesConfigTransport(t *testing.T) {
	rt := &countingTransport{}
	c, err := NewClient(&Config{ConsumerKey: "k", ConsumerSecret: "s", Transport: rt})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	// The authenticator's token request must flow through the injected transport,
	// confirming it governs outbound EPO traffic (API requests share the same base).
	if _, err := c.authenticator.GetToken(context.Background()); err != nil {
		t.Fatalf("GetToken: %v", err)
	}
	if rt.calls == 0 {
		t.Error("injected Config.Transport was not used for the token request")
	}
}
