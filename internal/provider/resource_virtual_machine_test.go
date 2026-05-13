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

func TestAccEvrocVirtualMachine_Basic(t *testing.T) {
	resourceName := "evroc_virtual_machine.test"
	vmName := fmt.Sprintf("tf-test-vm-%d", time.Now().Unix())
	diskName := fmt.Sprintf("tf-test-disk-%d", time.Now().Unix())

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckEvrocVirtualMachineDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEvrocVirtualMachineConfig_basic(vmName, diskName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEvrocVirtualMachineExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", vmName),
					resource.TestCheckResourceAttr(resourceName, "flavor", "a1a.s"),
					resource.TestMatchResourceAttr(resourceName, "boot_disk", regexp.MustCompile(`/compute/projects/.+/regions/.+/disks/`+regexp.QuoteMeta(diskName)+`$`)),
					resource.TestCheckResourceAttrSet(resourceName, "vm_id"),
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

func testAccCheckEvrocVirtualMachineExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no virtual machine ID is set")
		}

		config := testAccProvider.Meta().(*ProviderConfig)
		_, err := config.Client.Compute().VirtualMachines().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("virtual machine %s not found: %w", rs.Primary.ID, err)
		}

		return nil
	}
}

func testAccCheckEvrocVirtualMachineDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*ProviderConfig)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "evroc_virtual_machine" {
			continue
		}

		_, err := config.Client.Compute().VirtualMachines().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			// VM not found - successfully deleted
			continue
		}

		return fmt.Errorf("virtual machine %s still exists", rs.Primary.ID)
	}

	return nil
}

func TestAccEvrocVirtualMachine_WithDataDisks(t *testing.T) {
	resourceName := "evroc_virtual_machine.test"
	vmName := fmt.Sprintf("tf-test-vm-dd-%d", time.Now().Unix())
	bootDiskName := fmt.Sprintf("tf-test-boot-%d", time.Now().Unix())
	dataDiskName := fmt.Sprintf("tf-test-data-%d", time.Now().Unix())

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckEvrocVirtualMachineDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEvrocVirtualMachineConfig_withDataDisks(vmName, bootDiskName, dataDiskName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEvrocVirtualMachineExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", vmName),
					resource.TestCheckResourceAttr(resourceName, "data_disks.#", "1"),
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

func testAccEvrocVirtualMachineConfig_basic(vmName, diskName string) string {
	return fmt.Sprintf(`
resource "evroc_disk" "test" {
  name          = "%s"
  size          = 100
  image         = "ubuntu-minimal.24-04.1"
  zone          = "a"
}

resource "evroc_virtual_machine" "test" {
  name      = "%s"
  flavor    = "a1a.s"
  boot_disk = evroc_disk.test.name
  zone      = "a"
}
`, diskName, vmName)
}

func testAccEvrocVirtualMachineConfig_withDataDisks(vmName, bootDiskName, dataDiskName string) string {
	return fmt.Sprintf(`
resource "evroc_disk" "boot" {
  name  = "%s"
  size  = 100
  image = "ubuntu-minimal.24-04.1"
  zone  = "a"
}

resource "evroc_disk" "data" {
  name = "%s"
  size = 50
  zone = "a"
}

resource "evroc_virtual_machine" "test" {
  name       = "%s"
  flavor     = "a1a.s"
  boot_disk  = evroc_disk.boot.name
  data_disks = [evroc_disk.data.fqid]
  zone       = "a"
}
`, bootDiskName, dataDiskName, vmName)
}
