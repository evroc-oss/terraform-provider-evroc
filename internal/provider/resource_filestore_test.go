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

func TestAccEvrocFilestore_Basic(t *testing.T) {
	resourceName := "evroc_filestore.test"
	fsName := fmt.Sprintf("tf-test-fs-%d", time.Now().Unix())

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckEvrocFilestoreDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEvrocFilestoreConfig_basic(fsName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEvrocFilestoreExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", fsName),
					resource.TestCheckResourceAttr(resourceName, "zone", "a"),
					resource.TestCheckResourceAttr(resourceName, "status", "Available"),
					resource.TestCheckResourceAttrSet(resourceName, "nfs_endpoint"),
					resource.TestCheckResourceAttrSet(resourceName, "nfs_export_path"),
					resource.TestCheckResourceAttrSet(resourceName, "nfs_version"),
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

func TestAccEvrocFilestore_WithLabels(t *testing.T) {
	resourceName := "evroc_filestore.test"
	fsName := fmt.Sprintf("tf-test-fs-labels-%d", time.Now().Unix())

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckEvrocFilestoreDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEvrocFilestoreConfig_withLabels(fsName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEvrocFilestoreExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", fsName),
					resource.TestCheckResourceAttr(resourceName, "user_labels.env", "test"),
					resource.TestCheckResourceAttr(resourceName, "user_labels.team", "platform"),
				),
			},
		},
	})
}

func testAccCheckEvrocFilestoreExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no filestore ID is set")
		}

		config := testAccProvider.Meta().(*ProviderConfig)
		_, err := config.Client.Storage().FileStores().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("filestore %s not found: %w", rs.Primary.ID, err)
		}

		return nil
	}
}

func testAccCheckEvrocFilestoreDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*ProviderConfig)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "evroc_filestore" {
			continue
		}

		_, err := config.Client.Storage().FileStores().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			continue
		}

		return fmt.Errorf("filestore %s still exists", rs.Primary.ID)
	}

	return nil
}

func testAccEvrocFilestoreConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "evroc_filestore" "test" {
  name = "%s"
  zone = "a"
}
`, name)
}

func testAccEvrocFilestoreConfig_withLabels(name string) string {
	return fmt.Sprintf(`
resource "evroc_filestore" "test" {
  name = "%s"
  zone = "a"

  user_labels = {
    env  = "test"
    team = "platform"
  }
}
`, name)
}
