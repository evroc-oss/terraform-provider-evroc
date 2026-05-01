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

func TestAccEvrocBucketServiceAccount_Basic(t *testing.T) {
	resourceName := "evroc_bucket_service_account.test"
	bucketName := fmt.Sprintf("tf-test-bucket-%d", time.Now().Unix())
	saName := fmt.Sprintf("tf-test-sa-%d", time.Now().Unix())

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckEvrocBucketServiceAccountDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEvrocBucketServiceAccountConfig_basic(bucketName, saName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEvrocBucketServiceAccountExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", saName),
					resource.TestCheckResourceAttr(resourceName, "buckets.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "buckets.0", bucketName),
					resource.TestCheckResourceAttrSet(resourceName, "service_account_id"),
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

func testAccCheckEvrocBucketServiceAccountExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no bucket service account ID is set")
		}

		config := testAccProvider.Meta().(*ProviderConfig)
		_, err := config.Client.Storage().BucketServiceAccounts().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("bucket service account %s not found: %w", rs.Primary.ID, err)
		}

		return nil
	}
}

func testAccCheckEvrocBucketServiceAccountDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*ProviderConfig)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "evroc_bucket_service_account" {
			continue
		}

		// Check if bucket service account still exists
		_, err := config.Client.Storage().BucketServiceAccounts().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			// Service account not found - successfully deleted
			continue
		}

		// Service account still exists
		return fmt.Errorf("bucket service account %s still exists", rs.Primary.ID)
	}

	return nil
}

func testAccEvrocBucketServiceAccountConfig_basic(bucketName, saName string) string {
	return fmt.Sprintf(`
resource "evroc_bucket" "test" {
  name = "%s"
}

resource "evroc_bucket_service_account" "test" {
  name    = "%s"
  buckets = [evroc_bucket.test.name]
}
`, bucketName, saName)
}
