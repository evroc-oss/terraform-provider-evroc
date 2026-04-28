// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccEvrocSecurityGroup_Basic(t *testing.T) {
	resourceName := "evroc_security_group.test"
	sgName := fmt.Sprintf("tf-test-sg-%d", time.Now().Unix())

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckEvrocSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEvrocSecurityGroupConfig_basic(sgName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEvrocSecurityGroupExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", sgName),
					resource.TestCheckResourceAttr(resourceName, "rule.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "sg_id"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckResourceAttrSet(resourceName, "region"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckEvrocSecurityGroupExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no security group ID is set")
		}

		config := testAccProvider.Meta().(*ProviderConfig)
		_, err := config.Client.Networking().SecurityGroups().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("security group %s not found: %w", rs.Primary.ID, err)
		}

		return nil
	}
}

func testAccCheckEvrocSecurityGroupDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*ProviderConfig)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "evroc_security_group" {
			continue
		}

		// Check if security group still exists
		_, err := config.Client.Networking().SecurityGroups().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			// Security group not found - successfully deleted
			continue
		}

		// Security group still exists
		return fmt.Errorf("security group %s still exists", rs.Primary.ID)
	}

	return nil
}

func testAccEvrocSecurityGroupConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "evroc_security_group" "test" {
  name = "%s"

  rule {
    name      = "allow-ssh"
    direction = "Ingress"
    protocol  = "TCP"
    port      = 22
    remote_ip = "0.0.0.0/0"
  }
}
`, name)
}
