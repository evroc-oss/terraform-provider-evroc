// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// ============================================================================
// Disk CRUD Tests
// ============================================================================

func assertField(t *testing.T, d *schema.ResourceData, key string, expected interface{}) {
	t.Helper()
	got := d.Get(key)
	if got != expected {
		t.Errorf("expected %s=%v, got %v", key, expected, got)
	}
}

// newResourceDataWithDiff creates a ResourceData with old state and a diff so that HasChange() works.
// oldAttrs is the prior state (flat key=value map), diffAttrs maps changed keys to their new values.
func newResourceDataWithDiff(t *testing.T, res *schema.Resource, id string, oldAttrs map[string]string, diffAttrs map[string]*terraform.ResourceAttrDiff) *schema.ResourceData {
	t.Helper()
	state := &terraform.InstanceState{
		ID:         id,
		Attributes: oldAttrs,
	}
	diff := &terraform.InstanceDiff{
		Attributes: diffAttrs,
	}
	d, err := schema.InternalMap(res.Schema).Data(state, diff)
	if err != nil {
		t.Fatalf("failed to create resource data with diff: %v", err)
	}
	d.SetId(id)
	return d
}

func TestResourceDiskCreateAndRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupDiskHandlers(ms, "test-disk")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceDisk()
	d := newTestResourceData(t, res)

	d.Set("name", "test-disk")
	d.Set("size", 100)
	d.Set("image", "ubuntu-minimal.24-04.1")
	d.Set("zone", "se-sto-1a")

	ctx := context.Background()
	d.SetId("test-disk")

	// Test Read
	diags := resourceDiskRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	assertField(t, d, "name", "test-disk")
	assertField(t, d, "region", "se-sto")
	assertField(t, d, "size", 100)
	assertField(t, d, "image", "ubuntu-minimal.24-04.1")
	assertField(t, d, "zone", "se-sto-1a")
}

func TestResourceDiskReadNotFound(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceDisk()
	d := newTestResourceData(t, res)
	d.SetId("nonexistent-disk")

	ctx := context.Background()
	diags := resourceDiskRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("expected no error for not found, got: %v", diagnosticsToString(diags))
	}
	if d.Id() != "" {
		t.Errorf("expected ID to be cleared on not-found, got %q", d.Id())
	}
}

func TestResourceDiskUpdate(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupDiskHandlers(ms, "test-disk")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceDisk()
	d := newTestResourceData(t, res)
	d.SetId("test-disk")
	d.Set("name", "test-disk")

	ctx := context.Background()
	diags := resourceDiskUpdate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected update error: %v", diagnosticsToString(diags))
	}
}

func TestResourceDiskDelete(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupDiskHandlers(ms, "test-disk")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceDisk()
	d := newTestResourceData(t, res)
	d.SetId("test-disk")
	d.Set("name", "test-disk")

	// Set a short timeout for the test
	d.Set("name", "test-disk")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	diags := resourceDiskDelete(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected delete error: %v", diagnosticsToString(diags))
	}
}

// ============================================================================
// Public IP CRUD Tests
// ============================================================================

func TestResourcePublicIPCreateAndRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupPublicIPHandlers(ms, "test-pip")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourcePublicIP()
	d := newTestResourceData(t, res)
	d.SetId("test-pip")
	d.Set("name", "test-pip")

	ctx := context.Background()
	diags := resourcePublicIPRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	if d.Get("name").(string) != "test-pip" {
		t.Errorf("expected name test-pip, got %s", d.Get("name"))
	}
	if d.Get("ip_address").(string) != "203.0.113.1" {
		t.Errorf("expected ip_address 203.0.113.1, got %s", d.Get("ip_address"))
	}
}

func TestResourcePublicIPReadNotFound(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourcePublicIP()
	d := newTestResourceData(t, res)
	d.SetId("nonexistent-pip")

	ctx := context.Background()
	diags := resourcePublicIPRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("expected no error for not found, got: %v", diagnosticsToString(diags))
	}
	if d.Id() != "" {
		t.Errorf("expected ID to be cleared on not-found, got %q", d.Id())
	}
}

func TestResourcePublicIPUpdate(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupPublicIPHandlers(ms, "test-pip")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourcePublicIP()
	d := newTestResourceData(t, res)
	d.SetId("test-pip")
	d.Set("name", "test-pip")

	ctx := context.Background()
	diags := resourcePublicIPUpdate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected update error: %v", diagnosticsToString(diags))
	}
}

func TestResourcePublicIPDelete(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupPublicIPHandlers(ms, "test-pip")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourcePublicIP()
	d := newTestResourceData(t, res)
	d.SetId("test-pip")
	d.Set("name", "test-pip")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	diags := resourcePublicIPDelete(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected delete error: %v", diagnosticsToString(diags))
	}
}

// ============================================================================
// Security Group CRUD Tests
// ============================================================================

func TestResourceSecurityGroupRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupSecurityGroupHandlers(ms, "test-sg")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceSecurityGroup()
	d := newTestResourceData(t, res)
	d.SetId("test-sg")

	ctx := context.Background()
	diags := resourceSecurityGroupRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	assertField(t, d, "name", "test-sg")
	assertField(t, d, "region", "se-sto")
}

func TestResourceSecurityGroupReadNotFound(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceSecurityGroup()
	d := newTestResourceData(t, res)
	d.SetId("nonexistent-sg")

	ctx := context.Background()
	diags := resourceSecurityGroupRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("expected no error for not found, got: %v", diagnosticsToString(diags))
	}
	if d.Id() != "" {
		t.Errorf("expected ID to be cleared on not-found, got %q", d.Id())
	}
}

func TestResourceSecurityGroupUpdate(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupSecurityGroupHandlers(ms, "test-sg")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceSecurityGroup()
	d := newTestResourceData(t, res)
	d.SetId("test-sg")
	d.Set("name", "test-sg")

	ctx := context.Background()
	diags := resourceSecurityGroupUpdate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected update error: %v", diagnosticsToString(diags))
	}
}

func TestResourceSecurityGroupDelete(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupSecurityGroupHandlers(ms, "test-sg")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceSecurityGroup()
	d := newTestResourceData(t, res)
	d.SetId("test-sg")
	d.Set("name", "test-sg")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	diags := resourceSecurityGroupDelete(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected delete error: %v", diagnosticsToString(diags))
	}
}

// ============================================================================
// Placement Group CRUD Tests
// ============================================================================

func TestResourcePlacementGroupRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupPlacementGroupHandlers(ms, "test-pg")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourcePlacementGroup()
	d := newTestResourceData(t, res)
	d.SetId("test-pg")

	ctx := context.Background()
	diags := resourcePlacementGroupRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	assertField(t, d, "name", "test-pg")
	assertField(t, d, "strategy", "spread")
	assertField(t, d, "region", "se-sto")
	assertField(t, d, "zone", "se-sto-1a")
}

func TestResourcePlacementGroupUpdate(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupPlacementGroupHandlers(ms, "test-pg")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourcePlacementGroup()
	d := newTestResourceData(t, res)
	d.SetId("test-pg")
	d.Set("name", "test-pg")

	ctx := context.Background()
	diags := resourcePlacementGroupUpdate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected update error: %v", diagnosticsToString(diags))
	}
}

func TestResourcePlacementGroupDelete(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupPlacementGroupHandlers(ms, "test-pg")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourcePlacementGroup()
	d := newTestResourceData(t, res)
	d.SetId("test-pg")
	d.Set("name", "test-pg")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	diags := resourcePlacementGroupDelete(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected delete error: %v", diagnosticsToString(diags))
	}
}

func TestResourcePlacementGroupReadNotFound(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourcePlacementGroup()
	d := newTestResourceData(t, res)
	d.SetId("nonexistent-pg")

	ctx := context.Background()
	diags := resourcePlacementGroupRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("expected no error for not found, got: %v", diagnosticsToString(diags))
	}
	if d.Id() != "" {
		t.Errorf("expected ID to be cleared on not-found, got %q", d.Id())
	}
}

// ============================================================================
// Bucket CRUD Tests
// ============================================================================

func TestResourceBucketReadNotFound(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceBucket()
	d := newTestResourceData(t, res)
	d.SetId("nonexistent-bucket")

	ctx := context.Background()
	diags := resourceBucketRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("expected no error for not found, got: %v", diagnosticsToString(diags))
	}
	if d.Id() != "" {
		t.Errorf("expected ID to be cleared on not-found, got %q", d.Id())
	}
}

func TestResourceBucketRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupBucketHandlers(ms, "test-bucket")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceBucket()
	d := newTestResourceData(t, res)
	d.SetId("test-bucket")

	ctx := context.Background()
	diags := resourceBucketRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	assertField(t, d, "name", "test-bucket")
	assertField(t, d, "region", "se-sto")
	assertField(t, d, "object_retention_mode", "Disabled")
}

func TestResourceBucketUpdate(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupBucketHandlers(ms, "test-bucket")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceBucket()
	d := newTestResourceData(t, res)
	d.SetId("test-bucket")
	d.Set("name", "test-bucket")

	ctx := context.Background()
	diags := resourceBucketUpdate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected update error: %v", diagnosticsToString(diags))
	}
}

func TestResourceBucketDelete(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupBucketHandlers(ms, "test-bucket")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceBucket()
	d := newTestResourceData(t, res)
	d.SetId("test-bucket")
	d.Set("name", "test-bucket")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	diags := resourceBucketDelete(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected delete error: %v", diagnosticsToString(diags))
	}
}

// ============================================================================
// Bucket Service Account CRUD Tests
// ============================================================================

func TestResourceBucketServiceAccountReadNotFound(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceBucketServiceAccount()
	d := newTestResourceData(t, res)
	d.SetId("nonexistent-sa")

	ctx := context.Background()
	diags := resourceBucketServiceAccountRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("expected no error for not found, got: %v", diagnosticsToString(diags))
	}
	if d.Id() != "" {
		t.Errorf("expected ID to be cleared on not-found, got %q", d.Id())
	}
}

func TestResourceBucketServiceAccountRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupBucketServiceAccountHandlers(ms, "test-sa")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceBucketServiceAccount()
	d := newTestResourceData(t, res)
	d.SetId("test-sa")

	ctx := context.Background()
	diags := resourceBucketServiceAccountRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	assertField(t, d, "name", "test-sa")
	assertField(t, d, "region", "se-sto")
	assertField(t, d, "credentials_secret", "s3-credentials-secret")
	buckets := d.Get("buckets").([]interface{})
	if len(buckets) != 1 || buckets[0].(string) != "test-bucket" {
		t.Errorf("expected buckets=[test-bucket], got %v", buckets)
	}
}

func TestResourceBucketServiceAccountUpdate(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupBucketServiceAccountHandlers(ms, "test-sa")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceBucketServiceAccount()
	d := newTestResourceData(t, res)
	d.SetId("test-sa")
	d.Set("name", "test-sa")

	ctx := context.Background()
	diags := resourceBucketServiceAccountUpdate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected update error: %v", diagnosticsToString(diags))
	}
}

func TestResourceBucketServiceAccountDelete(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupBucketServiceAccountHandlers(ms, "test-sa")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceBucketServiceAccount()
	d := newTestResourceData(t, res)
	d.SetId("test-sa")
	d.Set("name", "test-sa")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	diags := resourceBucketServiceAccountDelete(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected delete error: %v", diagnosticsToString(diags))
	}
}

// ============================================================================
// Project CRUD Tests
// ============================================================================

func TestResourceProjectReadNotFound(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceProject()
	d := newTestResourceData(t, res)
	d.SetId("nonexistent-project")

	ctx := context.Background()
	diags := resourceProjectRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("expected no error for not found, got: %v", diagnosticsToString(diags))
	}
	if d.Id() != "" {
		t.Errorf("expected ID to be cleared on not-found, got %q", d.Id())
	}
}

func TestResourceProjectRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupProjectHandlers(ms, "test-project")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceProject()
	d := newTestResourceData(t, res)
	d.SetId("test-project")

	ctx := context.Background()
	diags := resourceProjectRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	assertField(t, d, "name", "test-project")
	assertField(t, d, "organization", "test-org")
	assertField(t, d, "display_name", "Test Project")
}

// Regression: flattenLabels must return an empty map for nil labels,
// and the Read guard must skip writing empty labels to state. Without
// this, Terraform shows a spurious "user_labels = {} -> null" diff.
func TestFlattenLabelsNil(t *testing.T) {
	result := flattenLabels[map[string]string](nil)
	if len(result) != 0 {
		t.Errorf("flattenLabels(nil) should return empty map, got: %v", result)
	}
}

func TestResourceProjectReadNoLabelsNoDiff(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupProjectHandlers(ms, "test-project")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceProject()
	d := newTestResourceData(t, res)
	d.SetId("test-project")

	ctx := context.Background()
	diags := resourceProjectRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}

	// After Read, user_labels should not be present in state when the
	// API returns nil labels and the user did not configure any.
	raw := d.Get("user_labels").(map[string]interface{})
	if len(raw) != 0 {
		t.Errorf("expected user_labels to be absent/empty in state when API returns nil labels, got: %v", raw)
	}
}

func TestResourceProjectUpdate(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupProjectHandlers(ms, "test-project")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceProject()
	d := newTestResourceData(t, res)
	d.SetId("test-project")
	d.Set("name", "test-project")

	ctx := context.Background()
	diags := resourceProjectUpdate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected update error: %v", diagnosticsToString(diags))
	}
}

func TestResourceProjectDelete(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupProjectHandlers(ms, "test-project")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceProject()
	d := newTestResourceData(t, res)
	d.SetId("test-project")
	d.Set("name", "test-project")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	diags := resourceProjectDelete(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected delete error: %v", diagnosticsToString(diags))
	}
}

// ============================================================================
// Data Source Read Tests (API-backed)
// ============================================================================

func TestDataSourceDiskRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupDiskHandlers(ms, "test-disk")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := dataSourceDisk()
	d := res.TestResourceData()
	d.Set("name", "test-disk")

	ctx := context.Background()
	diags := dataSourceDiskRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	assertField(t, d, "name", "test-disk")
	assertField(t, d, "region", "se-sto")
	assertField(t, d, "size", 100)
	assertField(t, d, "image", "ubuntu-minimal.24-04.1")
}

func TestDataSourcePublicIPRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupPublicIPHandlers(ms, "test-pip")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := dataSourcePublicIP()
	d := res.TestResourceData()
	d.Set("name", "test-pip")

	ctx := context.Background()
	diags := dataSourcePublicIPRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	assertField(t, d, "name", "test-pip")
	assertField(t, d, "region", "se-sto")
	assertField(t, d, "ip_address", "203.0.113.1")
}

func TestDataSourceSecurityGroupRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupSecurityGroupHandlers(ms, "test-sg")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := dataSourceSecurityGroup()
	d := res.TestResourceData()
	d.Set("name", "test-sg")

	ctx := context.Background()
	diags := dataSourceSecurityGroupRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	assertField(t, d, "name", "test-sg")
	assertField(t, d, "region", "se-sto")
}

func TestDataSourcePlacementGroupRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupPlacementGroupHandlers(ms, "test-pg")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := dataSourcePlacementGroup()
	d := res.TestResourceData()
	d.Set("name", "test-pg")

	ctx := context.Background()
	diags := dataSourcePlacementGroupRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	assertField(t, d, "name", "test-pg")
	assertField(t, d, "region", "se-sto")
	assertField(t, d, "strategy", "spread")
	assertField(t, d, "zone", "se-sto-1a")
}

func TestDataSourceBucketRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupBucketHandlers(ms, "test-bucket")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := dataSourceBucket()
	d := res.TestResourceData()
	d.Set("name", "test-bucket")

	ctx := context.Background()
	diags := dataSourceBucketRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	assertField(t, d, "name", "test-bucket")
	assertField(t, d, "region", "se-sto")
	assertField(t, d, "object_retention_mode", "Disabled")
}

func TestDataSourceBucketServiceAccountRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupBucketServiceAccountHandlers(ms, "test-sa")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := dataSourceBucketServiceAccount()
	d := res.TestResourceData()
	d.Set("name", "test-sa")

	ctx := context.Background()
	diags := dataSourceBucketServiceAccountRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	assertField(t, d, "name", "test-sa")
	assertField(t, d, "region", "se-sto")
	assertField(t, d, "credentials_secret", "s3-credentials-secret")
	buckets := d.Get("buckets").([]interface{})
	if len(buckets) != 1 || buckets[0].(string) != "test-bucket" {
		t.Errorf("expected buckets=[test-bucket], got %v", buckets)
	}
}

func TestDataSourceProjectRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupProjectHandlers(ms, "test-project")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := dataSourceProject()
	d := res.TestResourceData()
	d.Set("name", "test-project")

	ctx := context.Background()
	diags := dataSourceProjectRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	assertField(t, d, "name", "test-project")
	assertField(t, d, "organization", "test-org")
	assertField(t, d, "display_name", "Test Project")
}

// ============================================================================
// Virtual Machine CRUD Tests
// ============================================================================

func TestResourceVirtualMachineRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupVirtualMachineHandlers(ms, "test-vm")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceVirtualMachine()
	d := newTestResourceData(t, res)
	d.SetId("test-vm")

	ctx := context.Background()
	diags := resourceVirtualMachineRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	if d.Get("name").(string) != "test-vm" {
		t.Errorf("expected name test-vm, got %s", d.Get("name"))
	}
	if d.Get("flavor").(string) != "a1a.s" {
		t.Errorf("expected flavor a1a.s, got %s", d.Get("flavor"))
	}
	if d.Get("boot_disk").(string) != "/compute/projects/test-project/regions/se-sto/disks/test-disk" {
		t.Errorf("expected boot_disk FQID, got %s", d.Get("boot_disk"))
	}
	if d.Get("zone").(string) != "se-sto-1a" {
		t.Errorf("expected zone se-sto-1a, got %s", d.Get("zone"))
	}
	if d.Get("public_ipv4_address").(string) != "203.0.113.1" {
		t.Errorf("expected public_ipv4_address 203.0.113.1, got %s", d.Get("public_ipv4_address"))
	}
	if d.Get("private_ipv4_address").(string) != "10.0.0.1" {
		t.Errorf("expected private_ipv4_address 10.0.0.1, got %s", d.Get("private_ipv4_address"))
	}
}

func TestResourceVirtualMachineReadNotFound(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceVirtualMachine()
	d := newTestResourceData(t, res)
	d.SetId("nonexistent-vm")

	ctx := context.Background()
	diags := resourceVirtualMachineRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("expected no error for not found, got: %v", diagnosticsToString(diags))
	}
	if d.Id() != "" {
		t.Errorf("expected ID to be cleared on not-found, got %q", d.Id())
	}
}

func TestResourceVirtualMachineUpdate(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupVirtualMachineHandlers(ms, "test-vm")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceVirtualMachine()
	d := newTestResourceData(t, res)
	d.SetId("test-vm")
	d.Set("name", "test-vm")

	ctx := context.Background()
	diags := resourceVirtualMachineUpdate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected update error: %v", diagnosticsToString(diags))
	}
}

func TestResourceVirtualMachineUpdateFlavor(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupVirtualMachineHandlers(ms, "test-vm")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	d := newResourceDataWithDiff(t, resourceVirtualMachine(), "test-vm",
		map[string]string{"name": "test-vm", "flavor": "a1a.s"},
		map[string]*terraform.ResourceAttrDiff{
			"flavor": {Old: "a1a.s", New: "c1a.m"},
		},
	)

	ctx := context.Background()
	diags := resourceVirtualMachineUpdate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected update error: %v", diagnosticsToString(diags))
	}
}

func TestResourceVirtualMachineUpdateRunning(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupVirtualMachineHandlers(ms, "test-vm")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	d := newResourceDataWithDiff(t, resourceVirtualMachine(), "test-vm",
		map[string]string{"name": "test-vm", "running": "true"},
		map[string]*terraform.ResourceAttrDiff{
			"running": {Old: "true", New: "false"},
		},
	)

	ctx := context.Background()
	diags := resourceVirtualMachineUpdate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected update error: %v", diagnosticsToString(diags))
	}
}

func TestResourceVirtualMachineUpdatePublicIP(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupVirtualMachineHandlers(ms, "test-vm")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	d := newResourceDataWithDiff(t, resourceVirtualMachine(), "test-vm",
		map[string]string{"name": "test-vm", "public_ip": "/networking/projects/test-project/regions/se-sto/publicIPs/old-pip"},
		map[string]*terraform.ResourceAttrDiff{
			"public_ip": {Old: "/networking/projects/test-project/regions/se-sto/publicIPs/old-pip", New: "/networking/projects/test-project/regions/se-sto/publicIPs/new-pip"},
		},
	)

	ctx := context.Background()
	diags := resourceVirtualMachineUpdate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected update error: %v", diagnosticsToString(diags))
	}
}

func TestResourceVirtualMachineUpdatePublicIPPlainName(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupVirtualMachineHandlers(ms, "test-vm")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	d := newResourceDataWithDiff(t, resourceVirtualMachine(), "test-vm",
		map[string]string{"name": "test-vm", "public_ip": "/networking/projects/test-project/regions/se-sto/publicIPs/old-pip"},
		map[string]*terraform.ResourceAttrDiff{
			"public_ip": {Old: "/networking/projects/test-project/regions/se-sto/publicIPs/old-pip", New: "new-pip"},
		},
	)

	ctx := context.Background()
	diags := resourceVirtualMachineUpdate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected update error: %v", diagnosticsToString(diags))
	}
}

func TestResourceVirtualMachineUpdateRemovePublicIP(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupVirtualMachineHandlers(ms, "test-vm")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	d := newResourceDataWithDiff(t, resourceVirtualMachine(), "test-vm",
		map[string]string{"name": "test-vm", "public_ip": "/networking/projects/test-project/regions/se-sto/publicIPs/old-pip"},
		map[string]*terraform.ResourceAttrDiff{
			"public_ip": {Old: "/networking/projects/test-project/regions/se-sto/publicIPs/old-pip", New: ""},
		},
	)

	ctx := context.Background()
	diags := resourceVirtualMachineUpdate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected update error: %v", diagnosticsToString(diags))
	}
}

func TestResourceVirtualMachineUpdateSecurityGroups(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupVirtualMachineHandlers(ms, "test-vm")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	d := newResourceDataWithDiff(t, resourceVirtualMachine(), "test-vm",
		map[string]string{
			"name":              "test-vm",
			"security_groups.#": "1",
			"security_groups.0": "/networking/projects/test-project/regions/se-sto/securityGroups/old-sg",
		},
		map[string]*terraform.ResourceAttrDiff{
			"security_groups.#": {Old: "1", New: "2"},
			"security_groups.0": {Old: "/networking/projects/test-project/regions/se-sto/securityGroups/old-sg", New: "/networking/projects/test-project/regions/se-sto/securityGroups/old-sg"},
			"security_groups.1": {Old: "", New: "/networking/projects/test-project/regions/se-sto/securityGroups/new-sg"},
		},
	)

	ctx := context.Background()
	diags := resourceVirtualMachineUpdate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected update error: %v", diagnosticsToString(diags))
	}
}

func TestResourceVirtualMachineUpdateSecurityGroupsPlainName(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupVirtualMachineHandlers(ms, "test-vm")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	d := newResourceDataWithDiff(t, resourceVirtualMachine(), "test-vm",
		map[string]string{
			"name":              "test-vm",
			"security_groups.#": "1",
			"security_groups.0": "/networking/projects/test-project/regions/se-sto/securityGroups/old-sg",
		},
		map[string]*terraform.ResourceAttrDiff{
			"security_groups.#": {Old: "1", New: "1"},
			"security_groups.0": {Old: "/networking/projects/test-project/regions/se-sto/securityGroups/old-sg", New: "new-sg"},
		},
	)

	ctx := context.Background()
	diags := resourceVirtualMachineUpdate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected update error: %v", diagnosticsToString(diags))
	}
}

func TestResourceVirtualMachineUpdatePlacementGroup(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupVirtualMachineHandlers(ms, "test-vm")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	d := newResourceDataWithDiff(t, resourceVirtualMachine(), "test-vm",
		map[string]string{"name": "test-vm", "placement_group": "/compute/projects/test-project/regions/se-sto/placementGroups/old-pg"},
		map[string]*terraform.ResourceAttrDiff{
			"placement_group": {Old: "/compute/projects/test-project/regions/se-sto/placementGroups/old-pg", New: "/compute/projects/test-project/regions/se-sto/placementGroups/new-pg"},
		},
	)

	ctx := context.Background()
	diags := resourceVirtualMachineUpdate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected update error: %v", diagnosticsToString(diags))
	}
}

func TestResourceVirtualMachineUpdatePlacementGroupPlainName(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupVirtualMachineHandlers(ms, "test-vm")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	d := newResourceDataWithDiff(t, resourceVirtualMachine(), "test-vm",
		map[string]string{"name": "test-vm", "placement_group": "/compute/projects/test-project/regions/se-sto/placementGroups/old-pg"},
		map[string]*terraform.ResourceAttrDiff{
			"placement_group": {Old: "/compute/projects/test-project/regions/se-sto/placementGroups/old-pg", New: "new-pg"},
		},
	)

	ctx := context.Background()
	diags := resourceVirtualMachineUpdate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected update error: %v", diagnosticsToString(diags))
	}
}

func TestResourceVirtualMachineUpdateRemovePlacementGroup(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupVirtualMachineHandlers(ms, "test-vm")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	d := newResourceDataWithDiff(t, resourceVirtualMachine(), "test-vm",
		map[string]string{"name": "test-vm", "placement_group": "/compute/projects/test-project/regions/se-sto/placementGroups/old-pg"},
		map[string]*terraform.ResourceAttrDiff{
			"placement_group": {Old: "/compute/projects/test-project/regions/se-sto/placementGroups/old-pg", New: ""},
		},
	)

	ctx := context.Background()
	diags := resourceVirtualMachineUpdate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected update error: %v", diagnosticsToString(diags))
	}
}

func TestResourceVirtualMachineUpdateLabels(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupVirtualMachineHandlers(ms, "test-vm")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	d := newResourceDataWithDiff(t, resourceVirtualMachine(), "test-vm",
		map[string]string{
			"name":            "test-vm",
			"user_labels.%":   "1",
			"user_labels.env": "prod",
		},
		map[string]*terraform.ResourceAttrDiff{
			"user_labels.%":       {Old: "1", New: "2"},
			"user_labels.env":     {Old: "prod", New: "staging"},
			"user_labels.new-key": {Old: "", New: "new-value"},
		},
	)

	ctx := context.Background()
	diags := resourceVirtualMachineUpdate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected update error: %v", diagnosticsToString(diags))
	}
}

func TestResourceVirtualMachineCreateWithFQID(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupVirtualMachineHandlers(ms, "test-vm")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceVirtualMachine()
	d := newTestResourceData(t, res)
	d.Set("name", "test-vm")
	d.Set("flavor", "a1a.s")
	d.Set("boot_disk", "/compute/projects/test-project/regions/se-sto/disks/test-disk")
	d.Set("zone", "se-sto-1a")
	d.Set("public_ip", "/networking/projects/test-project/regions/se-sto/publicIPs/test-pip")
	d.Set("security_groups", []interface{}{"/networking/projects/test-project/regions/se-sto/securityGroups/test-sg"})
	d.Set("placement_group", "/compute/projects/test-project/regions/se-sto/placementGroups/test-pg")

	ctx := context.Background()
	diags := resourceVirtualMachineCreate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected create error: %v", diagnosticsToString(diags))
	}
	if d.Id() == "" {
		t.Error("expected ID to be set after create")
	}
}

func TestResourceHotswapDiskAttachmentCreateWithFQID(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupDiskAttachmentHandlers(ms, "test-attach")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceHotswapDiskAttachment()
	d := newTestResourceData(t, res)
	d.Set("name", "test-attach")
	d.Set("virtual_machine", "/compute/projects/test-project/regions/se-sto/virtualMachines/test-vm")
	d.Set("disk", "/compute/projects/test-project/regions/se-sto/disks/test-disk")

	ctx := context.Background()
	diags := resourceHotswapDiskAttachmentCreate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected create error: %v", diagnosticsToString(diags))
	}
	if d.Id() == "" {
		t.Error("expected ID to be set after create")
	}
}

func TestResourceVirtualMachineDelete(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupVirtualMachineHandlers(ms, "test-vm")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceVirtualMachine()
	d := newTestResourceData(t, res)
	d.SetId("test-vm")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	diags := resourceVirtualMachineDelete(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected delete error: %v", diagnosticsToString(diags))
	}
}

// ============================================================================
// Disk Attachment CRUD Tests
// ============================================================================

func TestResourceHotswapDiskAttachmentReadNotFound(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceHotswapDiskAttachment()
	d := newTestResourceData(t, res)
	d.SetId("nonexistent-attach")

	ctx := context.Background()
	diags := resourceHotswapDiskAttachmentRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("expected no error for not found, got: %v", diagnosticsToString(diags))
	}
	if d.Id() != "" {
		t.Errorf("expected ID to be cleared on not-found, got %q", d.Id())
	}
}

func TestResourceHotswapDiskAttachmentRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupDiskAttachmentHandlers(ms, "test-attach")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceHotswapDiskAttachment()
	d := newTestResourceData(t, res)
	d.SetId("test-attach")

	ctx := context.Background()
	diags := resourceHotswapDiskAttachmentRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	if d.Get("name").(string) != "test-attach" {
		t.Errorf("expected name test-attach, got %s", d.Get("name"))
	}
	if d.Get("virtual_machine").(string) != "/compute/projects/test-project/regions/se-sto/virtualMachines/test-vm" {
		t.Errorf("expected virtual_machine FQID, got %s", d.Get("virtual_machine"))
	}
	if d.Get("disk").(string) != "/compute/projects/test-project/regions/se-sto/disks/test-disk" {
		t.Errorf("expected disk FQID, got %s", d.Get("disk"))
	}
	if d.Get("serial").(string) != "disk-serial-123" {
		t.Errorf("expected serial disk-serial-123, got %s", d.Get("serial"))
	}
}

func TestResourceHotswapDiskAttachmentUpdate(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupDiskAttachmentHandlers(ms, "test-attach")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceHotswapDiskAttachment()
	d := newTestResourceData(t, res)
	d.SetId("test-attach")
	d.Set("name", "test-attach")

	ctx := context.Background()
	diags := resourceHotswapDiskAttachmentUpdate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected update error: %v", diagnosticsToString(diags))
	}
}

func TestResourceHotswapDiskAttachmentDelete(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupDiskAttachmentHandlers(ms, "test-attach")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceHotswapDiskAttachment()
	d := newTestResourceData(t, res)
	d.SetId("test-attach")

	ctx := context.Background()
	diags := resourceHotswapDiskAttachmentDelete(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected delete error: %v", diagnosticsToString(diags))
	}
}

// ============================================================================
// Create Tests (all resources)
// ============================================================================

func TestResourceDiskCreate(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupDiskHandlers(ms, "test-disk")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceDisk()
	d := newTestResourceData(t, res)
	d.Set("name", "test-disk")
	d.Set("size", 100)
	d.Set("image", "ubuntu-minimal.24-04.1")
	d.Set("zone", "se-sto-1a")

	ctx := context.Background()
	diags := resourceDiskCreate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected create error: %v", diagnosticsToString(diags))
	}
	if d.Id() == "" {
		t.Error("expected ID to be set after create")
	}
}

func TestResourcePublicIPCreate(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupPublicIPHandlers(ms, "test-pip")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourcePublicIP()
	d := newTestResourceData(t, res)
	d.Set("name", "test-pip")

	ctx := context.Background()
	diags := resourcePublicIPCreate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected create error: %v", diagnosticsToString(diags))
	}
	if d.Id() == "" {
		t.Error("expected ID to be set after create")
	}
}

func TestResourceSecurityGroupCreate(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupSecurityGroupHandlers(ms, "test-sg")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceSecurityGroup()
	d := newTestResourceData(t, res)
	d.Set("name", "test-sg")

	ctx := context.Background()
	diags := resourceSecurityGroupCreate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected create error: %v", diagnosticsToString(diags))
	}
	if d.Id() == "" {
		t.Error("expected ID to be set after create")
	}
}

func TestResourcePlacementGroupCreate(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupPlacementGroupHandlers(ms, "test-pg")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourcePlacementGroup()
	d := newTestResourceData(t, res)
	d.Set("name", "test-pg")
	d.Set("strategy", "spread")
	d.Set("zone", "se-sto-1a")

	ctx := context.Background()
	diags := resourcePlacementGroupCreate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected create error: %v", diagnosticsToString(diags))
	}
	if d.Id() == "" {
		t.Error("expected ID to be set after create")
	}
}

func TestResourceBucketCreate(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupBucketHandlers(ms, "test-bucket")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceBucket()
	d := newTestResourceData(t, res)
	d.Set("name", "test-bucket")

	ctx := context.Background()
	diags := resourceBucketCreate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected create error: %v", diagnosticsToString(diags))
	}
	if d.Id() == "" {
		t.Error("expected ID to be set after create")
	}
}

func TestResourceBucketServiceAccountCreate(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupBucketServiceAccountHandlers(ms, "test-sa")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceBucketServiceAccount()
	d := newTestResourceData(t, res)
	d.Set("name", "test-sa")
	d.Set("buckets", []interface{}{"test-bucket"})

	ctx := context.Background()
	diags := resourceBucketServiceAccountCreate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected create error: %v", diagnosticsToString(diags))
	}
	if d.Id() == "" {
		t.Error("expected ID to be set after create")
	}
}

func TestResourceProjectCreate(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupProjectHandlers(ms, "test-project")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceProject()
	d := newTestResourceData(t, res)
	d.Set("name", "test-project")
	d.Set("organization", "test-org")

	ctx := context.Background()
	diags := resourceProjectCreate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected create error: %v", diagnosticsToString(diags))
	}
	if d.Id() == "" {
		t.Error("expected ID to be set after create")
	}
}

func TestResourceVirtualMachineCreate(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupVirtualMachineHandlers(ms, "test-vm")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceVirtualMachine()
	d := newTestResourceData(t, res)
	d.Set("name", "test-vm")
	d.Set("flavor", "a1a.s")
	d.Set("boot_disk", "test-disk")
	d.Set("zone", "se-sto-1a")

	ctx := context.Background()
	diags := resourceVirtualMachineCreate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected create error: %v", diagnosticsToString(diags))
	}
	if d.Id() == "" {
		t.Error("expected ID to be set after create")
	}
}

func TestResourceHotswapDiskAttachmentCreate(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupDiskAttachmentHandlers(ms, "test-attach")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceHotswapDiskAttachment()
	d := newTestResourceData(t, res)
	d.Set("name", "test-attach")
	d.Set("virtual_machine", "test-vm")
	d.Set("disk", "test-disk")

	ctx := context.Background()
	diags := resourceHotswapDiskAttachmentCreate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected create error: %v", diagnosticsToString(diags))
	}
	if d.Id() == "" {
		t.Error("expected ID to be set after create")
	}
}

// ============================================================================
// Data Source Read Tests (VM and HotswapDiskAttachment)
// ============================================================================

func TestDataSourceVirtualMachineRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupVirtualMachineHandlers(ms, "test-vm")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := dataSourceVirtualMachine()
	d := res.TestResourceData()
	d.Set("name", "test-vm")

	ctx := context.Background()
	diags := dataSourceVirtualMachineRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	assertField(t, d, "name", "test-vm")
	assertField(t, d, "region", "se-sto")
	assertField(t, d, "flavor", "a1a.s")
	assertField(t, d, "boot_disk", "/compute/projects/test-project/regions/se-sto/disks/test-disk")
	assertField(t, d, "cloud_config_user_data", "#!/bin/bash\necho hello")
	assertField(t, d, "status", "Running")
}

func TestDataSourceHotswapDiskAttachmentRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupDiskAttachmentHandlers(ms, "test-attach")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := dataSourceHotswapDiskAttachment()
	d := res.TestResourceData()
	d.Set("name", "test-attach")

	ctx := context.Background()
	diags := dataSourceHotswapDiskAttachmentRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	assertField(t, d, "name", "test-attach")
	assertField(t, d, "region", "se-sto")
	assertField(t, d, "virtual_machine", "/compute/projects/test-project/regions/se-sto/virtualMachines/test-vm")
	assertField(t, d, "disk", "/compute/projects/test-project/regions/se-sto/disks/test-disk")
	assertField(t, d, "serial", "disk-serial-123")
}

// ============================================================================
// Data Source NotFound Tests
// ============================================================================

func TestDataSourceDiskReadNotFound(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := dataSourceDisk()
	d := res.TestResourceData()
	d.Set("name", "nonexistent-disk")

	diags := dataSourceDiskRead(context.Background(), d, config)
	if !diags.HasError() {
		t.Fatal("expected error for nonexistent data source, got none")
	}
}

func TestDataSourcePublicIPReadNotFound(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := dataSourcePublicIP()
	d := res.TestResourceData()
	d.Set("name", "nonexistent-pip")

	diags := dataSourcePublicIPRead(context.Background(), d, config)
	if !diags.HasError() {
		t.Fatal("expected error for nonexistent data source, got none")
	}
}

func TestDataSourceSecurityGroupReadNotFound(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := dataSourceSecurityGroup()
	d := res.TestResourceData()
	d.Set("name", "nonexistent-sg")

	diags := dataSourceSecurityGroupRead(context.Background(), d, config)
	if !diags.HasError() {
		t.Fatal("expected error for nonexistent data source, got none")
	}
}

func TestDataSourcePlacementGroupReadNotFound(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := dataSourcePlacementGroup()
	d := res.TestResourceData()
	d.Set("name", "nonexistent-pg")

	diags := dataSourcePlacementGroupRead(context.Background(), d, config)
	if !diags.HasError() {
		t.Fatal("expected error for nonexistent data source, got none")
	}
}

func TestDataSourceBucketReadNotFound(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := dataSourceBucket()
	d := res.TestResourceData()
	d.Set("name", "nonexistent-bucket")

	diags := dataSourceBucketRead(context.Background(), d, config)
	if !diags.HasError() {
		t.Fatal("expected error for nonexistent data source, got none")
	}
}

func TestDataSourceBucketServiceAccountReadNotFound(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := dataSourceBucketServiceAccount()
	d := res.TestResourceData()
	d.Set("name", "nonexistent-sa")

	diags := dataSourceBucketServiceAccountRead(context.Background(), d, config)
	if !diags.HasError() {
		t.Fatal("expected error for nonexistent data source, got none")
	}
}

func TestDataSourceProjectReadNotFound(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := dataSourceProject()
	d := res.TestResourceData()
	d.Set("name", "nonexistent-project")

	diags := dataSourceProjectRead(context.Background(), d, config)
	if !diags.HasError() {
		t.Fatal("expected error for nonexistent data source, got none")
	}
}

func TestDataSourceVirtualMachineReadNotFound(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := dataSourceVirtualMachine()
	d := res.TestResourceData()
	d.Set("name", "nonexistent-vm")

	diags := dataSourceVirtualMachineRead(context.Background(), d, config)
	if !diags.HasError() {
		t.Fatal("expected error for nonexistent data source, got none")
	}
}

func TestDataSourceHotswapDiskAttachmentReadNotFound(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := dataSourceHotswapDiskAttachment()
	d := res.TestResourceData()
	d.Set("name", "nonexistent-attach")

	diags := dataSourceHotswapDiskAttachmentRead(context.Background(), d, config)
	if !diags.HasError() {
		t.Fatal("expected error for nonexistent data source, got none")
	}
}

// ============================================================================
// Think Instance CRUD Tests
// ============================================================================

func TestResourceThinkInstanceRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupThinkInstanceHandlers(ms, "test-instance")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceThinkInstance()
	d := newTestResourceData(t, res)
	d.SetId("test-instance")

	ctx := context.Background()
	diags := resourceThinkInstanceRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	assertField(t, d, "name", "test-instance")
	assertField(t, d, "region", "se-sto")
	assertField(t, d, "model", "meta-llama/Llama-3.3-70B-Instruct")
	assertField(t, d, "size", "a100.2x")
	assertField(t, d, "phase", "Running")
	assertField(t, d, "running", true)
	assertField(t, d, "endpoint", "https://models.think.se-sto.evroc.com/projects/test-project/instances/test-instance")
}

func TestResourceThinkInstanceReadNotFound(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceThinkInstance()
	d := newTestResourceData(t, res)
	d.SetId("nonexistent-instance")

	ctx := context.Background()
	diags := resourceThinkInstanceRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("expected no error for not found, got: %v", diagnosticsToString(diags))
	}
	if d.Id() != "" {
		t.Errorf("expected ID to be cleared on not-found, got %q", d.Id())
	}
}

func TestResourceThinkInstanceDelete(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupThinkInstanceHandlers(ms, "test-instance")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceThinkInstance()
	d := newTestResourceData(t, res)
	d.SetId("test-instance")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	diags := resourceThinkInstanceDelete(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected delete error: %v", diagnosticsToString(diags))
	}
}

// ============================================================================
// Think API Key CRUD Tests
// ============================================================================

func TestResourceThinkAPIKeyCreate(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupThinkAPIKeyHandlers(ms, "test-api-key")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceThinkAPIKey()
	d := newTestResourceData(t, res)
	d.Set("name", "test-api-key")

	ctx := context.Background()
	diags := resourceThinkAPIKeyCreate(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected create error: %v", diagnosticsToString(diags))
	}
	if d.Id() == "" {
		t.Error("expected ID to be set after create")
	}
	if d.Get("token").(string) != "ev-test-token-secret-1234567890" {
		t.Errorf("expected token to be set, got %q", d.Get("token"))
	}
	if d.Get("token_prefix").(string) != "ev-test" {
		t.Errorf("expected token_prefix ev-test, got %q", d.Get("token_prefix"))
	}
}

func TestResourceThinkAPIKeyRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupThinkAPIKeyHandlers(ms, "test-api-key")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceThinkAPIKey()
	d := newTestResourceData(t, res)
	d.SetId("test-api-key")

	ctx := context.Background()
	diags := resourceThinkAPIKeyRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	assertField(t, d, "name", "test-api-key")
	assertField(t, d, "token_prefix", "ev-test")
}

func TestResourceThinkAPIKeyReadNotFound(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	// Set up handlers for a different key so the list is empty for our ID
	setupThinkAPIKeyHandlers(ms, "other-key")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceThinkAPIKey()
	d := newTestResourceData(t, res)
	d.SetId("nonexistent-key")

	ctx := context.Background()
	diags := resourceThinkAPIKeyRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("expected no error for not found, got: %v", diagnosticsToString(diags))
	}
	if d.Id() != "" {
		t.Errorf("expected ID to be cleared on not-found, got %q", d.Id())
	}
}

func TestResourceThinkAPIKeyDelete(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupThinkAPIKeyHandlers(ms, "test-api-key")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := resourceThinkAPIKey()
	d := newTestResourceData(t, res)
	d.SetId("test-api-key")

	ctx := context.Background()
	diags := resourceThinkAPIKeyDelete(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected delete error: %v", diagnosticsToString(diags))
	}
}

// ============================================================================
// Think Data Source Read Tests
// ============================================================================

func TestDataSourceThinkInstanceRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupThinkInstanceHandlers(ms, "test-instance")
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := dataSourceThinkInstance()
	d := res.TestResourceData()
	d.Set("name", "test-instance")

	ctx := context.Background()
	diags := dataSourceThinkInstanceRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	assertField(t, d, "name", "test-instance")
	assertField(t, d, "region", "se-sto")
	assertField(t, d, "model", "meta-llama/Llama-3.3-70B-Instruct")
	assertField(t, d, "size", "a100.2x")
	assertField(t, d, "phase", "Running")
	assertField(t, d, "endpoint", "https://models.think.se-sto.evroc.com/projects/test-project/instances/test-instance")
}

func TestDataSourceThinkInstanceReadNotFound(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := dataSourceThinkInstance()
	d := res.TestResourceData()
	d.Set("name", "nonexistent-instance")

	diags := dataSourceThinkInstanceRead(context.Background(), d, config)
	if !diags.HasError() {
		t.Fatal("expected error for nonexistent data source, got none")
	}
}

func TestDataSourceThinkModelsRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupThinkModelsHandlers(ms)
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := dataSourceThinkModels()
	d := res.TestResourceData()

	ctx := context.Background()
	diags := dataSourceThinkModelsRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	models := d.Get("models").([]interface{})
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}
	model := models[0].(map[string]interface{})
	if model["name"] != "meta-llama/Llama-3.3-70B-Instruct" {
		t.Errorf("expected model name meta-llama/Llama-3.3-70B-Instruct, got %v", model["name"])
	}
}

func TestDataSourceThinkSizesRead(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupThinkSizesHandlers(ms)
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	res := dataSourceThinkSizes()
	d := res.TestResourceData()

	ctx := context.Background()
	diags := dataSourceThinkSizesRead(ctx, d, config)
	if diags.HasError() {
		t.Fatalf("unexpected read error: %v", diagnosticsToString(diags))
	}
	sizes := d.Get("sizes").([]interface{})
	if len(sizes) != 1 {
		t.Fatalf("expected 1 size, got %d", len(sizes))
	}
	size := sizes[0].(map[string]interface{})
	if size["name"] != "a100.2x" {
		t.Errorf("expected size name a100.2x, got %v", size["name"])
	}
}
