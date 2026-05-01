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

func TestAccEvrocPlacementGroup_Basic(t *testing.T) {
	resourceName := "evroc_placement_group.test"
	pgName := fmt.Sprintf("tf-test-pg-%d", time.Now().Unix())

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckEvrocPlacementGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEvrocPlacementGroupConfig_basic(pgName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEvrocPlacementGroupExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", pgName),
					resource.TestCheckResourceAttr(resourceName, "strategy", "spread"),
					resource.TestCheckResourceAttrSet(resourceName, "pg_id"),
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

func testAccCheckEvrocPlacementGroupExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no placement group ID is set")
		}

		config := testAccProvider.Meta().(*ProviderConfig)
		_, err := config.Client.Compute().PlacementGroups().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("placement group %s not found: %w", rs.Primary.ID, err)
		}

		return nil
	}
}

func testAccCheckEvrocPlacementGroupDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*ProviderConfig)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "evroc_placement_group" {
			continue
		}

		// Check if placement group still exists
		_, err := config.Client.Compute().PlacementGroups().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			// Placement group not found - successfully deleted
			continue
		}

		// Placement group still exists
		return fmt.Errorf("placement group %s still exists", rs.Primary.ID)
	}

	return nil
}

func testAccEvrocPlacementGroupConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "evroc_placement_group" "test" {
  name     = "%s"
  strategy = "spread"
  zone     = "a"
}
`, name)
}
