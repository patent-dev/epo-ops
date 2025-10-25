package epo_ops

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
)

// Helper function to create a slice of patent numbers
func makePatentNumbers(n int) []string {
	numbers := make([]string, n)
	for i := 0; i < n; i++ {
		numbers[i] = fmt.Sprintf("EP.%07d.B1", i+1000000)
	}
	return numbers
}

func TestSplitIntoBatches(t *testing.T) {
	tests := []struct {
		name      string
		items     []string
		batchSize int
		wantCount int
		want      []int // expected batch sizes
	}{
		{
			name:      "Empty slice",
			items:     []string{},
			batchSize: 100,
			wantCount: 0,
			want:      []int{},
		},
		{
			name:      "Less than batch size",
			items:     makePatentNumbers(50),
			batchSize: 100,
			wantCount: 1,
			want:      []int{50},
		},
		{
			name:      "Exact multiple",
			items:     makePatentNumbers(200),
			batchSize: 100,
			wantCount: 2,
			want:      []int{100, 100},
		},
		{
			name:      "With remainder",
			items:     makePatentNumbers(250),
			batchSize: 100,
			wantCount: 3,
			want:      []int{100, 100, 50},
		},
		{
			name:      "150 items (2 batches)",
			items:     makePatentNumbers(150),
			batchSize: 100,
			wantCount: 2,
			want:      []int{100, 50},
		},
		{
			name:      "Single item",
			items:     makePatentNumbers(1),
			batchSize: 100,
			wantCount: 1,
			want:      []int{1},
		},
		{
			name:      "Exact batch size",
			items:     makePatentNumbers(100),
			batchSize: 100,
			wantCount: 1,
			want:      []int{100},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batches := splitIntoBatches(tt.items, tt.batchSize)

			if len(batches) != tt.wantCount {
				t.Errorf("splitIntoBatches() got %d batches, want %d", len(batches), tt.wantCount)
			}

			for i, wantSize := range tt.want {
				if len(batches[i]) != wantSize {
					t.Errorf("batch %d: got size %d, want %d", i, len(batches[i]), wantSize)
				}
			}

			// Verify all items are present and in order
			if len(tt.items) > 0 {
				flatBatches := []string{}
				for _, batch := range batches {
					flatBatches = append(flatBatches, batch...)
				}
				if len(flatBatches) != len(tt.items) {
					t.Errorf("Total items after batching: got %d, want %d", len(flatBatches), len(tt.items))
				}
			}
		})
	}
}

func TestGetBibliosBulk(t *testing.T) {
	// Track number of API calls
	callCount := atomic.Int32{}

	// Mock auth server
	authServer := newMockAuthServer(t)
	defer authServer.Close()

	// Mock OPS server
	opsServer := newMockOPSServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		// Return sample XML
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<ops:world-patent-data xmlns:ops="http://ops.epo.org">
  <ops:biblio-search>Batch result</ops:biblio-search>
</ops:world-patent-data>`))
	})
	defer opsServer.Close()

	config := DefaultConfig()
	config.ConsumerKey = "test"
	config.ConsumerSecret = "test"
	config.BaseURL = opsServer.URL
	config.AuthURL = authServer.URL + "/auth/accesstoken"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	tests := []struct {
		name          string
		numbers       []string
		wantBatches   int
		wantCallCount int32
	}{
		{
			name:          "50 numbers (1 batch)",
			numbers:       makePatentNumbers(50),
			wantBatches:   1,
			wantCallCount: 1,
		},
		{
			name:          "150 numbers (2 batches)",
			numbers:       makePatentNumbers(150),
			wantBatches:   2,
			wantCallCount: 2,
		},
		{
			name:          "250 numbers (3 batches)",
			numbers:       makePatentNumbers(250),
			wantBatches:   3,
			wantCallCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount.Store(0) // Reset counter

			ctx := context.Background()
			results, err := client.GetBibliosBulk(ctx, RefTypePublication, FormatDocDB, tt.numbers, nil)
			if err != nil {
				t.Fatalf("GetBibliosBulk() error = %v", err)
			}

			if len(results) != tt.wantBatches {
				t.Errorf("GetBibliosBulk() got %d results, want %d", len(results), tt.wantBatches)
			}

			if callCount.Load() != tt.wantCallCount {
				t.Errorf("GetBibliosBulk() made %d API calls, want %d", callCount.Load(), tt.wantCallCount)
			}

			// Verify each result contains XML
			for i, result := range results {
				if result == "" {
					t.Errorf("Result %d is empty", i)
				}
				if len(result) < 10 {
					t.Errorf("Result %d too short: %d bytes", i, len(result))
				}
			}
		})
	}
}

func TestGetBibliosBulk_WithProgress(t *testing.T) {
	authServer := newMockAuthServer(t)
	defer authServer.Close()

	opsServer := newMockOPSServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<ops:world-patent-data xmlns:ops="http://ops.epo.org"></ops:world-patent-data>`))
	})
	defer opsServer.Close()

	config := DefaultConfig()
	config.ConsumerKey = "test"
	config.ConsumerSecret = "test"
	config.BaseURL = opsServer.URL
	config.AuthURL = authServer.URL + "/auth/accesstoken"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Track progress callbacks
	progressCalls := []string{}

	opts := &BulkOptions{
		OnProgress: func(current, total int) {
			progressCalls = append(progressCalls, fmt.Sprintf("%d/%d", current, total))
		},
	}

	numbers := makePatentNumbers(250) // 3 batches
	ctx := context.Background()

	_, err = client.GetBibliosBulk(ctx, RefTypePublication, FormatDocDB, numbers, opts)
	if err != nil {
		t.Fatalf("GetBibliosBulk() error = %v", err)
	}

	// Verify progress callbacks
	wantProgress := []string{"1/3", "2/3", "3/3"}
	if len(progressCalls) != len(wantProgress) {
		t.Errorf("Got %d progress callbacks, want %d", len(progressCalls), len(wantProgress))
	}

	for i, want := range wantProgress {
		if i < len(progressCalls) && progressCalls[i] != want {
			t.Errorf("Progress callback %d: got %s, want %s", i, progressCalls[i], want)
		}
	}
}

func TestGetClaimsBulk(t *testing.T) {
	callCount := atomic.Int32{}

	authServer := newMockAuthServer(t)
	defer authServer.Close()

	opsServer := newMockOPSServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<ops:world-patent-data xmlns:ops="http://ops.epo.org"></ops:world-patent-data>`))
	})
	defer opsServer.Close()

	config := DefaultConfig()
	config.ConsumerKey = "test"
	config.ConsumerSecret = "test"
	config.BaseURL = opsServer.URL
	config.AuthURL = authServer.URL + "/auth/accesstoken"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	callCount.Store(0)
	numbers := makePatentNumbers(150)
	ctx := context.Background()

	results, err := client.GetClaimsBulk(ctx, RefTypePublication, FormatDocDB, numbers, nil)
	if err != nil {
		t.Fatalf("GetClaimsBulk() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("GetClaimsBulk() got %d results, want 2", len(results))
	}

	if callCount.Load() != 2 {
		t.Errorf("GetClaimsBulk() made %d API calls, want 2", callCount.Load())
	}
}

func TestGetDescriptionsBulk(t *testing.T) {
	authServer := newMockAuthServer(t)
	defer authServer.Close()

	opsServer := newMockOPSServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<ops:world-patent-data xmlns:ops="http://ops.epo.org"></ops:world-patent-data>`))
	})
	defer opsServer.Close()

	config := DefaultConfig()
	config.ConsumerKey = "test"
	config.ConsumerSecret = "test"
	config.BaseURL = opsServer.URL
	config.AuthURL = authServer.URL + "/auth/accesstoken"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	numbers := makePatentNumbers(100)
	ctx := context.Background()

	results, err := client.GetDescriptionsBulk(ctx, RefTypePublication, FormatDocDB, numbers, nil)
	if err != nil {
		t.Fatalf("GetDescriptionsBulk() error = %v", err)
	}

	if len(results) != 1 {
		t.Errorf("GetDescriptionsBulk() got %d results, want 1", len(results))
	}
}

func TestGetAbstractsBulk(t *testing.T) {
	authServer := newMockAuthServer(t)
	defer authServer.Close()

	opsServer := newMockOPSServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<ops:world-patent-data xmlns:ops="http://ops.epo.org"></ops:world-patent-data>`))
	})
	defer opsServer.Close()

	config := DefaultConfig()
	config.ConsumerKey = "test"
	config.ConsumerSecret = "test"
	config.BaseURL = opsServer.URL
	config.AuthURL = authServer.URL + "/auth/accesstoken"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	numbers := makePatentNumbers(200)
	ctx := context.Background()

	results, err := client.GetAbstractsBulk(ctx, RefTypePublication, FormatDocDB, numbers, nil)
	if err != nil {
		t.Fatalf("GetAbstractsBulk() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("GetAbstractsBulk() got %d results, want 2", len(results))
	}
}

func TestGetFulltextsBulk(t *testing.T) {
	authServer := newMockAuthServer(t)
	defer authServer.Close()

	opsServer := newMockOPSServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<ops:world-patent-data xmlns:ops="http://ops.epo.org"></ops:world-patent-data>`))
	})
	defer opsServer.Close()

	config := DefaultConfig()
	config.ConsumerKey = "test"
	config.ConsumerSecret = "test"
	config.BaseURL = opsServer.URL
	config.AuthURL = authServer.URL + "/auth/accesstoken"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	numbers := makePatentNumbers(50)
	ctx := context.Background()

	results, err := client.GetFulltextsBulk(ctx, RefTypePublication, FormatDocDB, numbers, nil)
	if err != nil {
		t.Fatalf("GetFulltextsBulk() error = %v", err)
	}

	if len(results) != 1 {
		t.Errorf("GetFulltextsBulk() got %d results, want 1", len(results))
	}
}

func TestBulkOperations_NilOptions(t *testing.T) {
	authServer := newMockAuthServer(t)
	defer authServer.Close()

	opsServer := newMockOPSServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<ops:world-patent-data xmlns:ops="http://ops.epo.org"></ops:world-patent-data>`))
	})
	defer opsServer.Close()

	config := DefaultConfig()
	config.ConsumerKey = "test"
	config.ConsumerSecret = "test"
	config.BaseURL = opsServer.URL
	config.AuthURL = authServer.URL + "/auth/accesstoken"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	numbers := makePatentNumbers(50)
	ctx := context.Background()

	// Test with nil options (should use defaults)
	_, err = client.GetBibliosBulk(ctx, RefTypePublication, FormatDocDB, numbers, nil)
	if err != nil {
		t.Errorf("GetBibliosBulk() with nil options error = %v", err)
	}

	_, err = client.GetClaimsBulk(ctx, RefTypePublication, FormatDocDB, numbers, nil)
	if err != nil {
		t.Errorf("GetClaimsBulk() with nil options error = %v", err)
	}

	_, err = client.GetDescriptionsBulk(ctx, RefTypePublication, FormatDocDB, numbers, nil)
	if err != nil {
		t.Errorf("GetDescriptionsBulk() with nil options error = %v", err)
	}

	_, err = client.GetAbstractsBulk(ctx, RefTypePublication, FormatDocDB, numbers, nil)
	if err != nil {
		t.Errorf("GetAbstractsBulk() with nil options error = %v", err)
	}

	_, err = client.GetFulltextsBulk(ctx, RefTypePublication, FormatDocDB, numbers, nil)
	if err != nil {
		t.Errorf("GetFulltextsBulk() with nil options error = %v", err)
	}
}
