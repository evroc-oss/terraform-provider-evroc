// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// ============================================================================
// Snapshot CRUD Tests
// ============================================================================

func TestResourceSnapshotRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupSnapshotHandlers(ms, "test-snap")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceSnapshot()
	d := newTestResourceData(t, res)
	d.SetId("test-snap")

	ctx := context.Background()
	diags := resourceSnapshotRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	assertField(t, d, "name", "test-snap")
	assertField(t, d, "region", "se-sto")
	if d.Get("restore_size").(int) != 20 {
		t.Errorf("expected restore_size=20, got %v", d.Get("restore_size"))
	}
}

func TestResourceSnapshotDelete(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupSnapshotHandlers(ms, "test-snap")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceSnapshot()
	d := newTestResourceData(t, res)
	d.SetId("test-snap")
	d.Set("name", "test-snap")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	diags := resourceSnapshotDelete(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected delete error: %v", diagnosticsToString(diags))
	}
}

// ============================================================================
// VPC CRUD Tests
// ============================================================================

func TestResourceVPCRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupVPCHandlers(ms, "test-vpc")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceVPC()
	d := newTestResourceData(t, res)
	d.SetId("test-vpc")

	ctx := context.Background()
	diags := resourceVPCRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	assertField(t, d, "name", "test-vpc")
	assertField(t, d, "region", "se-sto")
	assertField(t, d, "stack_type", "dual-stack")
}

func TestResourceVPCDelete(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupVPCHandlers(ms, "test-vpc")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceVPC()
	d := newTestResourceData(t, res)
	d.SetId("test-vpc")

	ctx := context.Background()
	diags := resourceVPCDelete(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected delete error: %v", diagnosticsToString(diags))
	}
}

// ============================================================================
// Subnet CRUD Tests
// ============================================================================

func TestResourceSubnetRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupSubnetHandlers(ms, "test-subnet")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceSubnet()
	d := newTestResourceData(t, res)
	d.SetId("test-subnet")

	ctx := context.Background()
	diags := resourceSubnetRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	assertField(t, d, "name", "test-subnet")
	assertField(t, d, "region", "se-sto")
	assertField(t, d, "stack_type", "dual-stack")
	assertField(t, d, "zone", "a")
}

func TestResourceSubnetDelete(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupSubnetHandlers(ms, "test-subnet")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceSubnet()
	d := newTestResourceData(t, res)
	d.SetId("test-subnet")

	ctx := context.Background()
	diags := resourceSubnetDelete(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected delete error: %v", diagnosticsToString(diags))
	}
}

// ============================================================================
// LoadBalancer CRUD Tests
// ============================================================================

func TestResourceLoadBalancerRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupLoadBalancerHandlers(ms, "test-lb")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceLoadBalancer()
	d := newTestResourceData(t, res)
	d.SetId("test-lb")

	ctx := context.Background()
	diags := resourceLoadBalancerRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	assertField(t, d, "name", "test-lb")
	assertField(t, d, "region", "se-sto")
	assertField(t, d, "public_ipv4_address", "203.0.113.10")
}

func TestResourceLoadBalancerDelete(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupLoadBalancerHandlers(ms, "test-lb")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceLoadBalancer()
	d := newTestResourceData(t, res)
	d.SetId("test-lb")
	d.Set("name", "test-lb")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	diags := resourceLoadBalancerDelete(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected delete error: %v", diagnosticsToString(diags))
	}
}

// ============================================================================
// Backend Pool CRUD Tests
// ============================================================================

func TestResourceBackendPoolRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupBackendPoolHandlers(ms, "test-pool")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceLBBackendPool()
	d := newTestResourceData(t, res)
	d.SetId("test-pool")

	ctx := context.Background()
	diags := resourceLBBackendPoolRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	assertField(t, d, "name", "test-pool")
	assertField(t, d, "region", "se-sto")
}

func TestResourceBackendPoolDelete(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupBackendPoolHandlers(ms, "test-pool")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceLBBackendPool()
	d := newTestResourceData(t, res)
	d.SetId("test-pool")
	d.Set("name", "test-pool")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	diags := resourceLBBackendPoolDelete(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected delete error: %v", diagnosticsToString(diags))
	}
}

// ============================================================================
// Backend Service CRUD Tests
// ============================================================================

func TestResourceBackendServiceRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupBackendServiceHandlers(ms, "test-svc")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceLBBackendService()
	d := newTestResourceData(t, res)
	d.SetId("test-svc")

	ctx := context.Background()
	diags := resourceLBBackendServiceRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	assertField(t, d, "name", "test-svc")
	assertField(t, d, "port", 80)
	backends := d.Get("backends").([]interface{})
	if len(backends) != 1 {
		t.Errorf("expected 1 backend, got %d", len(backends))
	}
	assertField(t, d, "ip_protocol_selection", "IPv4")
}

func TestResourceBackendServiceDelete(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupBackendServiceHandlers(ms, "test-svc")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceLBBackendService()
	d := newTestResourceData(t, res)
	d.SetId("test-svc")
	d.Set("name", "test-svc")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	diags := resourceLBBackendServiceDelete(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected delete error: %v", diagnosticsToString(diags))
	}
}

// ============================================================================
// L4 Route CRUD Tests
// ============================================================================

func TestResourceL4RouteRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupL4RouteHandlers(ms, "test-route")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceLBL4Route()
	d := newTestResourceData(t, res)
	d.SetId("test-route")

	ctx := context.Background()
	diags := resourceLBL4RouteRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	assertField(t, d, "name", "test-route")
	assertField(t, d, "region", "se-sto")
}

func TestResourceL4RouteDelete(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupL4RouteHandlers(ms, "test-route")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceLBL4Route()
	d := newTestResourceData(t, res)
	d.SetId("test-route")
	d.Set("name", "test-route")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	diags := resourceLBL4RouteDelete(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected delete error: %v", diagnosticsToString(diags))
	}
}

// ============================================================================
// Service Account CRUD Tests
// ============================================================================

func TestResourceServiceAccountCreate(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupServiceAccountHandlers(ms, "test-sa")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceServiceAccount()
	d := newTestResourceData(t, res)
	d.Set("name", "test-sa")
	d.Set("enabled", true)
	d.Set("description", "Test service account")

	ctx := context.Background()
	diags := resourceServiceAccountCreate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected create error: %v", diagnosticsToString(diags))
	}
	if d.Id() == "" {
		t.Error("expected ID to be set after create")
	}
}

func TestResourceServiceAccountRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupServiceAccountHandlers(ms, "test-sa")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceServiceAccount()
	d := newTestResourceData(t, res)
	d.SetId("test-sa")

	ctx := context.Background()
	diags := resourceServiceAccountRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	assertField(t, d, "name", "test-sa")
	assertField(t, d, "enabled", true)
	assertField(t, d, "description", "Test service account")
	assertField(t, d, "oauth_client_id", "oauth-client-id-123")
	assertField(t, d, "fqid", "/iam/projects/test-project/serviceAccounts/test-sa")
}

func TestResourceServiceAccountReadNotFound(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceServiceAccount()
	d := newTestResourceData(t, res)
	d.SetId("nonexistent-sa")

	ctx := context.Background()
	diags := resourceServiceAccountRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("expected no error for not found, got: %v", diagnosticsToString(diags))
	}
	if d.Id() != "" {
		t.Errorf("expected ID to be cleared on not-found, got %q", d.Id())
	}
}

func TestResourceServiceAccountUpdate(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupServiceAccountHandlers(ms, "test-sa")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceServiceAccount()

	d := newResourceDataWithDiff(t, res, "test-sa",
		map[string]string{
			"name":        "test-sa",
			"enabled":     "true",
			"description": "Old description",
		},
		map[string]*terraform.ResourceAttrDiff{
			"description": {Old: "Old description", New: "Updated description"},
		},
	)

	ctx := context.Background()
	diags := resourceServiceAccountUpdate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected update error: %v", diagnosticsToString(diags))
	}
}

func TestResourceServiceAccountDelete(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupServiceAccountHandlers(ms, "test-sa")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceServiceAccount()
	d := newTestResourceData(t, res)
	d.SetId("test-sa")
	d.Set("name", "test-sa")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	diags := resourceServiceAccountDelete(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected delete error: %v", diagnosticsToString(diags))
	}
}

// ============================================================================
// Service Account Credential CRUD Tests
// ============================================================================

func TestResourceServiceAccountCredentialCreate(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupServiceAccountCredentialHandlers(ms, "test-cred")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceServiceAccountCredential()
	d := newTestResourceData(t, res)
	d.Set("name", "test-cred")
	d.Set("service_account_ref", "/iam/projects/test-project/serviceAccounts/test-sa")
	d.Set("expires_at", time.Now().Add(24*time.Hour).Format(time.RFC3339))
	d.Set("access_token_lifetime", 3600)

	ctx := context.Background()
	diags := resourceServiceAccountCredentialCreate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected create error: %v", diagnosticsToString(diags))
	}
	if d.Id() == "" {
		t.Error("expected ID to be set after create")
	}
	if d.Get("private_key_jwk").(string) == "" {
		t.Error("expected private_key_jwk to be set after create")
	}
}

func TestResourceServiceAccountCredentialRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupServiceAccountCredentialHandlers(ms, "test-cred")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceServiceAccountCredential()
	d := newTestResourceData(t, res)
	d.SetId("test-cred")
	d.Set("service_account_ref", "/iam/projects/test-project/serviceAccounts/test-sa")

	ctx := context.Background()
	diags := resourceServiceAccountCredentialRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	assertField(t, d, "name", "test-cred")
	assertField(t, d, "service_account_ref", "/iam/projects/test-project/serviceAccounts/test-sa")
	assertField(t, d, "description", "Test credential")
	assertField(t, d, "access_token_lifetime", 3600)
}

func TestResourceServiceAccountCredentialReadNotFound(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceServiceAccountCredential()
	d := newTestResourceData(t, res)
	d.SetId("nonexistent-cred")
	d.Set("service_account_ref", "/iam/projects/test-project/serviceAccounts/test-sa")

	ctx := context.Background()
	diags := resourceServiceAccountCredentialRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("expected no error for not found, got: %v", diagnosticsToString(diags))
	}
	if d.Id() != "" {
		t.Errorf("expected ID to be cleared on not-found, got %q", d.Id())
	}
}

func TestResourceServiceAccountCredentialDelete(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupServiceAccountCredentialHandlers(ms, "test-cred")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceServiceAccountCredential()
	d := newTestResourceData(t, res)
	d.SetId("test-cred")
	d.Set("name", "test-cred")
	d.Set("service_account_ref", "/iam/projects/test-project/serviceAccounts/test-sa")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	diags := resourceServiceAccountCredentialDelete(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected delete error: %v", diagnosticsToString(diags))
	}
}

// ============================================================================
// Service Account Data Source Tests
// ============================================================================

func TestDataSourceServiceAccountRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupServiceAccountHandlers(ms, "test-sa")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := dataSourceServiceAccount()
	d := res.TestResourceData()
	d.Set("name", "test-sa")

	ctx := context.Background()
	diags := dataSourceServiceAccountRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	assertField(t, d, "name", "test-sa")
	assertField(t, d, "enabled", true)
	assertField(t, d, "description", "Test service account")
	assertField(t, d, "oauth_client_id", "oauth-client-id-123")
	assertField(t, d, "fqid", "/iam/projects/test-project/serviceAccounts/test-sa")
}

func TestDataSourceServiceAccountReadNotFound(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := dataSourceServiceAccount()
	d := res.TestResourceData()
	d.Set("name", "nonexistent-sa")

	ctx := context.Background()
	diags := dataSourceServiceAccountRead(ctx, d, config)
	if !diags.HasError() {
		t.Fatal("expected error for not found data source read")
	}
}

// ============================================================================
// Service Account Credential Data Source Tests
// ============================================================================

func TestDataSourceServiceAccountCredentialRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupServiceAccountCredentialHandlers(ms, "test-cred")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := dataSourceServiceAccountCredential()
	d := res.TestResourceData()
	d.Set("name", "test-cred")
	d.Set("service_account_ref", "/iam/projects/test-project/serviceAccounts/test-sa")

	ctx := context.Background()
	diags := dataSourceServiceAccountCredentialRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	assertField(t, d, "name", "test-cred")
	assertField(t, d, "service_account_ref", "/iam/projects/test-project/serviceAccounts/test-sa")
	assertField(t, d, "description", "Test credential")
	assertField(t, d, "access_token_lifetime", 3600)
}

func TestDataSourceServiceAccountCredentialReadNotFound(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := dataSourceServiceAccountCredential()
	d := res.TestResourceData()
	d.Set("name", "nonexistent-cred")
	d.Set("service_account_ref", "/iam/projects/test-project/serviceAccounts/test-sa")

	ctx := context.Background()
	diags := dataSourceServiceAccountCredentialRead(ctx, d, config)
	if !diags.HasError() {
		t.Fatal("expected error for not found data source read")
	}
}

// ============================================================================
// Role Binding CRUD Tests
// ============================================================================

func TestResourceRoleBindingCreate(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupRoleBindingHandlers(ms)
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceRoleBinding()
	d := newTestResourceData(t, res)
	d.Set("name", "rb-test")
	d.Set("principal", "/iam/projects/test-project/serviceAccounts/test-sa")
	d.Set("roles", []interface{}{
		map[string]interface{}{
			"role":      "/iam/roles/computeOperator",
			"resources": []interface{}{},
		},
	})

	ctx := context.Background()
	diags := resourceRoleBindingCreate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected create error: %v", diagnosticsToString(diags))
	}
	if d.Id() == "" {
		t.Error("expected ID to be set after create")
	}
	assertField(t, d, "principal", "/iam/projects/test-project/serviceAccounts/test-sa")
}

func TestResourceRoleBindingRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupRoleBindingHandlers(ms)
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceRoleBinding()
	d := newTestResourceData(t, res)
	d.SetId("rb-test")

	ctx := context.Background()
	diags := resourceRoleBindingRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	assertField(t, d, "principal", "/iam/projects/test-project/serviceAccounts/test-sa")
	assertField(t, d, "uid", "00000000-0000-0000-0000-000000000123")
}

func TestResourceRoleBindingDelete(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupRoleBindingHandlers(ms)
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceRoleBinding()
	d := newTestResourceData(t, res)
	d.SetId("rb-test")

	ctx := context.Background()
	diags := resourceRoleBindingDelete(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected delete error: %v", diagnosticsToString(diags))
	}
}

// ============================================================================
// Org Role Binding CRUD Tests
// ============================================================================

func TestResourceOrgRoleBindingCreate(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupRoleBindingHandlers(ms)
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceOrgRoleBinding()
	d := newTestResourceData(t, res)
	d.Set("name", "org-rb-test")
	d.Set("principal", "/iam/users/00000000-0000-0000-0000-000000000789")
	d.Set("roles", []interface{}{
		map[string]interface{}{
			"role":      "/iam/roles/organizationViewer",
			"resources": []interface{}{},
		},
	})

	ctx := context.Background()
	diags := resourceOrgRoleBindingCreate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected create error: %v", diagnosticsToString(diags))
	}
	if d.Id() == "" {
		t.Error("expected ID to be set after create")
	}
	assertField(t, d, "principal", "/iam/users/00000000-0000-0000-0000-000000000789")
}

func TestResourceOrgRoleBindingRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupRoleBindingHandlers(ms)
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceOrgRoleBinding()
	d := newTestResourceData(t, res)
	d.SetId("org-rb-test")

	ctx := context.Background()
	diags := resourceOrgRoleBindingRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	assertField(t, d, "principal", "/iam/users/00000000-0000-0000-0000-000000000789")
	assertField(t, d, "uid", "00000000-0000-0000-0000-000000000456")
}

func TestResourceOrgRoleBindingDelete(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupRoleBindingHandlers(ms)
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceOrgRoleBinding()
	d := newTestResourceData(t, res)
	d.SetId("org-rb-test")

	ctx := context.Background()
	diags := resourceOrgRoleBindingDelete(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected delete error: %v", diagnosticsToString(diags))
	}
}

// ============================================================================
// Roles Data Source Tests
// ============================================================================

func TestDataSourceRolesRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupRoleBindingHandlers(ms)
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := dataSourceRoles()
	d := res.TestResourceData()

	ctx := context.Background()
	diags := dataSourceRolesRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	roles := d.Get("roles").([]interface{})
	if len(roles) != 4 {
		t.Fatalf("expected 4 roles, got %d", len(roles))
	}
	first := roles[0].(map[string]interface{})
	if first["id"] != "/iam/roles/computeOperator" {
		t.Errorf("expected first role id /iam/roles/computeOperator, got %v", first["id"])
	}
}
