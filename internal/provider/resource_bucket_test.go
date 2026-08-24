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

func TestAccEvrocBucket_Basic(t *testing.T) {
	resourceName := "evroc_bucket.test"
	bucketName := fmt.Sprintf("tf-test-bucket-%d", time.Now().Unix())

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckEvrocBucketDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEvrocBucketConfig_basic(bucketName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEvrocBucketExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", bucketName),
					resource.TestCheckResourceAttr(resourceName, "object_retention_mode", "Disabled"),
					resource.TestCheckResourceAttrSet(resourceName, "bucket_id"),
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

func testAccCheckEvrocBucketExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no bucket ID is set")
		}

		config := testAccProvider.Meta().(*ProviderConfig)
		_, err := config.Client.Storage().Buckets().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("bucket %s not found: %w", rs.Primary.ID, err)
		}

		return nil
	}
}

func testAccCheckEvrocBucketDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*ProviderConfig)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "evroc_bucket" {
			continue
		}

		// Check if bucket still exists
		_, err := config.Client.Storage().Buckets().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			// Bucket not found - successfully deleted
			continue
		}

		// Bucket still exists
		return fmt.Errorf("bucket %s still exists", rs.Primary.ID)
	}

	return nil
}

func testAccEvrocBucketConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "evroc_bucket" "test" {
  name = "%s"
}
`, name)
}

func TestAccEvrocBucket_LifecycleRules(t *testing.T) {
	resourceName := "evroc_bucket.test"
	bucketName := fmt.Sprintf("tf-test-bucket-lc-%d", time.Now().Unix())

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckEvrocBucketDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEvrocBucketConfig_lifecycle(bucketName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEvrocBucketExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "lifecycle_rule.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "lifecycle_rule.0.id", "expire-tmp"),
					resource.TestCheckResourceAttr(resourceName, "lifecycle_rule.0.expire_current_version.0.days", "30"),
					resource.TestCheckResourceAttr(resourceName, "lifecycle_rule.0.filter.0.prefix", "tmp/"),
					resource.TestCheckResourceAttr(resourceName, "lifecycle_rule.1.id", "cleanup-multipart"),
					resource.TestCheckResourceAttr(resourceName, "lifecycle_rule.1.abort_incomplete_multipart.0.days", "7"),
				),
			},
			{
				// Update: drop the second rule and change the first
				Config: testAccEvrocBucketConfig_lifecycleUpdated(bucketName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEvrocBucketExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "lifecycle_rule.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "lifecycle_rule.0.expire_current_version.0.days", "60"),
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

func testAccEvrocBucketConfig_lifecycle(name string) string {
	return fmt.Sprintf(`
resource "evroc_bucket" "test" {
  name = "%s"

  lifecycle_rule {
    id = "expire-tmp"

    expire_current_version {
      days = 30
    }

    filter {
      prefix = "tmp/"
    }
  }

  lifecycle_rule {
    id = "cleanup-multipart"

    abort_incomplete_multipart {
      days = 7
    }
  }
}
`, name)
}

func testAccEvrocBucketConfig_lifecycleUpdated(name string) string {
	return fmt.Sprintf(`
resource "evroc_bucket" "test" {
  name = "%s"

  lifecycle_rule {
    id = "expire-tmp"

    expire_current_version {
      days = 60
    }

    filter {
      prefix = "tmp/"
    }
  }
}
`, name)
}
