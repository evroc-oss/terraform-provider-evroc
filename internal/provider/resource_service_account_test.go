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

func TestAccEvrocServiceAccount_Basic(t *testing.T) {
	resourceName := "evroc_service_account.test"
	saName := fmt.Sprintf("tf-test-sa-%d", time.Now().Unix())

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckEvrocServiceAccountDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEvrocServiceAccountConfig_basic(saName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEvrocServiceAccountExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", saName),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "description", "Acceptance test service account"),
					resource.TestCheckResourceAttr(resourceName, "user_labels.environment", "test"),
					resource.TestCheckResourceAttrSet(resourceName, "service_account_id"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
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

func TestAccEvrocServiceAccount_Update(t *testing.T) {
	resourceName := "evroc_service_account.test"
	saName := fmt.Sprintf("tf-test-sa-%d", time.Now().Unix())

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckEvrocServiceAccountDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEvrocServiceAccountConfig_basic(saName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEvrocServiceAccountExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "description", "Acceptance test service account"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
				),
			},
			{
				Config: testAccEvrocServiceAccountConfig_updated(saName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", "Updated description"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
				),
			},
		},
	})
}

func testAccCheckEvrocServiceAccountExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no service account ID is set")
		}

		config := testAccProvider.Meta().(*ProviderConfig)
		_, err := config.Client.IAM().ServiceAccounts().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("service account %s not found: %w", rs.Primary.ID, err)
		}

		return nil
	}
}

func testAccCheckEvrocServiceAccountDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*ProviderConfig)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "evroc_service_account" {
			continue
		}

		_, err := config.Client.IAM().ServiceAccounts().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			continue
		}

		return fmt.Errorf("service account %s still exists", rs.Primary.ID)
	}

	return nil
}

func testAccEvrocServiceAccountConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "evroc_service_account" "test" {
  name        = "%s"
  description = "Acceptance test service account"
  enabled     = true

  user_labels = {
    environment = "test"
  }
}
`, name)
}

func testAccEvrocServiceAccountConfig_updated(name string) string {
	return fmt.Sprintf(`
resource "evroc_service_account" "test" {
  name        = "%s"
  description = "Updated description"
  enabled     = false

  user_labels = {
    environment = "test"
    updated     = "true"
  }
}
`, name)
}
