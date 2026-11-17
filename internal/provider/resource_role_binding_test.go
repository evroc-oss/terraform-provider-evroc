// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccEvrocRoleBinding_Basic(t *testing.T) {
	saName := fmt.Sprintf("tf-test-sa-%d", time.Now().Unix())

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEvrocRoleBindingConfig_basic(saName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("evroc_role_binding.test", "roles.#", "2"),
					resource.TestCheckResourceAttr("evroc_role_binding.test", "roles.0.role", "/iam/roles/computeOperator"),
					resource.TestCheckResourceAttr("evroc_role_binding.test", "roles.1.role", "/iam/roles/networkingViewer"),
					resource.TestCheckResourceAttrPair("evroc_role_binding.test", "principal", "evroc_service_account.test", "fqid"),
				),
			},
		},
	})
}

func testAccEvrocRoleBindingConfig_basic(saName string) string {
	return fmt.Sprintf(`
resource "evroc_service_account" "test" {
  name        = "%s"
  description = "Role binding test SA"
  enabled     = true
}

resource "evroc_role_binding" "test" {
  principal = evroc_service_account.test.fqid

  roles {
    role = "computeOperator"
  }
  roles {
    role = "networkingViewer"
  }
}
`, saName)
}
