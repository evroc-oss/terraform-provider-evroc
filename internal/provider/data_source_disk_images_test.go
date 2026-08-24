// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"net/http"
	"strings"
	"testing"

	computetypes "github.com/evroc-oss/evroc-go-sdk/types/compute"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccEvrocDiskImages(t *testing.T) {
	dsName := "data.evroc_disk_images.all"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "evroc_disk_images" "all" {}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(dsName, "images.#"),
					// Every image has a detail entry, in the same order
					resource.TestCheckResourceAttrPair(dsName, "details.#", dsName, "images.#"),
					resource.TestCheckResourceAttrPair(dsName, "details.0.name", dsName, "images.0"),
					resource.TestCheckResourceAttrSet(dsName, "details.0.os_image"),
					resource.TestCheckResourceAttrSet(dsName, "details.0.os_version"),
					resource.TestCheckResourceAttrSet(dsName, "details.0.default_size_amount"),
					resource.TestCheckResourceAttrSet(dsName, "details.0.default_size_unit"),
				),
			},
		},
	})
}

func mockDiskImage(id string) computetypes.Diskimage {
	osArch := computetypes.Amd64
	return computetypes.Diskimage{
		ApiVersion: "compute/v1beta2",
		Kind:       "DiskImage",
		Metadata:   computetypes.GlobalMetadataResponse{Id: id},
		Spec: computetypes.DiskimageSpec{
			DefaultSize: computetypes.DiskimageSpecDefaultSize{Amount: 10, Unit: "GB"},
			OsArch:      &osArch,
			OsImage:     id,
			OsVersion:   "1",
			Version:     1,
		},
	}
}

func setupDiskImageHandlers(ms *mockServer, ids ...string) {
	items := make([]computetypes.Diskimage, 0, len(ids))
	for _, id := range ids {
		items = append(items, mockDiskImage(id))
	}
	ms.mux.HandleFunc("/compute/v1beta2/global/diskImages/evroc", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusOK, map[string]interface{}{"items": items})
	})
}

func TestDataSourceDiskImagesRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	// Include an image with no named schema field to verify it only shows up in the list.
	setupDiskImageHandlers(ms, "ubuntu-minimal.24-04.1", "ubuntu.24-04.1", "debian.13.1")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := dataSourceDiskImages()
	d := newTestResourceData(t, res)

	diags := dataSourceDiskImagesRead(context.Background(), d, config)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diagnosticsToString(diags))
	}

	// Verify ID is set
	if d.Id() != "disk-images" {
		t.Errorf("expected ID %q, got %q", "disk-images", d.Id())
	}

	// Verify images list contains everything the API returned, sorted
	images := d.Get("images").([]interface{})
	expectedImages := []string{"debian.13.1", "ubuntu-minimal.24-04.1", "ubuntu.24-04.1"}
	if len(images) != len(expectedImages) {
		t.Fatalf("expected %d images, got %d", len(expectedImages), len(images))
	}
	for i, expected := range expectedImages {
		if images[i].(string) != expected {
			t.Errorf("images[%d]: expected %q, got %q", i, expected, images[i])
		}
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

	// An image the API does not offer leaves its named attribute empty
	if val := d.Get("rocky_9_6_1").(string); val != "" {
		t.Errorf("expected empty rocky_9_6_1, got %q", val)
	}

	// Verify details align with the images list and expose spec fields
	details := d.Get("details").([]interface{})
	if len(details) != len(images) {
		t.Fatalf("expected %d details, got %d", len(images), len(details))
	}
	for i, det := range details {
		detMap := det.(map[string]interface{})
		if detMap["name"].(string) != images[i].(string) {
			t.Errorf("details[%d].name %q does not match images[%d] %q", i, detMap["name"], i, images[i])
		}
		if detMap["os_image"].(string) != images[i].(string) {
			t.Errorf("details[%d]: expected os_image %q, got %v", i, images[i], detMap["os_image"])
		}
		if detMap["os_version"].(string) != "1" || detMap["version"].(int) != 1 {
			t.Errorf("details[%d]: unexpected version fields %v/%v", i, detMap["os_version"], detMap["version"])
		}
		if detMap["os_arch"].(string) != "amd64" {
			t.Errorf("details[%d]: expected amd64, got %v", i, detMap["os_arch"])
		}
		if detMap["default_size_amount"].(int) != 10 || detMap["default_size_unit"].(string) != "GB" {
			t.Errorf("details[%d]: expected default size 10 GB, got %v %v", i, detMap["default_size_amount"], detMap["default_size_unit"])
		}
		if affinities := detMap["gpu_affinities"].([]interface{}); len(affinities) != 0 {
			t.Errorf("details[%d]: expected no gpu_affinities, got %v", i, affinities)
		}
	}
}

func TestDataSourceDiskImagesReadError(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupCatchAll(ms) // no diskImages handler: the API call returns not found

	config := newTestProviderConfig(t, ms.server.URL)
	res := dataSourceDiskImages()
	d := newTestResourceData(t, res)

	diags := dataSourceDiskImagesRead(context.Background(), d, config)
	if !diags.HasError() {
		t.Fatal("expected error when listing disk images fails")
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
