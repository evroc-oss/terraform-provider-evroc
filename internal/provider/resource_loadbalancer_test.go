// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"testing"

	lbtypes "github.com/evroc-oss/evroc-go-sdk/types/loadbalancer"
	"github.com/hashicorp/go-cty/cty"
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
			path := "/loadbalancer/v1alpha2/projects/test-project/regions/se-sto/loadBalancers"
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

func TestLoadBalancerUpdateRetriesConflict(t *testing.T) {
	for _, tc := range []struct {
		name        string
		failStatus  int
		wantPatches int
		wantErr     bool
	}{
		{"conflict is retried", http.StatusConflict, 2, false},
		{"other errors are not retried", http.StatusBadRequest, 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ms := newMockServer()
			defer ms.close()
			config := newTestProviderConfig(t, ms.server.URL)
			d := schema.TestResourceDataRaw(t, resourceLoadBalancer().Schema, loadBalancerTestConfig(false))
			d.SetId("test-lb")
			lb := mockLoadBalancer("test-lb")
			gets, patches := 0, 0
			ms.mux.HandleFunc("/loadbalancer/v1alpha2/projects/test-project/regions/se-sto/loadBalancers/test-lb", func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPatch {
					patches++
					if patches == 1 {
						respondJSON(w, tc.failStatus, map[string]string{"reason": "the object has been modified"})
						return
					}
				} else {
					gets++
				}
				respondJSON(w, http.StatusOK, lb)
			})
			if err := d.Set("user_labels", map[string]interface{}{"owner": "test"}); err != nil {
				t.Fatal(err)
			}
			diags := resourceLoadBalancerUpdate(context.Background(), d, config)
			if diags.HasError() != tc.wantErr {
				t.Fatalf("diags = %v, wantErr %v", diags, tc.wantErr)
			}
			if patches != tc.wantPatches {
				t.Fatalf("patches = %d, want %d", patches, tc.wantPatches)
			}
			// Each attempt must re-read the object to pick up a fresh resourceVersion.
			if !tc.wantErr && gets < 2 {
				t.Fatalf("gets = %d, want re-read before retrying PATCH", gets)
			}
		})
	}
}

func TestLoadBalancerL7ListenerMapping(t *testing.T) {
	const routeRef = "/loadbalancer/projects/test-project/regions/se-sto/httpRoutes/test-route"
	const secondRouteRef = "/loadbalancer/projects/test-project/regions/se-sto/httpRoutes/test-route-two"
	const certificateRef = "/loadbalancer/projects/test-project/regions/se-sto/loadBalancerCertificates/web"
	const httpListener = `[{"name":"web","port":443,"protocol":"HTTP","routeRefs":["/loadbalancer/projects/test-project/regions/se-sto/httpRoutes/test-route","/loadbalancer/projects/test-project/regions/se-sto/httpRoutes/test-route-two"]}]`
	checkListeners := func(t *testing.T, listeners []map[string]interface{}, want string) {
		t.Helper()
		for _, listener := range listeners {
			refs, _ := listener["routeRefs"].([]interface{})
			sort.Slice(refs, func(i, j int) bool { return refs[i].(string) < refs[j].(string) })
		}
		checkLBJSON(t, listeners, want)
	}
	for _, protocol := range []string{"HTTP", "HTTPS"} {
		t.Run(protocol, func(t *testing.T) {
			ms, config := newLBTest(t)
			res := resourceLoadBalancer()
			cfg := loadBalancerTestConfig(false)
			listener := map[string]interface{}{"name": "web", "port": 443, "protocol": protocol, "route_refs": []interface{}{routeRef, secondRouteRef}}
			wire := httpListener
			if protocol == "HTTPS" {
				listener["tls"] = []interface{}{map[string]interface{}{"certificate_ref": certificateRef}}
				wire = `[{"name":"web","port":443,"protocol":"HTTPS","routeRefs":["/loadbalancer/projects/test-project/regions/se-sto/httpRoutes/test-route","/loadbalancer/projects/test-project/regions/se-sto/httpRoutes/test-route-two"],"tls":{"certificateRef":"/loadbalancer/projects/test-project/regions/se-sto/loadBalancerCertificates/web"}}]`
			}
			cfg["listener"] = []interface{}{listener}
			d := schema.TestResourceDataRaw(t, res.Schema, cfg)
			lb := mockLoadBalancer("test-lb")
			requireLBNoError(t, json.Unmarshal([]byte(wire), &lb.Spec.Listeners))
			path := "/loadbalancer/v1alpha2/projects/test-project/regions/se-sto/loadBalancers"
			created, patched := false, false
			ms.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					Spec struct{ Listeners []map[string]interface{} }
				}
				if !decodeLBRequest(t, w, r, &body) {
					return
				}
				created = true
				checkListeners(t, body.Spec.Listeners, wire)
				respondJSON(w, http.StatusCreated, lb)
			})
			ms.mux.HandleFunc(path+"/test-lb", func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPatch {
					var body struct {
						Spec struct{ Listeners []map[string]interface{} }
					}
					if !decodeLBRequest(t, w, r, &body) {
						return
					}
					patched = true
					checkListeners(t, body.Spec.Listeners, httpListener)
					lb.Spec.Listeners = nil
					if err := json.Unmarshal([]byte(httpListener), &lb.Spec.Listeners); err != nil {
						t.Error(err)
					}
				}
				respondJSON(w, http.StatusOK, lb)
			})

			ctx := context.Background()
			requireLBSuccess(t, resourceLoadBalancerCreate(ctx, d, config))
			if !created {
				t.Fatal("create did not POST load balancer")
			}
			d = res.TestResourceData()
			d.SetId("test-lb")
			requireLBSuccess(t, resourceLoadBalancerRead(ctx, d, config))
			checkLBNoDiff(t, res, d, cfg, config, "listener.")
			ds := schema.TestResourceDataRaw(t, dataSourceLoadBalancer().Schema, map[string]interface{}{"name": "test-lb"})
			requireLBSuccess(t, dataSourceLoadBalancerRead(ctx, ds, config))
			got := d.Get("listener").(*schema.Set).List()[0].(map[string]interface{})
			observed := ds.Get("listener").([]interface{})[0].(map[string]interface{})
			// The resource and data source use different collection schemas.
			refs := got["route_refs"].(*schema.Set)
			observedRefs := schema.NewSet(schema.HashString, observed["route_refs"].([]interface{}))
			if refs.Len() != 2 || observedRefs.Len() != 2 || !refs.Contains(routeRef) || !refs.Contains(secondRouteRef) || !observedRefs.Contains(routeRef) || !observedRefs.Contains(secondRouteRef) {
				t.Error("read omitted HTTP route references")
			}
			for _, read := range []map[string]interface{}{got, observed} {
				if read["protocol"] != protocol {
					t.Errorf("read protocol = %v", read["protocol"])
				}
				tls := read["tls"].([]interface{})
				if protocol == "HTTPS" {
					if len(tls) != 1 || tls[0].(map[string]interface{})["certificate_ref"] != certificateRef {
						t.Errorf("read TLS = %#v", tls)
					}
				} else if len(tls) != 0 {
					t.Errorf("unexpected TLS = %#v", tls)
				}
			}
			if protocol == "HTTPS" {
				listener["protocol"] = "HTTP"
				delete(listener, "tls")
				diff, err := res.Diff(ctx, d.State(), terraform.NewResourceConfigRaw(cfg), config)
				requireLBNoError(t, err)
				_, diags := res.Apply(ctx, d.State(), diff, config)
				requireLBSuccess(t, diags)
				requireLBSuccess(t, resourceLoadBalancerRead(ctx, d, config))
				if !patched {
					t.Fatal("update did not PATCH load balancer")
				}
				if got := d.Get("listener").(*schema.Set).List()[0].(map[string]interface{})["tls"]; len(got.([]interface{})) != 0 {
					t.Fatal("HTTP update retained TLS")
				}
			}
		})
	}
}

func TestLoadBalancerListenerValidation(t *testing.T) {
	for _, tc := range []struct {
		name, protocol string
		tls, wantError bool
	}{
		{"TCP", "TCP", false, false},
		{"HTTP", "HTTP", false, false},
		{"HTTPS", "HTTPS", true, false},
		{"HTTPS requires a certificate", "HTTPS", false, true},
		{"TCP forbids TLS", "TCP", true, true},
		{"HTTP forbids TLS", "HTTP", true, true},
		{"TLS protocol is unsupported", "TLS", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := loadBalancerTestConfig(false)
			listener := cfg["listener"].([]interface{})[0].(map[string]interface{})
			listener["protocol"] = tc.protocol
			if tc.tls {
				listener["tls"] = []interface{}{map[string]interface{}{"certificate_ref": "/loadbalancer/projects/test-project/regions/se-sto/loadBalancerCertificates/web"}}
			}
			checkLBValidation(t, resourceLoadBalancer(), cfg, tc.wantError)
		})
	}
}

func TestLoadBalancerListenerIdentityValidation(t *testing.T) {
	for _, field := range []string{"name", "port"} {
		t.Run("duplicate "+field, func(t *testing.T) {
			cfg := loadBalancerTestConfig(false)
			first := cfg["listener"].([]interface{})[0].(map[string]interface{})
			first["name"] = "one"
			second := map[string]interface{}{"name": "two", "port": 9346, "protocol": "TCP", "route_refs": first["route_refs"]}
			second[field] = first[field]
			cfg["listener"] = []interface{}{first, second}
			checkLBValidation(t, resourceLoadBalancer(), cfg, true)
		})
	}
}

func TestLoadBalancerListenerUnknownValues(t *testing.T) {
	for _, field := range []string{"port", "protocol", "name"} {
		t.Run(field, func(t *testing.T) {
			res := resourceLoadBalancer()
			var listeners []cty.Value
			for i, name := range []string{"one", "two"} {
				listener := map[string]cty.Value{
					"name": cty.StringVal(name), "port": cty.NumberIntVal(int64(443 + i)),
					"protocol":   cty.StringVal("TCP"),
					"route_refs": cty.SetVal([]cty.Value{cty.StringVal("/loadbalancer/projects/test-project/regions/se-sto/l4Routes/test-route")}),
				}
				listener[field] = cty.UnknownVal(listener[field].Type())
				if field == "protocol" {
					listener["route_refs"] = cty.SetVal([]cty.Value{cty.StringVal("/loadbalancer/projects/test-project/regions/se-sto/httpRoutes/test-route")})
					listener["tls"] = cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
						"certificate_ref": cty.StringVal("/loadbalancer/projects/test-project/regions/se-sto/loadBalancerCertificates/web"),
					})})
				}
				listeners = append(listeners, cty.ObjectVal(listener))
			}
			raw := cty.ObjectVal(map[string]cty.Value{
				"name": cty.StringVal("test-lb"), "public_ip_ref": cty.StringVal("/networking/projects/test-project/regions/se-sto/publicIPs/test-pip"),
				"listener": cty.SetVal(listeners),
			})
			cfg := terraform.NewResourceConfigShimmed(raw, res.CoreConfigSchema())

			// The SDK's plan RPC supplies raw configuration through the prior state.
			_, err := res.Diff(context.Background(), &terraform.InstanceState{RawConfig: raw}, cfg, nil)
			requireLBNoError(t, err)
		})
	}
}

func TestLoadBalancerListenerMultipleRoutes(t *testing.T) {
	for _, protocol := range []string{"TCP", "HTTP", "HTTPS"} {
		t.Run(protocol, func(t *testing.T) {
			cfg := loadBalancerTestConfig(false)
			listener := cfg["listener"].([]interface{})[0].(map[string]interface{})
			listener["protocol"] = protocol
			routeKind := "l4Routes"
			if protocol != "TCP" {
				routeKind = "httpRoutes"
			}
			listener["route_refs"] = []interface{}{
				"/loadbalancer/projects/test-project/regions/se-sto/" + routeKind + "/one",
				"/loadbalancer/projects/test-project/regions/se-sto/" + routeKind + "/two",
			}
			if protocol == "HTTPS" {
				listener["tls"] = []interface{}{map[string]interface{}{"certificate_ref": "/loadbalancer/projects/test-project/regions/se-sto/loadBalancerCertificates/web"}}
			}
			checkLBValidation(t, resourceLoadBalancer(), cfg, false)
		})
	}
}
