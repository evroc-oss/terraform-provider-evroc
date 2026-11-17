// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	lbtypes "github.com/evroc-oss/evroc-go-sdk/types/loadbalancer"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func httpRouteTestConfig() map[string]interface{} {
	return map[string]interface{}{
		"name": "test-route", "hostnames": []interface{}{"api.example.com"},
		"user_labels": map[string]interface{}{"team": "api", "owner": "platform"},
		"rule": []interface{}{
			map[string]interface{}{
				"name": "api", "backend_service_ref": "/loadbalancer/projects/test-project/regions/se-sto/backendServices/api",
				"match": []interface{}{
					map[string]interface{}{
						"method": "GET", "path": []interface{}{map[string]interface{}{"type": "Exact", "value": "/api"}},
						"header":      []interface{}{map[string]interface{}{"name": "X-Version", "value": "v2"}},
						"query_param": []interface{}{map[string]interface{}{"name": "preview", "value": "yes"}},
					},
					map[string]interface{}{"method": "HEAD", "path": []interface{}{map[string]interface{}{}}},
				},
			},
			map[string]interface{}{"name": "fallback", "backend_service_ref": "/loadbalancer/projects/test-project/regions/se-sto/backendServices/fallback"},
		},
	}
}

func TestLBHTTPRouteMapping(t *testing.T) {
	ms, config := newLBTest(t)
	res := resourceLBHTTPRoute()
	cfg := httpRouteTestConfig()
	d := schema.TestResourceDataRaw(t, res.Schema, cfg)
	const spec = `{"hostnames":["api.example.com"],"rules":[{"name":"api","backendServiceRef":"/loadbalancer/projects/test-project/regions/se-sto/backendServices/api","matches":[{"method":"GET","path":{"type":"Exact","value":"/api"},"headers":[{"name":"X-Version","value":"v2"}],"queryParams":[{"name":"preview","value":"yes"}]},{"method":"HEAD","path":{"type":"PathPrefix","value":"/"}}]},{"name":"fallback","backendServiceRef":"/loadbalancer/projects/test-project/regions/se-sto/backendServices/fallback"}]}`
	route := &lbtypes.Httproute{Metadata: mockLoadBalancer("test-route").Metadata}
	requireLBNoError(t, json.Unmarshal([]byte(spec), &route.Spec))
	path := "/loadbalancer/v1alpha2/projects/test-project/regions/se-sto/httpRoutes"
	created := false
	ms.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		var body struct{ Spec json.RawMessage }
		if !decodeLBRequest(t, w, r, &body) {
			return
		}
		created = true
		checkLBJSON(t, body.Spec, spec)
		respondJSON(w, http.StatusCreated, route)
	})
	missing := false
	ms.mux.HandleFunc(path+"/test-route", func(w http.ResponseWriter, r *http.Request) {
		if missing {
			respondJSON(w, http.StatusNotFound, map[string]string{"reason": "not found"})
			return
		}
		respondJSON(w, http.StatusOK, route)
	})

	ctx := context.Background()
	requireLBSuccess(t, resourceLBHTTPRouteCreate(ctx, d, config))
	if !created {
		t.Fatal("create did not POST route")
	}
	d = res.TestResourceData()
	d.SetId("test-route")
	requireLBSuccess(t, resourceLBHTTPRouteRead(ctx, d, config))
	assertAttrs(t, d, map[string]interface{}{
		"rule.0.name": "api", "rule.1.name": "fallback",
		"rule.0.backend_service_ref": "/loadbalancer/projects/test-project/regions/se-sto/backendServices/api",
		"rule.0.match.0.method":      "GET", "rule.0.match.1.method": "HEAD",
		"rule.0.match.0.path.0.type": "Exact", "rule.0.match.0.path.0.value": "/api",
		"rule.0.match.1.path.0.type": "PathPrefix", "rule.0.match.1.path.0.value": "/",
	})
	if !d.Get("hostnames").(*schema.Set).Contains("api.example.com") {
		t.Error("read omitted hostname")
	}
	match := d.Get("rule.0.match.0").(map[string]interface{})
	for _, field := range []string{"header", "query_param"} {
		checkLBJSON(t, match[field].(*schema.Set).List(), map[string]string{
			"header": `[{"name":"X-Version","value":"v2"}]`, "query_param": `[{"name":"preview","value":"yes"}]`,
		}[field])
	}
	checkLBNoDiff(t, res, d, cfg, config, "rule.", "hostnames.")
	ds := schema.TestResourceDataRaw(t, dataSourceLBHTTPRoute().Schema, map[string]interface{}{"name": "test-route"})
	requireLBSuccess(t, dataSourceLBHTTPRoute().ReadContext(ctx, ds, config))
	assertAttrs(t, ds, map[string]interface{}{"rule.0.name": "api", "rule.1.name": "fallback"})

	// A later observation must replace optional collections, including removed matches.
	route.Spec.Hostnames = nil
	route.Spec.Rules = []lbtypes.HttprouteSpecRulesItem{{Name: "fallback", BackendServiceRef: "/loadbalancer/projects/test-project/regions/se-sto/backendServices/fallback"}}
	requireLBSuccess(t, resourceLBHTTPRouteRead(ctx, d, config))
	if d.Get("hostnames").(*schema.Set).Len() != 0 || len(d.Get("rule").([]interface{})) != 1 || len(d.Get("rule.0.match").([]interface{})) != 0 {
		t.Fatal("read retained stale hostnames, rules or matches")
	}
	missing = true
	if diags := resourceLBHTTPRouteRead(ctx, d, config); diags.HasError() || d.Id() != "" {
		t.Fatalf("missing resource read = %v, ID=%q", diags, d.Id())
	}
	if diags := dataSourceLBHTTPRoute().ReadContext(ctx, ds, config); !diags.HasError() {
		t.Fatal("missing data source must report an error")
	}
}

func TestLBHTTPRoutePatchRemovals(t *testing.T) {
	for _, tc := range []struct {
		name, labels string
		desired      map[string]interface{}
	}{
		{"partial labels", `{"team":null,"owner":"platform"}`, map[string]interface{}{"owner": "platform"}},
		{"all labels", `{"team":null,"owner":null}`, map[string]interface{}{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ms, config := newLBTest(t)
			route := &lbtypes.Httproute{Metadata: mockLoadBalancer("test-route").Metadata}
			labels := lbtypes.UserLabels{"team": "api", "owner": "platform"}
			route.Metadata.UserLabels = &labels
			requireLBNoError(t, json.Unmarshal([]byte(`{"hostnames":["old.example.com"],"rules":[{"name":"old","backendServiceRef":"/loadbalancer/projects/test-project/regions/se-sto/backendServices/old","matches":[{"method":"GET"}]}]}`), &route.Spec))
			version := "1"
			route.Metadata.ResourceVersion = &version
			d := schema.TestResourceDataRaw(t, resourceLBHTTPRoute().Schema, map[string]interface{}{
				"name": "test-route", "user_labels": tc.desired,
				"rule": []interface{}{map[string]interface{}{"name": "fallback", "backend_service_ref": "/loadbalancer/projects/test-project/regions/se-sto/backendServices/fallback"}},
			})
			d.SetId("test-route")
			patched := false
			ms.mux.HandleFunc("/loadbalancer/v1alpha2/projects/test-project/regions/se-sto/httpRoutes/test-route", func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPatch {
					var body json.RawMessage
					if !decodeLBRequest(t, w, r, &body) {
						return
					}
					patched = true
					checkLBJSON(t, body, fmt.Sprintf(`{"metadata":{"resourceVersion":"1","userLabels":%s},"spec":{"hostnames":null,"rules":[{"name":"fallback","backendServiceRef":"/loadbalancer/projects/test-project/regions/se-sto/backendServices/fallback"}]}}`, tc.labels))
				}
				respondJSON(w, http.StatusOK, route)
			})
			requireLBSuccess(t, resourceLBHTTPRouteUpdate(context.Background(), d, config))
			if !patched {
				t.Fatal("update did not PATCH route")
			}
		})
	}
}

func TestLBHTTPRouteValidation(t *testing.T) {
	for _, name := range []string{"valid", "duplicate rules", "invalid method", "invalid path type", "relative path", "duplicate header names"} {
		t.Run(name, func(t *testing.T) {
			cfg := httpRouteTestConfig()
			rules := cfg["rule"].([]interface{})
			first := rules[0].(map[string]interface{})
			match := first["match"].([]interface{})[0].(map[string]interface{})
			switch name {
			case "duplicate rules":
				rules[1].(map[string]interface{})["name"] = "api"
			case "invalid method":
				match["method"] = "FETCH"
			case "invalid path type":
				match["path"].([]interface{})[0].(map[string]interface{})["type"] = "Regex"
			case "relative path":
				match["path"].([]interface{})[0].(map[string]interface{})["value"] = "api"
			case "duplicate header names":
				match["header"] = []interface{}{map[string]interface{}{"name": "X-Version", "value": "v1"}, map[string]interface{}{"name": "x-version", "value": "v2"}}
			}
			checkLBValidation(t, resourceLBHTTPRoute(), cfg, name != "valid")
		})
	}
}

func TestLBHTTPRouteServerOwnedLimits(t *testing.T) {
	t.Run("collections and lengths", func(t *testing.T) {
		cfg := httpRouteTestConfig()
		var hosts, rules, values, matches []interface{}
		for i := 0; i < 17; i++ {
			name := fmt.Sprintf("%s%d", strings.Repeat("a", 257), i)
			hosts = append(hosts, name+".example.com")
			values = append(values, map[string]interface{}{"name": name, "value": strings.Repeat("v", 4097)})
			rules = append(rules, map[string]interface{}{"name": name, "backend_service_ref": "/loadbalancer/projects/test-project/regions/se-sto/backendServices/api"})
		}
		for i := 0; i < 65; i++ {
			matches = append(matches, map[string]interface{}{"path": []interface{}{map[string]interface{}{"value": "/" + strings.Repeat("a", 1024)}}})
		}
		matches[0].(map[string]interface{})["header"] = values
		matches[0].(map[string]interface{})["query_param"] = values
		rules[0].(map[string]interface{})["match"] = matches
		cfg["hostnames"], cfg["rule"] = hosts, rules

		checkLBValidation(t, resourceLBHTTPRoute(), cfg, false)
	})
	t.Run("aggregate matches", func(t *testing.T) {
		cfg := httpRouteTestConfig()
		var manyRules []interface{}
		for i := 0; i < 3; i++ {
			var matches []interface{}
			for j := 0; j < 43; j++ {
				matches = append(matches, map[string]interface{}{"path": []interface{}{map[string]interface{}{"value": fmt.Sprintf("/%d", j)}}})
			}
			manyRules = append(manyRules, map[string]interface{}{"name": fmt.Sprintf("rule-%d", i), "backend_service_ref": "/loadbalancer/projects/test-project/regions/se-sto/backendServices/api", "match": matches})
		}
		cfg["rule"] = manyRules

		checkLBValidation(t, resourceLBHTTPRoute(), cfg, false)
	})
}

func TestLBHTTPRouteUnknownRuleNames(t *testing.T) {
	res := resourceLBHTTPRoute()
	var rules []cty.Value
	for _, backend := range []string{"api", "fallback"} {
		rules = append(rules, cty.ObjectVal(map[string]cty.Value{
			"name":                cty.UnknownVal(cty.String),
			"backend_service_ref": cty.StringVal("/loadbalancer/projects/test-project/regions/se-sto/backendServices/" + backend),
		}))
	}
	raw := cty.ObjectVal(map[string]cty.Value{"name": cty.StringVal("test-route"), "rule": cty.ListVal(rules)})
	cfg := terraform.NewResourceConfigShimmed(raw, res.CoreConfigSchema())

	// The SDK's plan RPC supplies raw configuration through the prior state.
	_, err := res.Diff(context.Background(), &terraform.InstanceState{RawConfig: raw}, cfg, nil)
	requireLBNoError(t, err)
}

func TestLBHTTPRouteUpdateRetriesConflict(t *testing.T) {
	ms, config := newLBTest(t)
	d := schema.TestResourceDataRaw(t, resourceLBHTTPRoute().Schema, map[string]interface{}{
		"name": "test-route", "user_labels": map[string]interface{}{"team": "api"},
		"rule": []interface{}{map[string]interface{}{"name": "fallback", "backend_service_ref": "/loadbalancer/projects/test-project/regions/se-sto/backendServices/fallback"}},
	})
	d.SetId("test-route")
	route := &lbtypes.Httproute{Metadata: mockLoadBalancer("test-route").Metadata, Spec: lbtypes.HttprouteSpec{Rules: []lbtypes.HttprouteSpecRulesItem{{Name: "fallback", BackendServiceRef: "/loadbalancer/projects/test-project/regions/se-sto/backendServices/fallback"}}}}
	version := "1"
	route.Metadata.ResourceVersion = &version
	patches := 0
	ms.mux.HandleFunc("/loadbalancer/v1alpha2/projects/test-project/regions/se-sto/httpRoutes/test-route", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			patches++
			var patch struct {
				Metadata struct {
					ResourceVersion string `json:"resourceVersion"`
				} `json:"metadata"`
			}
			if !decodeLBRequest(t, w, r, &patch) {
				return
			}
			if patch.Metadata.ResourceVersion != *route.Metadata.ResourceVersion {
				t.Errorf("PATCH resourceVersion = %q, want fresh %q", patch.Metadata.ResourceVersion, *route.Metadata.ResourceVersion)
			}
			if patches == 1 {
				version = "2"
				respondJSON(w, http.StatusConflict, map[string]string{"reason": "the object has been modified"})
				return
			}
		}
		respondJSON(w, http.StatusOK, route)
	})
	requireLBSuccess(t, resourceLBHTTPRouteUpdate(context.Background(), d, config))
	if patches != 2 {
		t.Fatalf("PATCHes = %d, want conflict retry", patches)
	}
}
