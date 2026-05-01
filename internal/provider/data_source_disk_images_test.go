// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/evroc-oss/evroc-go-sdk/compute"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

func TestDataSourceDiskImagesRead(t *testing.T) {
	res := dataSourceDiskImages()
	d := res.TestResourceData()

	diags := dataSourceDiskImagesRead(context.Background(), d, nil)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diagnosticsToString(diags))
	}

	// Verify ID is set
	if d.Id() != "disk-images" {
		t.Errorf("expected ID %q, got %q", "disk-images", d.Id())
	}

	// Verify images list is populated
	images := d.Get("images").([]interface{})
	if len(images) == 0 {
		t.Error("expected at least one image in the list")
	}

	// Verify all ValidDiskImages are in the list
	if len(images) != len(compute.ValidDiskImages) {
		t.Errorf("expected %d images, got %d", len(compute.ValidDiskImages), len(images))
	}

	// Verify individual named attributes are set for known images
	knownMappings := map[string]string{
		"ubuntu_minimal_24_04_1": "ubuntu-minimal.24-04.1",
		"ubuntu_24_04_1":         "ubuntu.24-04.1",
	}

	for field, expected := range knownMappings {
		val := d.Get(field).(string)
		if val != expected {
			t.Errorf("field %q: expected %q, got %q", field, expected, val)
		}
	}
}

func TestDiskImageFieldNameMapping(t *testing.T) {
	// Verify the string replacement logic maps image names to valid Terraform field names
	tests := []struct {
		imageName string
		fieldName string
	}{
		{"ubuntu-minimal.24-04.1", "ubuntu_minimal_24_04_1"},
		{"ubuntu.24-04.1", "ubuntu_24_04_1"},
		{"rocky.9-6.1", "rocky_9_6_1"},
	}

	for _, tt := range tests {
		t.Run(tt.imageName, func(t *testing.T) {
			fieldName := tt.imageName
			fieldName = strings.ReplaceAll(fieldName, ".", "_")
			fieldName = strings.ReplaceAll(fieldName, "-", "_")
			if fieldName != tt.fieldName {
				t.Errorf("expected field name %q, got %q", tt.fieldName, fieldName)
			}
		})
	}
}

func diagnosticsToString(diags diag.Diagnostics) string {
	var msgs []string
	for _, d := range diags {
		msgs = append(msgs, d.Summary)
	}
	return strings.Join(msgs, "; ")
}
