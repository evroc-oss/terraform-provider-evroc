// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccEvrocOrganizationQuota(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "evroc_organization_quota" "test" {}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.evroc_organization_quota.test", "compute_vcpus"),
					resource.TestCheckResourceAttrSet("data.evroc_organization_quota.test", "compute_memory"),
					resource.TestCheckResourceAttrSet("data.evroc_organization_quota.test", "networking_public_ips"),
				),
			},
		},
	})
}

func TestAccEvrocProjectQuota(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "evroc_project_quota" "test" {}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.evroc_project_quota.test", "object_storage_total_size"),
				),
			},
		},
	})
}

func TestDataSourceOrganizationQuotaReadGPUs(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	ms.mux.HandleFunc("/quotas/", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/orgQuotas") {
			http.NotFound(w, r)
			return
		}
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"apiVersion": "quotas/v1",
			"kind":       "OrgQuota",
			"spec": map[string]interface{}{
				"quotas": map[string]interface{}{
					"compute": map[string]interface{}{
						"vCPUs": 64,
						"gpus":  map[string]interface{}{"nvidia.com/AD102GL_L40S": 4, "nvidia.com/GB100_B200": 2},
					},
				},
			},
			"status": map[string]interface{}{
				"quotaUsage": map[string]interface{}{
					"compute": map[string]interface{}{
						"vCPUs": 8,
						"gpus":  map[string]interface{}{"nvidia.com/AD102GL_L40S": 1},
					},
				},
			},
		})
	})
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	d := newTestResourceData(t, dataSourceOrganizationQuota())

	diags := dataSourceOrganizationQuotaRead(context.Background(), d, config)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diagnosticsToString(diags))
	}

	limits := d.Get("compute_gpus").(map[string]interface{})
	if len(limits) != 2 || limits["nvidia.com/AD102GL_L40S"] != 4 || limits["nvidia.com/GB100_B200"] != 2 {
		t.Errorf("unexpected compute_gpus: %v", limits)
	}
	usage := d.Get("usage_gpus").(map[string]interface{})
	if len(usage) != 1 || usage["nvidia.com/AD102GL_L40S"] != 1 {
		t.Errorf("unexpected usage_gpus: %v", usage)
	}
	if d.Get("compute_vcpus").(int) != 64 {
		t.Errorf("expected compute_vcpus 64, got %v", d.Get("compute_vcpus"))
	}
}

func TestDataSourceOrganizationQuotaReadNoGPUs(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	ms.mux.HandleFunc("/quotas/", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/orgQuotas") {
			http.NotFound(w, r)
			return
		}
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"apiVersion": "quotas/v1",
			"kind":       "OrgQuota",
			"spec":       map[string]interface{}{"quotas": map[string]interface{}{"compute": map[string]interface{}{"vCPUs": 32}}},
			"status":     map[string]interface{}{},
		})
	})
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	d := newTestResourceData(t, dataSourceOrganizationQuota())

	diags := dataSourceOrganizationQuotaRead(context.Background(), d, config)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diagnosticsToString(diags))
	}

	// The maps must be empty rather than null so lookup() works in configs.
	attrs := d.State().Attributes
	for _, k := range []string{"compute_gpus.%", "usage_gpus.%"} {
		if v, ok := attrs[k]; !ok || v != "0" {
			t.Errorf("expected %s = \"0\", got %q (present: %v)", k, v, ok)
		}
	}
}
