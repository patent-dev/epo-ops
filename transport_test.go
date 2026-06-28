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
