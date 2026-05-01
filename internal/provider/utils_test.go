// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"errors"
	"fmt"
	"testing"

	evroc "github.com/evroc-oss/evroc-go-sdk"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "ErrNotFound",
			err:      evroc.ErrNotFound,
			expected: true,
		},
		{
			name:     "wrapped ErrNotFound",
			err:      fmt.Errorf("resource lookup failed: %w", evroc.ErrNotFound),
			expected: true,
		},
		{
			name:     "generic error",
			err:      errors.New("something went wrong"),
			expected: false,
		},
		{
			name:     "ErrConflict",
			err:      evroc.ErrConflict,
			expected: false,
		},
		{
			name:     "ErrForbidden",
			err:      evroc.ErrForbidden,
			expected: false,
		},
		{
			name:     "ErrBadRequest",
			err:      evroc.ErrBadRequest,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isNotFoundError(tt.err)
			if got != tt.expected {
				t.Errorf("isNotFoundError(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestSetDiag(t *testing.T) {
	// Create a minimal resource schema for testing
	res := &schema.Resource{
		Schema: map[string]*schema.Schema{
			"test_field": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"test_int": {
				Type:     schema.TypeInt,
				Optional: true,
			},
		},
	}

	t.Run("set string value", func(t *testing.T) {
		d := res.TestResourceData()
		var diags diag.Diagnostics
		diags = setDiag(d, "test_field", "hello", diags)
		if diags.HasError() {
			t.Errorf("unexpected error: %v", diags)
		}
		if v := d.Get("test_field").(string); v != "hello" {
			t.Errorf("expected %q, got %q", "hello", v)
		}
	})

	t.Run("set int value", func(t *testing.T) {
		d := res.TestResourceData()
		var diags diag.Diagnostics
		diags = setDiag(d, "test_int", 42, diags)
		if diags.HasError() {
			t.Errorf("unexpected error: %v", diags)
		}
		if v := d.Get("test_int").(int); v != 42 {
			t.Errorf("expected %d, got %d", 42, v)
		}
	})

	t.Run("preserves existing diags", func(t *testing.T) {
		d := res.TestResourceData()
		diags := diag.Diagnostics{
			{Severity: diag.Warning, Summary: "existing warning"},
		}
		diags = setDiag(d, "test_field", "value", diags)
		if len(diags) != 1 {
			t.Errorf("expected 1 diagnostic, got %d", len(diags))
		}
	})
}

func TestFlattenLabels(t *testing.T) {
	t.Run("nil labels returns empty map", func(t *testing.T) {
		result := flattenLabels[map[string]string](nil)
		if len(result) != 0 {
			t.Errorf("expected empty map, got %v", result)
		}
	})

	t.Run("empty labels returns empty map", func(t *testing.T) {
		labels := map[string]string{}
		result := flattenLabels(&labels)
		if len(result) != 0 {
			t.Errorf("expected empty map, got %v", result)
		}
	})

	t.Run("populated labels are flattened", func(t *testing.T) {
		labels := map[string]string{"env": "prod", "team": "infra"}
		result := flattenLabels(&labels)
		if len(result) != 2 {
			t.Errorf("expected 2 entries, got %d", len(result))
		}
		if result["env"] != "prod" {
			t.Errorf("expected env=prod, got %v", result["env"])
		}
		if result["team"] != "infra" {
			t.Errorf("expected team=infra, got %v", result["team"])
		}
	})
}
