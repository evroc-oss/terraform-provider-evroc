// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
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
