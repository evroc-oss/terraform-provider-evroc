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

func TestResourceCustomDiskImageLifecycle(t *testing.T) {
	const (
		bucketRef = "/storage/projects/test-project/regions/se-sto/buckets/images"
		saRef     = "/storage/projects/test-project/regions/se-sto/bucketServiceAccounts/image-reader"
		imageRef  = "/compute/projects/test-project/regions/se-sto/customDiskImages/node-v1"
	)
	for _, tc := range []struct{ name, bucket, sa string }{
		{"plain names", "images", "image-reader"},
		{"fqids", bucketRef, saRef},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ms := newMockServer()
			defer ms.close()
			config := newTestProviderConfig(t, ms.server.URL)
			mock := newCustomDiskImageMock(t, ms, "node-v1")

			res := resourceCustomDiskImage()
			d := schema.TestResourceDataRaw(t, res.Schema, map[string]interface{}{
				"name":                   "node-v1",
				"bucket":                 tc.bucket,
				"bucket_service_account": tc.sa,
				"object_path":            "images/node-v1.qcow2",
				"object_version":         "v3",
				"default_disk_size":      50,
				"os_name":                "ubuntu",
				"user_labels":            map[string]interface{}{"team": "platform"},
			})
			ctx := context.Background()
			if diags := resourceCustomDiskImageCreate(ctx, d, config); diags.HasError() {
				t.Fatal(diags)
			}

			// Names are expanded to FQIDs in the client's project and region.
			created := mock.created
			src := created.Spec.Source
			if src.BucketRef != bucketRef || src.ServiceAccountRef != saRef || src.Path != "images/node-v1.qcow2" || src.Version == nil || *src.Version != "v3" {
				t.Fatalf("source sent = %+v", src)
			}
			if created.Spec.DefaultDiskSize.Amount != 50 || created.Spec.DefaultDiskSize.Unit != "GB" || created.Spec.Architecture != "amd64" {
				t.Fatalf("size/arch sent = %+v / %s", created.Spec.DefaultDiskSize, created.Spec.Architecture)
			}
			assertAttrs(t, d, map[string]interface{}{
				"fqid": imageRef, "bucket": bucketRef, "bucket_service_account": saRef,
				"object_version": "v3", "default_disk_size": 50, "ready": true, "os_name": "ubuntu", "region": "se-sto",
			})

			// Update through a real state/config diff: only mutable fields are
			// patched, the immutable source is never resent, and a plain-name
			// config against an FQID state must not force replacement.
			cfg := map[string]interface{}{
				"name":                   "node-v1",
				"bucket":                 tc.bucket,
				"bucket_service_account": tc.sa,
				"object_path":            "images/node-v1.qcow2",
				"object_version":         "v3",
				"default_disk_size":      50,
				"os_name":                "ubuntu",
				"description":            "k8s 1.35 node",
				"user_labels":            map[string]interface{}{"team": "platform"},
			}
			state := d.State()
			diff, err := res.Diff(ctx, state, terraform.NewResourceConfigRaw(cfg), config)
			if err != nil {
				t.Fatal(err)
			}
			if diff == nil || diff.RequiresNew() {
				t.Fatalf("expected in-place update, diff=%#v", diff)
			}
			newState, diags := res.Apply(ctx, state, diff, config)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if len(mock.patches) != 1 {
				t.Fatalf("patches = %d", len(mock.patches))
			}
			spec := mock.patches[0]["spec"].(map[string]interface{})
			if spec["description"] != "k8s 1.35 node" || len(spec) != 1 {
				t.Errorf("patch spec = %v", spec)
			}
			if _, present := mock.patches[0]["metadata"]; present {
				t.Errorf("labels patched without change: %v", mock.patches[0])
			}
			if newState.Attributes["description"] != "k8s 1.35 node" {
				t.Errorf("description not refreshed: %v", newState.Attributes["description"])
			}

			if diags := resourceCustomDiskImageDelete(ctx, d, config); diags.HasError() {
				t.Fatal(diags)
			}
			d.SetId("node-v1")
			if diags := resourceCustomDiskImageRead(ctx, d, config); diags.HasError() || d.Id() != "" {
				t.Fatalf("read after delete: id=%q diags=%v", d.Id(), diags)
			}
		})
	}
}

// customDiskImageMock serves one custom disk image at the compute v1beta2
// path, recording the create request and every patch body.
type customDiskImageMock struct {
	created computetypes.CustomDiskImageRequest
	patches []map[string]interface{}
	stored  *computetypes.CustomDiskImage
}

func newCustomDiskImageMock(t *testing.T, ms *mockServer, name string) *customDiskImageMock {
	m := &customDiskImageMock{}
	base := "/compute/v1beta2/projects/test-project/regions/se-sto/customDiskImages"
	ms.mux.HandleFunc(base, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&m.created); err != nil {
			t.Error(err)
		}
		region := "se-sto"
		conds := []computetypes.CustomDiskImageStatusConditionsItem{{Type: "Ready", Status: "True"}}
		m.stored = &computetypes.CustomDiskImage{
			Metadata: computetypes.RegionalMetadataResponse{Id: m.created.Metadata.Id, Region: &region, UserLabels: m.created.Metadata.UserLabels},
			Spec:     m.created.Spec,
			Status:   computetypes.CustomDiskImageStatus{Conditions: &conds},
		}
		respondJSON(w, http.StatusCreated, m.stored)
	})
	ms.mux.HandleFunc(base+"/"+name, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if ms.isDeleted(base + "/" + name) {
				respondJSON(w, http.StatusNotFound, map[string]string{"reason": "not found"})
				return
			}
			respondJSON(w, http.StatusOK, m.stored)
		case http.MethodPatch:
			var patch map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
				t.Error(err)
			}
			m.patches = append(m.patches, patch)
			if spec, ok := patch["spec"].(map[string]interface{}); ok {
				if v, ok := spec["description"].(string); ok {
					m.stored.Spec.Description = &v
				}
			}
			respondJSON(w, http.StatusOK, m.stored)
		case http.MethodDelete:
			ms.markDeleted(base + "/" + name)
			w.WriteHeader(http.StatusNoContent)
		}
	})
	return m
}

func assertAttrs(t *testing.T, d *schema.ResourceData, want map[string]interface{}) {
	t.Helper()
	for key, w := range want {
		if got := d.Get(key); got != w {
			t.Errorf("%s = %v, want %v", key, got, w)
		}
	}
}
