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

func TestAccEvrocServiceAccountCredential_Basic(t *testing.T) {
	resourceName := "evroc_service_account_credential.test"
	saName := fmt.Sprintf("tf-test-sa-%d", time.Now().Unix())
	credName := fmt.Sprintf("tf-test-cred-%d", time.Now().Unix())
	expiresAt := time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckEvrocServiceAccountCredentialDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEvrocServiceAccountCredentialConfig_basic(saName, credName, expiresAt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEvrocServiceAccountCredentialExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", credName),
					resource.TestCheckResourceAttr(resourceName, "description", "Acceptance test credential"),
					resource.TestCheckResourceAttr(resourceName, "access_token_lifetime", "3600"),
					resource.TestCheckResourceAttrSet(resourceName, "credential_id"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckResourceAttrSet(resourceName, "private_key_jwk"),
					resource.TestCheckResourceAttrSet(resourceName, "service_account_ref"),
				),
			},
		},
	})
}

func testAccCheckEvrocServiceAccountCredentialExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no credential ID is set")
		}

		config := testAccProvider.Meta().(*ProviderConfig)
		saRef := rs.Primary.Attributes["service_account_ref"]
		saID := serviceAccountIDFromRef(saRef)
		_, err := config.Client.IAM().ServiceAccountCredentials(saID).Get(context.Background(), rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("service account credential %s not found: %w", rs.Primary.ID, err)
		}

		return nil
	}
}

func testAccCheckEvrocServiceAccountCredentialDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*ProviderConfig)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "evroc_service_account_credential" {
			continue
		}

		saRef := rs.Primary.Attributes["service_account_ref"]
		saID := serviceAccountIDFromRef(saRef)
		_, err := config.Client.IAM().ServiceAccountCredentials(saID).Get(context.Background(), rs.Primary.ID)
		if err != nil {
			continue
		}

		return fmt.Errorf("service account credential %s still exists", rs.Primary.ID)
	}

	return nil
}

func testAccEvrocServiceAccountCredentialConfig_basic(saName, credName, expiresAt string) string {
	return fmt.Sprintf(`
resource "evroc_service_account" "test" {
  name        = "%s"
  description = "Parent SA for credential test"
  enabled     = true
}

resource "evroc_service_account_credential" "test" {
  name                  = "%s"
  service_account_ref   = evroc_service_account.test.fqid
  description           = "Acceptance test credential"
  expires_at            = "%s"
  access_token_lifetime = 3600
}
`, saName, credName, expiresAt)
}
