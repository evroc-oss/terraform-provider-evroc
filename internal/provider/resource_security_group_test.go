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

func TestAccEvrocSecurityGroup_Basic(t *testing.T) {
	resourceName := "evroc_security_group.test"
	sgName := fmt.Sprintf("tf-test-sg-%d", time.Now().Unix())

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckEvrocSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEvrocSecurityGroupConfig_basic(sgName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEvrocSecurityGroupExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", sgName),
					resource.TestCheckResourceAttr(resourceName, "rule.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "sg_id"),
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

func testAccCheckEvrocSecurityGroupExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no security group ID is set")
		}

		config := testAccProvider.Meta().(*ProviderConfig)
		_, err := config.Client.Networking().SecurityGroups().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("security group %s not found: %w", rs.Primary.ID, err)
		}

		return nil
	}
}

func testAccCheckEvrocSecurityGroupDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*ProviderConfig)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "evroc_security_group" {
			continue
		}

		// Retry to handle eventual consistency after delete
		var lastErr error
		for i := 0; i < 10; i++ {
			_, err := config.Client.Networking().SecurityGroups().Get(context.Background(), rs.Primary.ID)
			if err != nil {
				lastErr = nil
				break
			}
			lastErr = fmt.Errorf("security group %s still exists", rs.Primary.ID)
			time.Sleep(3 * time.Second)
		}
		if lastErr != nil {
			return lastErr
		}
	}

	return nil
}

func TestAccEvrocSecurityGroup_IPv6Rules(t *testing.T) {
	resourceName := "evroc_security_group.test"
	sgName := fmt.Sprintf("tf-test-sg-%d", time.Now().Unix())

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckEvrocSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEvrocSecurityGroupConfig_ipv6(sgName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEvrocSecurityGroupExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", sgName),
					resource.TestCheckResourceAttr(resourceName, "rule.#", "3"),
					resource.TestCheckResourceAttrSet(resourceName, "sg_id"),
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

func testAccEvrocSecurityGroupConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "evroc_security_group" "test" {
  name = "%s"

  rule {
    name      = "allow-ssh"
    direction = "Ingress"
    protocol  = "TCP"
    port      = 22
    remote_ip = "0.0.0.0/0"
  }
}
`, name)
}

func TestAccEvrocSecurityGroup_IPv6InVPC(t *testing.T) {
	resourceName := "evroc_security_group.test"
	ts := time.Now().Unix()
	sgName := fmt.Sprintf("tf-test-sg-%d", ts)
	vpcName := fmt.Sprintf("tf-test-vpc-%d", ts)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckEvrocSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEvrocSecurityGroupConfig_ipv6InVPC(vpcName, sgName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEvrocSecurityGroupExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", sgName),
					resource.TestCheckResourceAttr(resourceName, "rule.#", "2"),
					resource.TestCheckResourceAttrSet(resourceName, "vpc_ref"),
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

func testAccEvrocSecurityGroupConfig_ipv6(name string) string {
	return fmt.Sprintf(`
resource "evroc_security_group" "test" {
  name = "%s"

  rule {
    name      = "allow-ssh-ipv4"
    direction = "Ingress"
    protocol  = "TCP"
    port      = 22
    remote_ip = "0.0.0.0/0"
  }

  rule {
    name      = "allow-ssh-ipv6"
    direction = "Ingress"
    protocol  = "TCP"
    port      = 22
    remote_ip = "::/0"
  }

  rule {
    name      = "allow-http-ipv6"
    direction = "Ingress"
    protocol  = "TCP"
    port      = 80
    remote_ip = "2001:db8::/32"
  }
}
`, name)
}

func testAccEvrocSecurityGroupConfig_ipv6InVPC(vpcName, sgName string) string {
	return fmt.Sprintf(`
resource "evroc_vpc" "test" {
  name             = "%s"
  stack_type       = "dual-stack"
  ipv4_cidr_blocks = ["10.0.0.0/16"]
}

resource "evroc_security_group" "test" {
  name    = "%s"
  vpc_ref = evroc_vpc.test.fqid

  rule {
    name      = "allow-ssh-ipv6"
    direction = "Ingress"
    protocol  = "TCP"
    port      = 22
    remote_ip = "::/0"
  }

  rule {
    name      = "allow-egress-ipv6"
    direction = "Egress"
    protocol  = "All"
    remote_ip = "::/0"
  }
}
`, vpcName, sgName)
}
