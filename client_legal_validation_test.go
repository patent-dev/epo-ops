package epo_ops

import (
	"context"
	"errors"
	"testing"
)

// newValidationOnlyClient returns a client suitable for exercising input
// validation paths; validation failures return before any HTTP request.
func newValidationOnlyClient(t *testing.T) *Client {
	t.Helper()
	client, err := NewClient(&Config{ConsumerKey: "test", ConsumerSecret: "test"})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	return client
}

// TestBulkMethods_RejectEmbeddedNewlines verifies that the newline-separated
// bulk POST bodies cannot be used to smuggle extra numbers past validation,
// including through the permissive "original" format.
func TestBulkMethods_RejectEmbeddedNewlines(t *testing.T) {
	client := newValidationOnlyClient(t)
	ctx := context.Background()
	smuggled := []string{"EP1000000\nEP1000001"}

	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "GetLegalMultiple original format",
			call: func() error {
				_, err := client.GetLegalMultiple(ctx, RefTypePublication, FormatOriginal, smuggled)
				return err
			},
		},
		{
			name: "GetRegisterBiblioMultipleRaw",
			call: func() error {
				_, err := client.GetRegisterBiblioMultipleRaw(ctx, RefTypePublication, FormatEPODOC, smuggled)
				return err
			},
		},
		{
			name: "GetRegisterEventsMultipleRaw",
			call: func() error {
				_, err := client.GetRegisterEventsMultipleRaw(ctx, RefTypePublication, FormatEPODOC, smuggled)
				return err
			},
		},
		{
			name: "GetRegisterProceduralStepsMultipleRaw",
			call: func() error {
				_, err := client.GetRegisterProceduralStepsMultipleRaw(ctx, RefTypePublication, FormatEPODOC, smuggled)
				return err
			},
		},
		{
			name: "GetRegisterUNIPMultipleRaw",
			call: func() error {
				_, err := client.GetRegisterUNIPMultipleRaw(ctx, RefTypePublication, FormatEPODOC, smuggled)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			if err == nil {
				t.Fatal("expected validation error for embedded newline, got nil")
			}
			var valErr *ValidationError
			if !errors.As(err, &valErr) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}

// TestBulkMethods_ListBoundsAreValidationErrors verifies the shared list-level
// checks (1-100 entries) on the register bulk methods, including the two that
// previously returned ConfigError for these cases.
func TestBulkMethods_ListBoundsAreValidationErrors(t *testing.T) {
	client := newValidationOnlyClient(t)
	ctx := context.Background()

	oversized := make([]string, 101)
	for i := range oversized {
		oversized[i] = "EP1000000"
	}

	tests := []struct {
		name    string
		numbers []string
	}{
		{name: "empty list", numbers: nil},
		{name: "oversized list", numbers: oversized},
	}

	calls := []struct {
		name string
		call func(numbers []string) error
	}{
		{
			name: "GetRegisterProceduralStepsMultipleRaw",
			call: func(numbers []string) error {
				_, err := client.GetRegisterProceduralStepsMultipleRaw(ctx, RefTypePublication, FormatEPODOC, numbers)
				return err
			},
		},
		{
			name: "GetRegisterUNIPMultipleRaw",
			call: func(numbers []string) error {
				_, err := client.GetRegisterUNIPMultipleRaw(ctx, RefTypePublication, FormatEPODOC, numbers)
				return err
			},
		},
	}

	for _, c := range calls {
		for _, tt := range tests {
			t.Run(c.name+" "+tt.name, func(t *testing.T) {
				err := c.call(tt.numbers)
				if err == nil {
					t.Fatal("expected validation error, got nil")
				}
				var valErr *ValidationError
				if !errors.As(err, &valErr) {
					t.Errorf("expected ValidationError, got %T: %v", err, err)
				}
			})
		}
	}
}

// TestRegisterServices_RejectPriorityRefType verifies that the procedural-steps
// and UNIP services reject refType "priority" with a clear ValidationError
// instead of silently remapping it to "application".
func TestRegisterServices_RejectPriorityRefType(t *testing.T) {
	client := newValidationOnlyClient(t)
	ctx := context.Background()

	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "GetRegisterProceduralStepsRaw",
			call: func() error {
				_, err := client.GetRegisterProceduralStepsRaw(ctx, RefTypePriority, FormatEPODOC, "EP1000000")
				return err
			},
		},
		{
			name: "GetRegisterProceduralStepsMultipleRaw",
			call: func() error {
				_, err := client.GetRegisterProceduralStepsMultipleRaw(ctx, RefTypePriority, FormatEPODOC, []string{"EP1000000"})
				return err
			},
		},
		{
			name: "GetRegisterUNIPRaw",
			call: func() error {
				_, err := client.GetRegisterUNIPRaw(ctx, RefTypePriority, FormatEPODOC, "EP1000000")
				return err
			},
		},
		{
			name: "GetRegisterUNIPMultipleRaw",
			call: func() error {
				_, err := client.GetRegisterUNIPMultipleRaw(ctx, RefTypePriority, FormatEPODOC, []string{"EP1000000"})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			if err == nil {
				t.Fatal("expected validation error for refType priority, got nil")
			}
			var valErr *ValidationError
			if !errors.As(err, &valErr) {
				t.Fatalf("expected ValidationError, got %T: %v", err, err)
			}
			if valErr.Field != "refType" {
				t.Errorf("expected error field 'refType', got %q", valErr.Field)
			}
		})
	}
}
