// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	lbtypes "github.com/evroc-oss/evroc-go-sdk/types/loadbalancer"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func loadBalancerTestConfig(network bool) map[string]interface{} {
	cfg := map[string]interface{}{
		"name": "test-lb", "public_ip_ref": "/networking/projects/test-project/regions/se-sto/publicIPs/test-pip",
		"listener": []interface{}{map[string]interface{}{"protocol": "TCP", "port": 9345, "route_refs": []interface{}{"/loadbalancer/projects/test-project/regions/se-sto/l4Routes/test-route"}}},
	}
	if network {
		cfg["backend_network"] = []interface{}{map[string]interface{}{
			"vpc_ref": "/networking/projects/test-project/regions/se-sto/virtualPrivateClouds/custom",
			"subnet": []interface{}{
				map[string]interface{}{"zone": "a", "subnet_ref": "/networking/projects/test-project/regions/se-sto/subnets/custom-a"},
				map[string]interface{}{"zone": "b", "subnet_ref": "/networking/projects/test-project/regions/se-sto/subnets/custom-b"},
			},
		}}
	}
	return cfg
}

func TestLoadBalancerBackendNetworkLifecycle(t *testing.T) {
	for _, custom := range []bool{false, true} {
		name := "default"
		if custom {
			name = "custom"
		}
		t.Run(name, func(t *testing.T) {
			ms := newMockServer()
			defer ms.close()
			config := newTestProviderConfig(t, ms.server.URL)
			res := resourceLoadBalancer()
			d := schema.TestResourceDataRaw(t, res.Schema, loadBalancerTestConfig(custom))
			var want *lbtypes.LoadbalancerSpecBackendNetwork
			if custom {
				if err := json.Unmarshal([]byte(`{"vpcRef":"/networking/projects/test-project/regions/se-sto/virtualPrivateClouds/custom","subnets":[{"zone":"a","subnetRef":"/networking/projects/test-project/regions/se-sto/subnets/custom-a"},{"zone":"b","subnetRef":"/networking/projects/test-project/regions/se-sto/subnets/custom-b"}]}`), &want); err != nil {
					t.Fatal(err)
				}
			}
			checkNetwork := func(got *lbtypes.LoadbalancerSpecBackendNetwork) {
				t.Helper()
				if want == nil || got == nil {
					if want != got {
						t.Errorf("network = %#v, want %#v", got, want)
					}
					return
				}
				if got.VpcRef != want.VpcRef || len(got.Subnets) != len(want.Subnets) {
					t.Errorf("network = %#v, want %#v", got, want)
					return
				}
				for _, expected := range want.Subnets {
					found := false
					for _, actual := range got.Subnets {
						if actual == expected {
							found = true
						}
					}
					if !found {
						t.Errorf("missing subnet %#v in %#v", expected, got)
					}
				}
			}
			lb := mockLoadBalancer("test-lb")
			posts, patches := 0, 0
			path := "/loadbalancer/v1alpha1/projects/test-project/regions/se-sto/loadBalancers"
			ms.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
				var req lbtypes.LoadbalancerRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Error(err)
					w.WriteHeader(400)
					return
				}
				posts++
				checkNetwork(req.Spec.BackendNetwork)
				lb.Spec = req.Spec
				respondJSON(w, http.StatusCreated, lb)
			})
			ms.mux.HandleFunc(path+"/test-lb", func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPatch {
					var patch lbtypes.Loadbalancer
					if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
						t.Error(err)
						w.WriteHeader(400)
						return
					}
					patches++
					checkNetwork(patch.Spec.BackendNetwork)
					lb.Spec = patch.Spec
					lb.Metadata.UserLabels = patch.Metadata.UserLabels
				}
				respondJSON(w, http.StatusOK, lb)
			})
			ctx := context.Background()
			if diags := resourceLoadBalancerCreate(ctx, d, config); diags.HasError() {
				t.Fatal(diags)
			}
			checkNetwork(expandLoadBalancerBackendNetwork(d.Get("backend_network").([]interface{})))
			// Import must discover custom networking without prior configuration.
			imported := res.TestResourceData()
			imported.SetId("test-lb")
			if diags := resourceLoadBalancerRead(ctx, imported, config); diags.HasError() {
				t.Fatal(diags)
			}
			checkNetwork(expandLoadBalancerBackendNetwork(imported.Get("backend_network").([]interface{})))
			if err := d.Set("user_labels", map[string]interface{}{"owner": "test"}); err != nil {
				t.Fatal(err)
			}
			if diags := resourceLoadBalancerUpdate(ctx, d, config); diags.HasError() {
				t.Fatal(diags)
			}
			if posts != 1 || patches != 1 {
				t.Fatalf("posts=%d patches=%d", posts, patches)
			}
			// A subsequent API response with no network must clear stale state.
			lb.Spec.BackendNetwork = nil
			if diags := resourceLoadBalancerRead(ctx, d, config); diags.HasError() {
				t.Fatal(diags)
			}
			if len(d.Get("backend_network").([]interface{})) != 0 {
				t.Fatal("stale backend network retained")
			}
		})
	}
}

func TestLoadBalancerBackendNetworkDiff(t *testing.T) {
	res := resourceLoadBalancer()
	original := schema.TestResourceDataRaw(t, res.Schema, loadBalancerTestConfig(true))
	original.SetId("test-lb")
	for _, change := range []string{"unchanged", "reorder", "vpc", "subnet", "zone", "remove", "add"} {
		t.Run(change, func(t *testing.T) {
			cfg := loadBalancerTestConfig(true)
			block := cfg["backend_network"].([]interface{})[0].(map[string]interface{})
			subnets := block["subnet"].([]interface{})
			state := original.State()
			switch change {
			case "reorder":
				subnets[0], subnets[1] = subnets[1], subnets[0]
			case "vpc":
				block["vpc_ref"] = "/networking/projects/test-project/regions/se-sto/virtualPrivateClouds/other"
			case "subnet":
				subnets[0].(map[string]interface{})["subnet_ref"] = "/networking/projects/test-project/regions/se-sto/subnets/other"
			case "zone":
				subnets[0].(map[string]interface{})["zone"] = "c"
			case "remove":
				delete(cfg, "backend_network")
			case "add":
				d := schema.TestResourceDataRaw(t, res.Schema, loadBalancerTestConfig(false))
				d.SetId("test-lb")
				state = d.State()
			}
			diff, err := res.Diff(context.Background(), state, terraform.NewResourceConfigRaw(cfg), nil)
			if err != nil {
				t.Fatal(err)
			}
			want := change != "unchanged" && change != "reorder"
			if got := diff != nil && diff.RequiresNew(); got != want {
				t.Fatalf("RequiresNew = %v, want %v; diff=%#v", got, want, diff)
			}
			if !want && diff != nil {
				for key, attr := range diff.Attributes {
					if strings.HasPrefix(key, "backend_network.") {
						t.Fatalf("unexpected network drift: %s = %#v", key, attr)
					}
				}
			}
		})
	}
}

func TestLoadBalancerBackendNetworkValidation(t *testing.T) {
	for _, invalid := range []string{"valid", "missing_vpc", "missing_subnet", "empty_subnet", "invalid_zone", "empty_vpc", "missing_subnet_ref"} {
		t.Run(invalid, func(t *testing.T) {
			cfg := loadBalancerTestConfig(true)
			block := cfg["backend_network"].([]interface{})[0].(map[string]interface{})
			switch invalid {
			case "missing_vpc":
				delete(block, "vpc_ref")
			case "missing_subnet":
				delete(block, "subnet")
			case "empty_subnet":
				block["subnet"] = []interface{}{}
			case "invalid_zone":
				block["subnet"].([]interface{})[0].(map[string]interface{})["zone"] = "d"
			case "empty_vpc":
				block["vpc_ref"] = " "
			case "missing_subnet_ref":
				delete(block["subnet"].([]interface{})[0].(map[string]interface{}), "subnet_ref")
			}
			diags := resourceLoadBalancer().Validate(terraform.NewResourceConfigRaw(cfg))
			if diags.HasError() != (invalid != "valid") {
				t.Fatalf("unexpected validation: %v", diags)
			}
		})
	}
}
