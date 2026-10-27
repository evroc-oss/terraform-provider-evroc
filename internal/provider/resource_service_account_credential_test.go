// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	iamtypes "github.com/evroc-oss/evroc-go-sdk/types/iam"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	sdkterraform "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
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

func TestAccEvrocServiceAccountCredential_HMAC(t *testing.T) {
	suffix := time.Now().UnixNano()
	sa := fmt.Sprintf("tf-hmac-sa-%d", suffix)
	name := fmt.Sprintf("tf-hmac-%d", suffix)
	expires := time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339)
	config := fmt.Sprintf(`
resource "evroc_service_account" "test" { name = %q }
resource "evroc_service_account_credential" "test" {
 name = %q
 service_account_ref = evroc_service_account.test.fqid
 type = "hmac-sigv4"
 expires_at = %q
}
data "evroc_service_account_credential" "test" {
 name = evroc_service_account_credential.test.name
 service_account_ref = evroc_service_account.test.fqid
}
`, sa, name, expires)
	const address = "evroc_service_account_credential.test"
	var secret string
	checkSecret := func(s *terraform.State) error {
		r := s.RootModule().Resources[address]
		if r == nil {
			return fmt.Errorf("credential missing from state")
		}
		value := r.Primary.Attributes["secret_access_key"]
		if value == "" {
			return fmt.Errorf("HMAC secret missing")
		}
		if secret != "" && secret != value {
			return fmt.Errorf("refresh changed the one-time HMAC secret")
		}
		secret = value
		return nil
	}
	resource.Test(t, resource.TestCase{PreventPostDestroyRefresh: true, PreCheck: func() { testAccPreCheck(t) }, ProviderFactories: testAccProviderFactories,
		CheckDestroy: testAccCheckEvrocServiceAccountCredentialDestroy,
		Steps: []resource.TestStep{
			{Config: config, Check: resource.ComposeTestCheckFunc(checkSecret,
				resource.TestCheckResourceAttr(address, "type", "hmac-sigv4"),
				resource.TestCheckResourceAttrSet(address, "access_key_id"),
				resource.TestCheckResourceAttrPair(address, "access_key_id", "data.evroc_service_account_credential.test", "access_key_id"),
				resource.TestCheckResourceAttr("data.evroc_service_account_credential.test", "type", "hmac-sigv4"),
			)},
			{RefreshState: true, Check: checkSecret},
		},
	})
}

func TestHMACCredential(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	cred := mockServiceAccountCredential("test-cred")
	cred.Spec.Type = iamtypes.HmacSigv4
	cred.Spec.Rs256Jwt = nil
	cred.Spec.HmacSigv4 = &iamtypes.ServiceaccountcredentialSpecHmacSigv4{AccessKeyId: "access-id"}
	cred.Status.PrivateKeyJwk = nil
	prefix := "/iam/v1beta2/projects/test-project/serviceAccounts/test-sa/serviceAccountCredentials"
	ms.mux.HandleFunc(prefix, func(w http.ResponseWriter, r *http.Request) {
		var req iamtypes.ServiceaccountcredentialRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		if req.Spec.Type != iamtypes.HmacSigv4 || req.Spec.Rs256Jwt != nil {
			t.Errorf("unexpected HMAC request: %+v", req.Spec)
		}
		created := *cred
		secret := "one-time-secret"
		created.Status.SecretAccessKey = &secret
		respondJSON(w, 201, created)
	})
	ms.mux.HandleFunc(prefix+"/test-cred", func(w http.ResponseWriter, r *http.Request) { respondJSON(w, 200, cred) })
	res := resourceServiceAccountCredential()
	raw := map[string]interface{}{"name": "test-cred", "service_account_ref": "/iam/projects/test-project/serviceAccounts/test-sa", "expires_at": "2027-01-01T00:00:00Z", "type": "hmac-sigv4"}
	d := schema.TestResourceDataRaw(t, res.Schema, raw)
	cfg := newTestProviderConfig(t, ms.server.URL)
	if diags := res.CreateContext(context.Background(), d, cfg); diags.HasError() {
		t.Fatal(diags)
	}
	assertField(t, d, "secret_access_key", "one-time-secret")
	assertField(t, d, "access_key_id", "access-id")
	if diags := res.ReadContext(context.Background(), d, cfg); diags.HasError() {
		t.Fatal(diags)
	}
	assertField(t, d, "secret_access_key", "one-time-secret")
	if !res.Schema["secret_access_key"].Sensitive {
		t.Fatal("HMAC secret must be sensitive")
	}
	raw["access_token_lifetime"] = 300
	if _, err := res.Diff(context.Background(), nil, sdkterraform.NewResourceConfigRaw(raw), nil); err == nil {
		t.Fatal("HMAC cannot configure JWT token lifetime")
	}
}
