// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	networkingtypes "github.com/evroc-oss/evroc-go-sdk/types/networking"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestResourceSubnetCreateWaitsForReady(t *testing.T) {
	for _, tc := range []struct {
		name          string
		conditions    *[]networkingtypes.SubnetStatusConditionsItem
		neverReady    bool
		forbidden     bool
		notFound      bool
		createTimeout bool
	}{
		{name: "missing conditions"},
		{name: "temporarily missing", notFound: true},
		{name: "empty conditions", conditions: &[]networkingtypes.SubnetStatusConditionsItem{}},
		{name: "IPv6 allocated but not ready", conditions: &[]networkingtypes.SubnetStatusConditionsItem{{Type: "Ready", Status: "False"}}},
		{name: "unknown readiness", conditions: &[]networkingtypes.SubnetStatusConditionsItem{{Type: "Ready", Status: "Unknown"}}},
		{name: "unrelated condition", conditions: &[]networkingtypes.SubnetStatusConditionsItem{{Type: "Other", Status: "True"}}},
		{name: "context cancellation preserves resource ID", neverReady: true},
		{name: "create timeout preserves resource ID", neverReady: true, createTimeout: true},
		{name: "API error preserves resource ID", forbidden: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ms := newMockServer()
			defer ms.close()
			path := "/networking/v1beta2/projects/test-project/regions/se-sto/subnets"
			ms.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
				subnet := mockSubnet("test-subnet")
				subnet.Status.AssignedIPv6GUABlock = nil
				subnet.Status.Conditions = nil
				respondJSON(w, http.StatusCreated, subnet)
			})
			var reads atomic.Int32
			ms.mux.HandleFunc(path+"/test-subnet", func(w http.ResponseWriter, r *http.Request) {
				if tc.forbidden {
					respondJSON(w, http.StatusForbidden, map[string]string{"message": "access denied"})
					return
				}
				subnet := mockSubnet("test-subnet")
				read := reads.Add(1)
				if read == 1 && tc.notFound {
					respondJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
					return
				}
				ipv6 := "2001:db8::/64"
				count := 253
				ranges := []string{"10.0.1.2-10.0.1.254"}
				subnet.Status.AssignedIPv6GUABlock = &ipv6
				subnet.Status.AvailableIPv4AddressCount = &count
				subnet.Status.AvailableIPv4Ranges = &ranges
				if read == 1 || tc.neverReady {
					subnet.Status.Conditions = tc.conditions
					subnet.Status.AvailableIPv4AddressCount = nil
					subnet.Status.AvailableIPv4Ranges = nil
				}
				respondJSON(w, http.StatusOK, subnet)
			})
			setupCatchAll(ms)
			config := newTestProviderConfig(t, ms.server.URL)
			res := resourceSubnet()
			if tc.createTimeout {
				res.Timeouts.Create = schema.DefaultTimeout(200 * time.Millisecond)
			}
			d := newTestResourceData(t, res)
			if err := d.Set("name", "test-subnet"); err != nil {
				t.Fatal(err)
			}
			for key, value := range map[string]string{"vpc_ref": "test-vpc", "zone": "a", "ipv4_cidr_block": "10.0.1.0/24"} {
				if err := d.Set(key, value); err != nil {
					t.Fatal(err)
				}
			}
			timeout := 10 * time.Second
			if tc.neverReady && !tc.createTimeout {
				timeout = 200 * time.Millisecond
			}
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			diags := resourceSubnetCreate(ctx, d, config)
			if diags.HasError() != (tc.neverReady || tc.forbidden) {
				t.Fatalf("unexpected result: %v", diags)
			}
			if d.Id() != "test-subnet" {
				t.Fatalf("created subnet must remain in state, got ID %q", d.Id())
			}
			if !diags.HasError() {
				assertField(t, d, "assigned_ipv6_gua_block", "2001:db8::/64")
				assertField(t, d, "available_ipv4_address_count", 253)
				if got := d.Get("available_ipv4_ranges"); !reflect.DeepEqual(got, []interface{}{"10.0.1.2-10.0.1.254"}) {
					t.Fatalf("unexpected IPv4 ranges: %v", got)
				}
				if reads.Load() < 3 {
					t.Fatal("create returned without polling for readiness")
				}
			}
		})
	}
}
