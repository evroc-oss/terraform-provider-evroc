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

func TestAccEvrocDisk_Basic(t *testing.T) {
	resourceName := "evroc_disk.test"
	diskName := fmt.Sprintf("tf-test-disk-%d", time.Now().Unix())

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckEvrocDiskDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEvrocDiskConfig_basic(diskName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEvrocDiskExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", diskName),
					resource.TestCheckResourceAttr(resourceName, "size", "100"),
					resource.TestCheckResourceAttr(resourceName, "image", "ubuntu-minimal.24-04.1"),
					resource.TestCheckResourceAttrSet(resourceName, "disk_id"),
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

func testAccCheckEvrocDiskExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no disk ID is set")
		}

		config := testAccProvider.Meta().(*ProviderConfig)
		_, err := config.Client.Compute().Disks().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("disk %s not found: %w", rs.Primary.ID, err)
		}

		return nil
	}
}

func testAccCheckEvrocDiskDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*ProviderConfig)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "evroc_disk" {
			continue
		}

		_, err := config.Client.Compute().Disks().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			// Disk not found - successfully deleted
			continue
		}

		return fmt.Errorf("disk %s still exists", rs.Primary.ID)
	}

	return nil
}

func testAccEvrocDiskConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "evroc_disk" "test" {
  name          = "%s"
  size          = 100
  image         = "ubuntu-minimal.24-04.1"
  zone          = "a"
}
`, name)
}
