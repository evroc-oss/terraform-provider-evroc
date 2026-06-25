// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import (
	"context"
	"testing"
	"time"
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
	assertField(t, d, "backend_count", 1)
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
