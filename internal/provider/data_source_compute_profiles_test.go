// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/evroc-oss/evroc-go-sdk/compute"
)

func TestDataSourceComputeProfilesRead(t *testing.T) {
	res := dataSourceComputeProfiles()
	d := res.TestResourceData()

	diags := dataSourceComputeProfilesRead(context.Background(), d, nil)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diagnosticsToString(diags))
	}

	// Verify ID is set
	if d.Id() != "compute-profiles" {
		t.Errorf("expected ID %q, got %q", "compute-profiles", d.Id())
	}

	// Verify profiles list is populated
	profiles := d.Get("profiles").([]interface{})
	if len(profiles) == 0 {
		t.Error("expected at least one profile in the list")
	}

	// Verify count matches SDK
	if len(profiles) != len(compute.ValidVMSizes) {
		t.Errorf("expected %d profiles, got %d", len(compute.ValidVMSizes), len(profiles))
	}

	// Verify series list is populated
	series := d.Get("series").([]interface{})
	if len(series) == 0 {
		t.Error("expected at least one series")
	}
	if len(series) != len(compute.AllVMSizeSeries) {
		t.Errorf("expected %d series, got %d", len(compute.AllVMSizeSeries), len(series))
	}

	// Verify series structure
	for i, s := range series {
		seriesMap := s.(map[string]interface{})
		if seriesMap["name"] == nil || seriesMap["name"].(string) == "" {
			t.Errorf("series[%d] has empty name", i)
		}
		if seriesMap["description"] == nil || seriesMap["description"].(string) == "" {
			t.Errorf("series[%d] has empty description", i)
		}
		sizes := seriesMap["sizes"].([]interface{})
		if len(sizes) == 0 {
			t.Errorf("series[%d] has no sizes", i)
		}
	}

	// Verify individual named fields are set for known profiles
	knownMappings := map[string]string{
		"a1a_s": "a1a.s",
		"a1a_m": "a1a.m",
		"a1a_l": "a1a.l",
		"c1a_s": "c1a.s",
		"m1a_s": "m1a.s",
	}

	for field, expected := range knownMappings {
		val := d.Get(field).(string)
		if val != expected {
			t.Errorf("field %q: expected %q, got %q", field, expected, val)
		}
	}
}

func TestComputeProfileFieldNameMapping(t *testing.T) {
	// Verify the string replacement logic maps profile names to valid Terraform field names
	tests := []struct {
		profileName string
		fieldName   string
	}{
		{"a1a.s", "a1a_s"},
		{"a1a.m", "a1a_m"},
		{"c1a.l", "c1a_l"},
		{"m1a.xl", "m1a_xl"},
	}

	for _, tt := range tests {
		t.Run(tt.profileName, func(t *testing.T) {
			fieldName := strings.ReplaceAll(tt.profileName, ".", "_")
			fieldName = strings.ReplaceAll(fieldName, "-", "_")
			if fieldName != tt.fieldName {
				t.Errorf("expected field name %q, got %q", tt.fieldName, fieldName)
			}
		})
	}
}
