// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	networkingtypes "github.com/evroc-oss/evroc-go-sdk/types/networking"
)

func TestResourceVPCCreateWaitsForReady(t *testing.T) {
	for _, tc := range []struct {
		name       string
		conditions *[]networkingtypes.VirtualPrivateCloudStatusConditionsItem
		neverReady bool
		forbidden  bool
	}{
		{name: "missing conditions"},
		{name: "empty conditions", conditions: &[]networkingtypes.VirtualPrivateCloudStatusConditionsItem{}},
		{name: "IPv6 allocated but not ready", conditions: &[]networkingtypes.VirtualPrivateCloudStatusConditionsItem{{Type: "Ready", Status: "False"}}},
		{name: "unknown readiness", conditions: &[]networkingtypes.VirtualPrivateCloudStatusConditionsItem{{Type: "Ready", Status: "Unknown"}}},
		{name: "unrelated condition", conditions: &[]networkingtypes.VirtualPrivateCloudStatusConditionsItem{{Type: "Other", Status: "True"}}},
		{name: "timeout preserves resource ID", neverReady: true},
		{name: "API error preserves resource ID", forbidden: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ms := newMockServer()
			defer ms.close()
			path := "/networking/v1beta2/projects/test-project/regions/se-sto/virtualPrivateClouds"
			ms.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
				vpc := mockVPC("test-vpc")
				vpc.Status.AssignedIPv6GUABlock = nil
				vpc.Status.Conditions = nil
				respondJSON(w, http.StatusCreated, vpc)
			})
			var reads atomic.Int32
			ms.mux.HandleFunc(path+"/test-vpc", func(w http.ResponseWriter, r *http.Request) {
				if tc.forbidden {
					respondJSON(w, http.StatusForbidden, map[string]string{"message": "access denied"})
					return
				}
				vpc := mockVPC("test-vpc")
				if reads.Add(1) == 1 || tc.neverReady {
					vpc.Status.Conditions = tc.conditions
				}
				respondJSON(w, http.StatusOK, vpc)
			})
			setupCatchAll(ms)
			config := newTestProviderConfig(t, ms.server.URL)
			d := newTestResourceData(t, resourceVPC())
			if err := d.Set("name", "test-vpc"); err != nil {
				t.Fatal(err)
			}
			timeout := 10 * time.Second
			if tc.neverReady {
				timeout = 200 * time.Millisecond
			}
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			diags := resourceVPCCreate(ctx, d, config)
			if diags.HasError() != (tc.neverReady || tc.forbidden) {
				t.Fatalf("unexpected result: %v", diags)
			}
			if d.Id() != "test-vpc" {
				t.Fatalf("created VPC must remain in state, got ID %q", d.Id())
			}
			if !diags.HasError() {
				assertField(t, d, "assigned_ipv6_gua_block", "2001:db8::/56")
				if reads.Load() < 3 {
					t.Fatal("create returned without polling for readiness")
				}
			}
		})
	}
}
