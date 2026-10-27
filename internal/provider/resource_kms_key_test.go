// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	kmstypes "github.com/evroc-oss/evroc-go-sdk/types/kms"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccEvrocKMSKey(t *testing.T) {
	name := fmt.Sprintf("tf-kms-%d", time.Now().UnixNano())
	const address = "evroc_kms_key.test"
	config := func(enabled bool, days int) string {
		return fmt.Sprintf(`
resource "evroc_kms_key" "test" {
 name = %q
 enabled = %t
 auto_rotate_after_days = %d
 expire_after_days = 365
}
data "evroc_kms_key" "test" { name = evroc_kms_key.test.name }
`, name, enabled, days)
	}
	resource.Test(t, resource.TestCase{PreventPostDestroyRefresh: true,
		PreCheck: func() { testAccPreCheck(t) }, ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			for _, r := range s.RootModule().Resources {
				if r.Type == "evroc_kms_key" {
					key, err := testAccProvider.Meta().(*ProviderConfig).Client.KMS().Keys().Get(ctx, r.Primary.ID)
					if isNotFoundError(err) {
						continue
					}
					if err != nil {
						return err
					}
					if key.Status.State != "PendingDeletion" && key.Status.State != "Deleted" {
						return fmt.Errorf("key %s is still %s", r.Primary.ID, key.Status.State)
					}
				}
			}
			return nil
		},
		Steps: []resource.TestStep{
			{Config: config(true, 30), Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(address, "state", "Active"),
				resource.TestCheckResourceAttr(address, "auto_rotate_after_days", "30"),
				resource.TestCheckResourceAttr(address, "expire_after_days", "365"),
				resource.TestCheckResourceAttr(address, "imported", "false"),
				resource.TestCheckResourceAttrSet(address, "versions.0.version"),
				resource.TestCheckResourceAttrPair(address, "fqid", "data.evroc_kms_key.test", "fqid"),
			)},
			{ResourceName: address, ImportState: true, ImportStateVerify: true},
			{Config: config(false, 30), Check: resource.TestCheckResourceAttr(address, "state", "Suspended")},
			{Config: config(true, 30), Check: resource.TestCheckResourceAttr(address, "state", "Active")},
			{Config: config(true, 31), PlanOnly: true, ExpectNonEmptyPlan: true, ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPreRefresh: []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionDestroyBeforeCreate)}}},
		},
	})
}

func TestKMSKeyLifecycle(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	key := kmstypes.Key{Metadata: kmstypes.RegionalMetadataResponse{Id: "test-key"}, Status: kmstypes.KeyStatus{State: "PreActivated"}}
	missing := false
	ms.mux.HandleFunc("/kms/v1/projects/test-project/regions/se-sto/keys", func(w http.ResponseWriter, r *http.Request) {
		var req kmstypes.KeyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		key.Spec = req.Spec
		if req.ApiVersion != "kms/v1" || req.Metadata.Id != "test-key" || req.Spec.AutoRotateAfterDays == nil || *req.Spec.AutoRotateAfterDays != 30 {
			t.Errorf("unexpected create: %+v", req)
		}
		respondJSON(w, http.StatusCreated, key)
	})
	ms.mux.HandleFunc("/kms/v1/projects/test-project/regions/se-sto/keys/test-key", func(w http.ResponseWriter, r *http.Request) {
		if missing {
			respondJSON(w, 404, map[string]string{"reason": "NotFound"})
			return
		}
		if r.Method == http.MethodDelete {
			key.Status.State = "PendingDeletion"
			w.WriteHeader(202)
			return
		}
		respondJSON(w, 200, key)
	})
	ms.mux.HandleFunc("/kms/v1/projects/test-project/regions/se-sto/keys/test-key/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/activate"):
			key.Status.State = "Active"
		case strings.HasSuffix(r.URL.Path, "/suspend"):
			key.Status.State = "Suspended"
		default:
			t.Errorf("unexpected action: %s", r.URL.Path)
		}
		w.WriteHeader(202)
	})
	ctx := context.Background()
	cfg := newTestProviderConfig(t, ms.server.URL)
	res := resourceKMSKey()
	d := schema.TestResourceDataRaw(t, res.Schema, map[string]interface{}{"name": "test-key", "auto_rotate_after_days": 30})
	if diags := res.CreateContext(ctx, d, cfg); diags.HasError() {
		t.Fatal(diags)
	}
	assertField(t, d, "enabled", true)
	assertField(t, d, "state", "Active")
	for _, enabled := range []bool{false, true} {
		if err := d.Set("enabled", enabled); err != nil {
			t.Fatal(err)
		}
		if diags := res.UpdateContext(ctx, d, cfg); diags.HasError() {
			t.Fatal(diags)
		}
		assertField(t, d, "enabled", enabled)
	}
	lookup := dataSourceKMSKey()
	ld := schema.TestResourceDataRaw(t, lookup.Schema, map[string]interface{}{"name": "test-key"})
	if diags := lookup.ReadContext(ctx, ld, cfg); diags.HasError() {
		t.Fatal(diags)
	}
	assertField(t, ld, "auto_rotate_after_days", 30)
	if diags := res.DeleteContext(ctx, d, cfg); diags.HasError() {
		t.Fatal(diags)
	}
	if d.Id() != "" || key.Status.State != "PendingDeletion" {
		t.Fatal("delete must forget the key only after scheduling deletion")
	}
	missing = true
	d.SetId("test-key")
	if diags := res.ReadContext(ctx, d, cfg); diags.HasError() || d.Id() != "" {
		t.Fatalf("missing resource: %v", diags)
	}
	if diags := lookup.ReadContext(ctx, ld, cfg); !diags.HasError() {
		t.Fatal("missing data source must fail")
	}
}
