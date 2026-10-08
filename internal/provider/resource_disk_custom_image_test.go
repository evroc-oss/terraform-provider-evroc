// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	computetypes "github.com/evroc-oss/evroc-go-sdk/types/compute"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestDiskImageRoundTrip(t *testing.T) {
	const custom = "/compute/projects/test-project/regions/se-sto/customDiskImages/node-v1"
	for _, tc := range []struct{ name, image, ref, source string }{
		{"custom", custom, custom, "image"},
		{"stock", "ubuntu.24-04.1", "/compute/global/diskImages/evroc/ubuntu.24-04.1", "image"},
		{"blank", "", "", "blank"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ms := newMockServer()
			defer ms.close()
			config := newTestProviderConfig(t, ms.server.URL)
			disk := mockDisk("test-disk")
			zone := "a"
			disk.Spec.Placement.Zone = &zone
			disk.Spec.Source = &computetypes.DiskSpecSource{Type: computetypes.DiskSpecSourceType(tc.source)}
			if tc.ref != "" {
				disk.Spec.Source.DiskImageRef = &tc.ref
			}
			base := "/compute/v1beta2/projects/test-project/regions/se-sto/disks"
			creates := 0
			ms.mux.HandleFunc(base, func(w http.ResponseWriter, r *http.Request) {
				creates++
				if r.Method != http.MethodPost {
					t.Errorf("method = %s", r.Method)
				}
				var req computetypes.DiskRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Error(err)
				}
				if req.Spec.Source == nil {
					t.Error("missing source")
				} else {
					if string(req.Spec.Source.Type) != tc.source {
						t.Errorf("source = %s, want %s", req.Spec.Source.Type, tc.source)
					}
					if tc.ref == "" {
						if req.Spec.Source.DiskImageRef != nil {
							t.Error("blank disk has image ref")
						}
					} else if req.Spec.Source.DiskImageRef == nil || *req.Spec.Source.DiskImageRef != tc.ref {
						t.Errorf("image ref = %v, want %s", req.Spec.Source.DiskImageRef, tc.ref)
					}
				}
				respondJSON(w, http.StatusCreated, disk)
			})
			ms.mux.HandleFunc(base+"/test-disk", func(w http.ResponseWriter, r *http.Request) {
				respondJSON(w, http.StatusOK, disk)
			})
			raw := map[string]interface{}{"name": "test-disk", "size": 100, "zone": "a", "project": "test-project", "region": "se-sto"}
			if tc.image != "" {
				raw["image"] = tc.image
			}
			res := resourceDisk()
			d := schema.TestResourceDataRaw(t, res.Schema, raw)
			ctx := context.Background()
			if diags := res.CreateContext(ctx, d, config); diags.HasError() {
				t.Fatal(diags)
			}
			if creates != 1 {
				t.Fatalf("create calls = %d", creates)
			}
			if got := d.Get("image"); got != tc.image {
				t.Fatalf("image after create = %v, want %s", got, tc.image)
			}
			// A refreshed/imported custom image must not produce a replacement plan.
			imported := schema.TestResourceDataRaw(t, res.Schema, map[string]interface{}{})
			imported.SetId("test-disk")
			if diags := res.ReadContext(ctx, imported, config); diags.HasError() {
				t.Fatal(diags)
			}
			if got := imported.Get("image"); got != tc.image {
				t.Fatalf("imported image = %v, want %s", got, tc.image)
			}
			diff, err := res.Diff(ctx, imported.State(), terraform.NewResourceConfigRaw(raw), config)
			if err != nil {
				t.Fatal(err)
			}
			if diff != nil && !diff.Empty() {
				t.Fatalf("unexpected diff after import: %#v", diff)
			}
			ds := dataSourceDisk()
			data := schema.TestResourceDataRaw(t, ds.Schema, map[string]interface{}{"name": "test-disk"})
			if diags := ds.ReadContext(ctx, data, config); diags.HasError() {
				t.Fatal(diags)
			}
			if got := data.Get("image"); got != tc.image {
				t.Fatalf("data source image = %v, want %s", got, tc.image)
			}
		})
	}
}

func TestCustomDiskImageReplacementAndSnapshotConflict(t *testing.T) {
	res := resourceDisk()
	const old = "/compute/projects/test-project/regions/se-sto/customDiskImages/node-v1"
	state := &terraform.InstanceState{ID: "test-disk", Attributes: map[string]string{"name": "test-disk", "zone": "a", "image": old}}
	raw := map[string]interface{}{"name": "test-disk", "zone": "a", "image": old + "-new"}
	diff, err := res.Diff(context.Background(), state, terraform.NewResourceConfigRaw(raw), nil)
	if err != nil {
		t.Fatal(err)
	}
	if diff == nil || !diff.RequiresNew() {
		t.Fatal("changing a custom image must replace the disk")
	}
	raw["snapshot"] = "test-snapshot"
	diags := res.Validate(terraform.NewResourceConfigRaw(raw))
	if !diags.HasError() {
		t.Fatal("image and snapshot must conflict")
	}
}
