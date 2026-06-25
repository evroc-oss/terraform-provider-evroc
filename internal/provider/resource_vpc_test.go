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

func TestAccEvrocVPC_DualStack(t *testing.T) {
	resourceName := "evroc_vpc.test"
	vpcName := fmt.Sprintf("tf-test-vpc-%d", time.Now().Unix())

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckEvrocVPCDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEvrocVPCConfig_dualStack(vpcName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEvrocVPCExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", vpcName),
					resource.TestCheckResourceAttr(resourceName, "stack_type", "dual-stack"),
					resource.TestCheckResourceAttr(resourceName, "ipv4_cidr_blocks.0", "10.0.0.0/16"),
					resource.TestCheckResourceAttrSet(resourceName, "vpc_id"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckResourceAttrSet(resourceName, "region"),
					resource.TestCheckResourceAttrSet(resourceName, "fqid"),
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

func TestAccEvrocVPC_IPv6Only(t *testing.T) {
	resourceName := "evroc_vpc.test"
	vpcName := fmt.Sprintf("tf-test-vpc-%d", time.Now().Unix())

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckEvrocVPCDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEvrocVPCConfig_ipv6Only(vpcName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEvrocVPCExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", vpcName),
					resource.TestCheckResourceAttr(resourceName, "stack_type", "ipv6-only"),
					resource.TestCheckResourceAttrSet(resourceName, "vpc_id"),
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

func testAccCheckEvrocVPCExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no VPC ID is set")
		}

		config := testAccProvider.Meta().(*ProviderConfig)
		_, err := config.Client.Networking().VirtualPrivateClouds().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("VPC %s not found: %w", rs.Primary.ID, err)
		}

		return nil
	}
}

func testAccCheckEvrocVPCDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*ProviderConfig)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "evroc_vpc" {
			continue
		}

		_, err := config.Client.Networking().VirtualPrivateClouds().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			continue
		}

		return fmt.Errorf("VPC %s still exists", rs.Primary.ID)
	}

	return nil
}

func testAccEvrocVPCConfig_dualStack(name string) string {
	return fmt.Sprintf(`
resource "evroc_vpc" "test" {
  name             = "%s"
  stack_type       = "dual-stack"
  ipv4_cidr_blocks = ["10.0.0.0/16"]
}
`, name)
}

func TestAccEvrocVPC_DualStackWithLabels(t *testing.T) {
	resourceName := "evroc_vpc.test"
	vpcName := fmt.Sprintf("tf-test-vpc-%d", time.Now().Unix())

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckEvrocVPCDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEvrocVPCConfig_dualStackWithLabels(vpcName, "v1"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEvrocVPCExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", vpcName),
					resource.TestCheckResourceAttr(resourceName, "stack_type", "dual-stack"),
					resource.TestCheckResourceAttr(resourceName, "user_labels.env", "test"),
					resource.TestCheckResourceAttr(resourceName, "user_labels.version", "v1"),
				),
			},
			{
				Config: testAccEvrocVPCConfig_dualStackWithLabels(vpcName, "v2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEvrocVPCExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "user_labels.version", "v2"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"assigned_ipv4_cidr_blocks", "assigned_ipv6_gua_block", "available_ipv4_cidr_blocks"},
			},
		},
	})
}

func TestAccEvrocVPC_DataSource(t *testing.T) {
	vpcName := fmt.Sprintf("tf-test-vpc-%d", time.Now().Unix())

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckEvrocVPCDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEvrocVPCConfig_dataSource(vpcName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.evroc_vpc.test", "name", vpcName),
					resource.TestCheckResourceAttr("data.evroc_vpc.test", "stack_type", "dual-stack"),
					resource.TestCheckResourceAttrSet("data.evroc_vpc.test", "vpc_id"),
					resource.TestCheckResourceAttrSet("data.evroc_vpc.test", "fqid"),
				),
			},
		},
	})
}

func testAccEvrocVPCConfig_ipv6Only(name string) string {
	return fmt.Sprintf(`
resource "evroc_vpc" "test" {
  name       = "%s"
  stack_type = "ipv6-only"
}
`, name)
}

func testAccEvrocVPCConfig_dualStackWithLabels(name, version string) string {
	return fmt.Sprintf(`
resource "evroc_vpc" "test" {
  name             = "%s"
  stack_type       = "dual-stack"
  ipv4_cidr_blocks = ["10.0.0.0/16"]

  user_labels = {
    env     = "test"
    version = "%s"
  }
}
`, name, version)
}

func testAccEvrocVPCConfig_dataSource(name string) string {
	return fmt.Sprintf(`
resource "evroc_vpc" "test" {
  name             = "%s"
  stack_type       = "dual-stack"
  ipv4_cidr_blocks = ["10.0.0.0/16"]
}

data "evroc_vpc" "test" {
  name = evroc_vpc.test.name
}
`, name)
}
