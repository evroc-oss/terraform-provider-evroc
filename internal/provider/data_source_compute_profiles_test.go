// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"net/http"
	"strings"
	"testing"

	computetypes "github.com/evroc-oss/evroc-go-sdk/types/compute"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccEvrocComputeProfiles(t *testing.T) {
	dsName := "data.evroc_compute_profiles.all"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "evroc_compute_profiles" "all" {}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(dsName, "profiles.#"),
					resource.TestCheckResourceAttrSet(dsName, "series.#"),
					// Every profile has a detail entry, in the same order
					resource.TestCheckResourceAttrPair(dsName, "details.#", dsName, "profiles.#"),
					resource.TestCheckResourceAttrPair(dsName, "details.0.name", dsName, "profiles.0"),
					resource.TestCheckResourceAttrSet(dsName, "details.0.vcpus"),
					resource.TestCheckResourceAttrSet(dsName, "details.0.memory_amount"),
					resource.TestCheckResourceAttrSet(dsName, "details.0.memory_unit"),
					resource.TestCheckResourceAttrSet(dsName, "details.0.processor_architecture"),
				),
			},
		},
	})
}

func mockComputeProfile(id string) computetypes.Computeprofile {
	profile := computetypes.Computeprofile{
		ApiVersion: "compute/v1beta2",
		Kind:       "ComputeProfile",
		Metadata:   computetypes.GlobalMetadataResponse{Id: id},
		Spec: computetypes.ComputeprofileSpec{
			Memory:                computetypes.ComputeprofileSpecMemory{Amount: 4, Unit: "GB"},
			ProcessorArchitecture: "amd64",
			VCPUs:                 2,
		},
	}
	// GPU series ("gn-" prefix) get a GPU spec, mirroring the real catalog.
	if strings.HasPrefix(id, "gn-") {
		profile.Spec.Gpus = &computetypes.ComputeprofileSpecGpus{
			LocalDisk: 100,
			Model:     "l40s",
			Quantity:  1,
		}
	}
	return profile
}

func setupComputeProfileHandlers(ms *mockServer, ids ...string) {
	items := make([]computetypes.Computeprofile, 0, len(ids))
	for _, id := range ids {
		items = append(items, mockComputeProfile(id))
	}
	ms.mux.HandleFunc("/compute/v1beta2/global/computeProfiles", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusOK, map[string]interface{}{"items": items})
	})
}

func TestDataSourceComputeProfilesRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	// Include a profile with no named schema field ("x9z.s") to verify it only
	// shows up in the lists.
	setupComputeProfileHandlers(ms, "a1a.s", "a1a.m", "c1a.s", "gn-l40s.s", "x9z.s")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := dataSourceComputeProfiles()
	d := newTestResourceData(t, res)

	diags := dataSourceComputeProfilesRead(context.Background(), d, config)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diagnosticsToString(diags))
	}

	// Verify ID is set
	if d.Id() != "compute-profiles" {
		t.Errorf("expected ID %q, got %q", "compute-profiles", d.Id())
	}

	// Verify profiles list contains everything the API returned, sorted
	profiles := d.Get("profiles").([]interface{})
	expectedProfiles := []string{"a1a.m", "a1a.s", "c1a.s", "gn-l40s.s", "x9z.s"}
	if len(profiles) != len(expectedProfiles) {
		t.Fatalf("expected %d profiles, got %d", len(expectedProfiles), len(profiles))
	}
	for i, expected := range expectedProfiles {
		if profiles[i].(string) != expected {
			t.Errorf("profiles[%d]: expected %q, got %q", i, expected, profiles[i])
		}
	}

	// Verify series grouping by prefix
	series := d.Get("series").([]interface{})
	expectedSeries := map[string][]string{
		"a1a":     {"a1a.m", "a1a.s"},
		"c1a":     {"c1a.s"},
		"gn-l40s": {"gn-l40s.s"},
		"x9z":     {"x9z.s"},
	}
	if len(series) != len(expectedSeries) {
		t.Fatalf("expected %d series, got %d", len(expectedSeries), len(series))
	}
	for _, s := range series {
		seriesMap := s.(map[string]interface{})
		name := seriesMap["name"].(string)
		expectedSizes, ok := expectedSeries[name]
		if !ok {
			t.Errorf("unexpected series %q", name)
			continue
		}
		sizes := seriesMap["sizes"].([]interface{})
		if len(sizes) != len(expectedSizes) {
			t.Errorf("series %q: expected %d sizes, got %d", name, len(expectedSizes), len(sizes))
			continue
		}
		for i, expected := range expectedSizes {
			if sizes[i].(string) != expected {
				t.Errorf("series %q sizes[%d]: expected %q, got %q", name, i, expected, sizes[i])
			}
		}
	}

	// Known series get a description from the SDK catalog; unknown ones are empty
	for _, s := range series {
		seriesMap := s.(map[string]interface{})
		name := seriesMap["name"].(string)
		desc := seriesMap["description"].(string)
		if name == "a1a" && desc == "" {
			t.Error("expected non-empty description for series a1a")
		}
		if name == "x9z" && desc != "" {
			t.Errorf("expected empty description for unknown series x9z, got %q", desc)
		}
	}

	// Verify individual named fields are set for offered profiles
	knownMappings := map[string]string{
		"a1a_s":     "a1a.s",
		"a1a_m":     "a1a.m",
		"c1a_s":     "c1a.s",
		"gn_l40s_s": "gn-l40s.s",
	}
	for field, expected := range knownMappings {
		val := d.Get(field).(string)
		if val != expected {
			t.Errorf("field %q: expected %q, got %q", field, expected, val)
		}
	}

	// A profile the API does not offer leaves its named attribute empty
	if val := d.Get("m1a_s").(string); val != "" {
		t.Errorf("expected empty m1a_s, got %q", val)
	}

	// Verify details align with the profiles list and expose spec fields
	details := d.Get("details").([]interface{})
	if len(details) != len(profiles) {
		t.Fatalf("expected %d details, got %d", len(profiles), len(details))
	}
	for i, det := range details {
		assertComputeProfileDetail(t, i, det.(map[string]interface{}), profiles[i].(string))
	}
}

// assertComputeProfileDetail checks one details entry against the mockComputeProfile spec.
func assertComputeProfileDetail(t *testing.T, i int, detMap map[string]interface{}, expectedName string) {
	t.Helper()

	name := detMap["name"].(string)
	if name != expectedName {
		t.Errorf("details[%d].name %q does not match profiles[%d] %q", i, name, i, expectedName)
	}
	if detMap["vcpus"].(int) != 2 {
		t.Errorf("details[%d]: expected 2 vcpus, got %v", i, detMap["vcpus"])
	}
	if detMap["memory_amount"].(int) != 4 || detMap["memory_unit"].(string) != "GB" {
		t.Errorf("details[%d]: expected 4 GB memory, got %v %v", i, detMap["memory_amount"], detMap["memory_unit"])
	}
	if detMap["processor_architecture"].(string) != "amd64" {
		t.Errorf("details[%d]: expected amd64, got %v", i, detMap["processor_architecture"])
	}

	if strings.HasPrefix(name, "gn-") {
		if detMap["gpu_model"].(string) != "l40s" || detMap["gpu_quantity"].(int) != 1 || detMap["gpu_local_disk_gb"].(int) != 100 {
			t.Errorf("details[%d] (%s): unexpected GPU fields %v/%v/%v", i, name,
				detMap["gpu_model"], detMap["gpu_quantity"], detMap["gpu_local_disk_gb"])
		}
	} else {
		if detMap["gpu_model"].(string) != "" || detMap["gpu_quantity"].(int) != 0 || detMap["gpu_local_disk_gb"].(int) != 0 {
			t.Errorf("details[%d] (%s): expected empty GPU fields, got %v/%v/%v", i, name,
				detMap["gpu_model"], detMap["gpu_quantity"], detMap["gpu_local_disk_gb"])
		}
	}
}

func TestDataSourceComputeProfilesReadError(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupCatchAll(ms) // no computeProfiles handler: the API call returns not found

	config := newTestProviderConfig(t, ms.server.URL)
	res := dataSourceComputeProfiles()
	d := newTestResourceData(t, res)

	diags := dataSourceComputeProfilesRead(context.Background(), d, config)
	if !diags.HasError() {
		t.Fatal("expected error when listing compute profiles fails")
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
		{"gn-b200.s", "gn_b200_s"},
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
