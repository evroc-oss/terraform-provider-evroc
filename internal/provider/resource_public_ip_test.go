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

func TestAccEvrocPublicIP_Basic(t *testing.T) {
	resourceName := "evroc_public_ip.test"
	ipName := fmt.Sprintf("tf-test-ip-%d", time.Now().Unix())

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckEvrocPublicIPDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEvrocPublicIPConfig_basic(ipName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEvrocPublicIPExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", ipName),
					resource.TestCheckResourceAttrSet(resourceName, "ip_id"),
					resource.TestCheckResourceAttrSet(resourceName, "ip_address"),
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

func testAccCheckEvrocPublicIPExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no public IP ID is set")
		}

		config := testAccProvider.Meta().(*ProviderConfig)
		_, err := config.Client.Networking().PublicIPs().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("public IP %s not found: %w", rs.Primary.ID, err)
		}

		return nil
	}
}

func testAccCheckEvrocPublicIPDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*ProviderConfig)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "evroc_public_ip" {
			continue
		}

		// Check if public IP still exists
		_, err := config.Client.Networking().PublicIPs().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			// Public IP not found - successfully deleted
			continue
		}

		// Public IP still exists
		return fmt.Errorf("public IP %s still exists", rs.Primary.ID)
	}

	return nil
}

func testAccEvrocPublicIPConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "evroc_public_ip" "test" {
  name = "%s"
}
`, name)
}
