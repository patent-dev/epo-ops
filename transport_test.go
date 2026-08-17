package epo_ops

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
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

func TestNewClient_SetsUserAgent(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		want      string
	}{
		{name: "default", userAgent: "", want: DefaultUserAgent},
		{name: "override", userAgent: "custom-agent/9.9", want: "custom-agent/9.9"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tokenUA, dataUA string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "accesstoken") {
					tokenUA = r.Header.Get("User-Agent")
					w.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(w, `{"access_token":"t","expires_in":"3600","token_type":"Bearer"}`)
					return
				}
				dataUA = r.Header.Get("User-Agent")
				_, _ = io.WriteString(w, `<x/>`)
			}))
			defer srv.Close()

			c, err := NewClient(&Config{
				ConsumerKey:    "k",
				ConsumerSecret: "s",
				BaseURL:        srv.URL,
				AuthURL:        srv.URL + "/auth/accesstoken",
				UserAgent:      tt.userAgent,
			})
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}

			// The junk body makes the call error, but the request still reaches the
			// server, which is where we capture the User-Agent.
			_, _ = c.GetBiblio(context.Background(), "publication", "docdb", "EP.1000000.B1")

			if dataUA != tt.want {
				t.Errorf("data request User-Agent = %q, want %q", dataUA, tt.want)
			}
			if tokenUA != tt.want {
				t.Errorf("token request User-Agent = %q, want %q", tokenUA, tt.want)
			}
		})
	}
}

func TestDefaultUserAgent_ContainsSlugAndVersion(t *testing.T) {
	if !strings.Contains(DefaultUserAgent, "epo-ops-go") {
		t.Errorf("DefaultUserAgent %q missing slug epo-ops-go", DefaultUserAgent)
	}
	if !strings.Contains(DefaultUserAgent, Version) {
		t.Errorf("DefaultUserAgent %q missing version %q", DefaultUserAgent, Version)
	}
	if Version != "1.7.0" {
		t.Errorf("Version = %q, want 1.7.0", Version)
	}
}
