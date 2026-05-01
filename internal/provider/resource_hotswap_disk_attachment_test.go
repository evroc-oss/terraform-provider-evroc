// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccEvrocHotswapDiskAttachment_Basic(t *testing.T) {
	resourceName := "evroc_hotswap_disk_attachment.test"
	vmName := fmt.Sprintf("tf-test-vm-%d", time.Now().Unix())
	bootDiskName := fmt.Sprintf("tf-test-boot-disk-%d", time.Now().Unix())
	dataDiskName := fmt.Sprintf("tf-test-data-disk-%d", time.Now().Unix())
	attachmentName := fmt.Sprintf("tf-test-attach-%d", time.Now().Unix())

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckEvrocHotswapDiskAttachmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEvrocHotswapDiskAttachmentConfig_basic(vmName, bootDiskName, dataDiskName, attachmentName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEvrocHotswapDiskAttachmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", attachmentName),
					resource.TestMatchResourceAttr(resourceName, "virtual_machine", regexp.MustCompile(`/compute/projects/.+/regions/.+/virtualMachines/`+regexp.QuoteMeta(vmName)+`$`)),
					resource.TestMatchResourceAttr(resourceName, "disk", regexp.MustCompile(`/compute/projects/.+/regions/.+/disks/`+regexp.QuoteMeta(dataDiskName)+`$`)),
					resource.TestCheckResourceAttrSet(resourceName, "attachment_id"),
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

func testAccCheckEvrocHotswapDiskAttachmentExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no disk attachment ID is set")
		}

		config := testAccProvider.Meta().(*ProviderConfig)
		_, err := config.Client.Compute().HotswapDiskAttachments().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("disk attachment %s not found: %w", rs.Primary.ID, err)
		}

		return nil
	}
}

func testAccCheckEvrocHotswapDiskAttachmentDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*ProviderConfig)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "evroc_hotswap_disk_attachment" {
			continue
		}

		// Check if disk attachment still exists
		_, err := config.Client.Compute().HotswapDiskAttachments().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			// Disk attachment not found - successfully deleted
			continue
		}

		// Disk attachment still exists
		return fmt.Errorf("disk attachment %s still exists", rs.Primary.ID)
	}

	return nil
}

func testAccEvrocHotswapDiskAttachmentConfig_basic(vmName, bootDiskName, dataDiskName, attachmentName string) string {
	return fmt.Sprintf(`
resource "evroc_disk" "boot" {
  name          = "%s"
  size          = 100
  image         = "ubuntu-minimal.24-04.1"
  zone          = "a"
}

resource "evroc_disk" "data" {
  name          = "%s"
  size          = 50
  zone          = "a"
}

resource "evroc_virtual_machine" "test" {
  name      = "%s"
  flavor    = "a1a.s"
  boot_disk = evroc_disk.boot.name
  zone      = "a"
}

resource "evroc_hotswap_disk_attachment" "test" {
  name            = "%s"
  virtual_machine = evroc_virtual_machine.test.name
  disk            = evroc_disk.data.name
}
`, bootDiskName, dataDiskName, vmName, attachmentName)
}
