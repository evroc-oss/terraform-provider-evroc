// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccEvrocProject_Basic(t *testing.T) {
	if os.Getenv("EVROC_ORGANIZATION") == "" {
		t.Skip("EVROC_ORGANIZATION not set, skipping project acceptance test")
	}

	resourceName := "evroc_project.test"
	projectName := fmt.Sprintf("tf-test-project-%d", time.Now().Unix())

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckEvrocProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEvrocProjectConfig_basic(projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "project_id"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckResourceAttr(resourceName, "name", projectName),
					resource.TestCheckResourceAttr(resourceName, "display_name", "Test Project"),
					resource.TestCheckResourceAttr(resourceName, "user_labels.environment", "test"),
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

func testAccCheckEvrocProjectDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*ProviderConfig)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "evroc_project" {
			continue
		}

		_, err := config.Client.IAM().Projects().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			// Project not found - successfully deleted
			continue
		}

		return fmt.Errorf("project %s still exists", rs.Primary.ID)
	}

	return nil
}

func testAccEvrocProjectConfig_basic(name string) string {
	org := os.Getenv("EVROC_ORGANIZATION")
	return fmt.Sprintf(`
resource "evroc_project" "test" {
  name         = "%s"
  display_name = "Test Project"
  organization = "%s"

  user_labels = {
    environment = "test"
    team        = "platform"
  }
}
`, name, org)
}
