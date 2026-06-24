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

func TestAccEvrocSubnet_DualStack(t *testing.T) {
	resourceName := "evroc_subnet.test"
	ts := time.Now().Unix()
	vpcName := fmt.Sprintf("tf-test-vpc-%d", ts)
	subnetName := fmt.Sprintf("tf-test-subnet-%d", ts)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckEvrocSubnetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEvrocSubnetConfig_dualStack(vpcName, subnetName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEvrocSubnetExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", subnetName),
					resource.TestCheckResourceAttr(resourceName, "stack_type", "dual-stack"),
					resource.TestCheckResourceAttr(resourceName, "ipv4_cidr_block", "10.0.1.0/24"),
					resource.TestCheckResourceAttr(resourceName, "zone", "a"),
					resource.TestCheckResourceAttrSet(resourceName, "subnet_id"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
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

func TestAccEvrocSubnet_IPv6Only(t *testing.T) {
	resourceName := "evroc_subnet.test"
	ts := time.Now().Unix()
	vpcName := fmt.Sprintf("tf-test-vpc-%d", ts)
	subnetName := fmt.Sprintf("tf-test-subnet-%d", ts)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckEvrocSubnetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEvrocSubnetConfig_ipv6Only(vpcName, subnetName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEvrocSubnetExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", subnetName),
					resource.TestCheckResourceAttr(resourceName, "stack_type", "ipv6-only"),
					resource.TestCheckResourceAttr(resourceName, "zone", "a"),
					resource.TestCheckResourceAttrSet(resourceName, "subnet_id"),
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

func testAccCheckEvrocSubnetExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no subnet ID is set")
		}

		config := testAccProvider.Meta().(*ProviderConfig)
		_, err := config.Client.Networking().Subnets().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("subnet %s not found: %w", rs.Primary.ID, err)
		}

		return nil
	}
}

func testAccCheckEvrocSubnetDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*ProviderConfig)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "evroc_subnet" {
			continue
		}

		_, err := config.Client.Networking().Subnets().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			continue
		}

		return fmt.Errorf("subnet %s still exists", rs.Primary.ID)
	}

	return nil
}

func testAccEvrocSubnetConfig_dualStack(vpcName, subnetName string) string {
	return fmt.Sprintf(`
resource "evroc_vpc" "test" {
  name             = "%s"
  stack_type       = "dual-stack"
  ipv4_cidr_blocks = ["10.0.0.0/16"]
}

resource "evroc_subnet" "test" {
  name            = "%s"
  vpc_ref         = evroc_vpc.test.fqid
  ipv4_cidr_block = "10.0.1.0/24"
  stack_type      = "dual-stack"
  zone            = "a"
}
`, vpcName, subnetName)
}

func TestAccEvrocSubnet_DualStackWithLabels(t *testing.T) {
	resourceName := "evroc_subnet.test"
	ts := time.Now().Unix()
	vpcName := fmt.Sprintf("tf-test-vpc-%d", ts)
	subnetName := fmt.Sprintf("tf-test-subnet-%d", ts)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckEvrocSubnetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEvrocSubnetConfig_dualStackWithLabels(vpcName, subnetName, "v1"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEvrocSubnetExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", subnetName),
					resource.TestCheckResourceAttr(resourceName, "stack_type", "dual-stack"),
					resource.TestCheckResourceAttr(resourceName, "user_labels.env", "test"),
					resource.TestCheckResourceAttr(resourceName, "user_labels.version", "v1"),
				),
			},
			{
				Config: testAccEvrocSubnetConfig_dualStackWithLabels(vpcName, subnetName, "v2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEvrocSubnetExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "user_labels.version", "v2"),
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

func TestAccEvrocSubnet_DataSource(t *testing.T) {
	ts := time.Now().Unix()
	vpcName := fmt.Sprintf("tf-test-vpc-%d", ts)
	subnetName := fmt.Sprintf("tf-test-subnet-%d", ts)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckEvrocSubnetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEvrocSubnetConfig_dataSource(vpcName, subnetName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.evroc_subnet.test", "name", subnetName),
					resource.TestCheckResourceAttr("data.evroc_subnet.test", "stack_type", "dual-stack"),
					resource.TestCheckResourceAttrSet("data.evroc_subnet.test", "subnet_id"),
					resource.TestCheckResourceAttrSet("data.evroc_subnet.test", "fqid"),
					resource.TestCheckResourceAttr("data.evroc_subnet.test", "zone", "a"),
				),
			},
		},
	})
}

func testAccEvrocSubnetConfig_dualStackWithLabels(vpcName, subnetName, version string) string {
	return fmt.Sprintf(`
resource "evroc_vpc" "test" {
  name             = "%s"
  stack_type       = "dual-stack"
  ipv4_cidr_blocks = ["10.0.0.0/16"]
}

resource "evroc_subnet" "test" {
  name            = "%s"
  vpc_ref         = evroc_vpc.test.fqid
  ipv4_cidr_block = "10.0.1.0/24"
  stack_type      = "dual-stack"
  zone            = "a"

  user_labels = {
    env     = "test"
    version = "%s"
  }
}
`, vpcName, subnetName, version)
}

func testAccEvrocSubnetConfig_dataSource(vpcName, subnetName string) string {
	return fmt.Sprintf(`
resource "evroc_vpc" "test" {
  name             = "%s"
  stack_type       = "dual-stack"
  ipv4_cidr_blocks = ["10.0.0.0/16"]
}

resource "evroc_subnet" "test" {
  name            = "%s"
  vpc_ref         = evroc_vpc.test.fqid
  ipv4_cidr_block = "10.0.1.0/24"
  stack_type      = "dual-stack"
  zone            = "a"
}

data "evroc_subnet" "test" {
  name = evroc_subnet.test.name
}
`, vpcName, subnetName)
}

func testAccEvrocSubnetConfig_ipv6Only(vpcName, subnetName string) string {
	return fmt.Sprintf(`
resource "evroc_vpc" "test" {
  name       = "%s"
  stack_type = "ipv6-only"
}

resource "evroc_subnet" "test" {
  name       = "%s"
  vpc_ref    = evroc_vpc.test.fqid
  stack_type = "ipv6-only"
  zone       = "a"
}
`, vpcName, subnetName)
}
