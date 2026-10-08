// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"net/http"
	"strings"
	"testing"

	evroc "github.com/evroc-oss/evroc-go-sdk"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestDataSourceCustomDiskImage(t *testing.T) {
	for _, tc := range []struct {
		name, project, region string
		status                int
	}{
		{"provider defaults", "test-project", "se-sto", http.StatusOK},
		{"project override", "another-project", "se-sto", http.StatusOK},
		{"provider region", "test-project", "fr-par", http.StatusOK},
		{"missing", "test-project", "se-sto", http.StatusNotFound},
		{"forbidden", "test-project", "se-sto", http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ms := newMockServer()
			defer ms.close()
			config := newTestProviderConfig(t, ms.server.URL)
			cfg := config.Client.Config()
			cfg.Context.Project = tc.project
			cfg.Context.Region = tc.region
			scoped, err := evroc.New(context.Background(), cfg, evroc.WithHTTPClient(http.DefaultClient))
			if err != nil {
				t.Fatal(err)
			}
			if tc.project == config.Project {
				config.Client = scoped
				config.Region = tc.region
			} else {
				config.clients[tc.project] = scoped
			}
			calls := 0
			ms.mux.HandleFunc("/compute/v1beta2/projects/"+tc.project+"/regions/"+tc.region+"/customDiskImages/node-v1", func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != http.MethodGet {
					t.Errorf("method = %s", r.Method)
				}
				if tc.status != http.StatusOK {
					respondJSON(w, tc.status, map[string]string{"reason": "unavailable"})
					return
				}
				respondJSON(w, http.StatusOK, map[string]interface{}{
					"metadata": map[string]string{"id": "node-v1", "project": tc.project, "region": tc.region},
					"spec": map[string]interface{}{
						"architecture":    "amd64",
						"defaultDiskSize": map[string]interface{}{"amount": 50, "unit": "GB"},
						"imageVersion":    "v1.35.0",
						"source": map[string]string{
							"bucketRef":         "/storage/projects/" + tc.project + "/regions/" + tc.region + "/buckets/images",
							"serviceAccountRef": "/storage/projects/" + tc.project + "/regions/" + tc.region + "/bucketServiceAccounts/image-reader",
							"path":              "images/node-v1.qcow2",
							"version":           "obj-v3",
						},
					},
					"status": map[string]interface{}{"conditions": []map[string]string{{"type": "Ready", "status": "True"}}},
				})
			})
			raw := map[string]interface{}{"name": "node-v1"}
			if tc.project != config.Project {
				raw["project"] = tc.project
			}
			res := New("test")().DataSourcesMap["evroc_custom_disk_image"]
			if res == nil {
				t.Fatal("data source not registered")
			}
			d := schema.TestResourceDataRaw(t, res.Schema, raw)
			diags := res.ReadContext(context.Background(), d, config)
			if calls != 1 {
				t.Fatalf("lookup calls = %d", calls)
			}
			if tc.status != http.StatusOK {
				if !diags.HasError() || !strings.Contains(diagnosticsToString(diags), "node-v1") {
					t.Fatalf("expected contextual error, got %v", diags)
				}
				if d.Id() != "" {
					t.Fatal("failed lookup acquired state")
				}
				return
			}
			if diags.HasError() {
				t.Fatal(diags)
			}
			ref := "/compute/projects/" + tc.project + "/regions/" + tc.region + "/customDiskImages/node-v1"
			for key, want := range map[string]interface{}{
				"name": "node-v1", "project": tc.project, "region": tc.region, "architecture": "amd64", "fqid": ref,
				"bucket":                 "/storage/projects/" + tc.project + "/regions/" + tc.region + "/buckets/images",
				"bucket_service_account": "/storage/projects/" + tc.project + "/regions/" + tc.region + "/bucketServiceAccounts/image-reader",
				"object_path":            "images/node-v1.qcow2",
				"object_version":         "obj-v3",
				"default_disk_size":      50,
				"ready":                  true,
				"image_version":          "v1.35.0",
			} {
				if got := d.Get(key); got != want {
					t.Errorf("%s = %v, want %v", key, got, want)
				}
			}
			if d.Id() != ref {
				t.Errorf("id = %s", d.Id())
			}
			disk := BuildDiskCreateRequest("boot", 50, d.Get("fqid").(string), "", "a", nil)
			if disk.Spec.Source == nil || disk.Spec.Source.DiskImageRef == nil || *disk.Spec.Source.DiskImageRef != ref {
				t.Fatal("lookup reference not preserved in disk request")
			}
		})
	}
}
