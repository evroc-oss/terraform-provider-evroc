// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccEvrocBucketServiceAccountSecret(t *testing.T) {
	ts := time.Now().Unix()
	bucketName := fmt.Sprintf("tf-test-bucket-%d", ts)
	saName := fmt.Sprintf("tf-test-bsa-%d", ts)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEvrocBucketServiceAccountSecretConfig(bucketName, saName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.evroc_bucket_service_account_secret.test", "access_key_id"),
					resource.TestCheckResourceAttrSet("data.evroc_bucket_service_account_secret.test", "secret_access_key"),
				),
			},
		},
	})
}

func testAccEvrocBucketServiceAccountSecretConfig(bucketName, saName string) string {
	return fmt.Sprintf(`
resource "evroc_bucket" "test" {
  name = "%s"
}

resource "evroc_bucket_service_account" "test" {
  name    = "%s"
  buckets = [evroc_bucket.test.name]
}

data "evroc_bucket_service_account_secret" "test" {
  name = evroc_bucket_service_account.test.credentials_secret
}
`, bucketName, saName)
}
