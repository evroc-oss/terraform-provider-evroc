// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestLBBackendServiceAPIDefaultsNoDiff(t *testing.T) {
	res := resourceLBBackendService()
	cfg := func() map[string]interface{} {
		return map[string]interface{}{
			"name":             "test-svc",
			"port":             80,
			"backend_pool_ref": "/loadbalancer/projects/test-project/regions/se-sto/backendPools/test-pool",
			"health_check": []interface{}{map[string]interface{}{
				"http": []interface{}{map[string]interface{}{"path": "/healthz"}},
			}},
		}
	}
	// State as read back after the API filled in its defaults.
	state := cfg()
	state["ip_protocol_selection"] = "IPv4"
	hc := state["health_check"].([]interface{})[0].(map[string]interface{})
	hc["target_port"] = 80
	hc["interval"] = "5s"
	hc["timeout"] = "3s"
	hc["healthy_threshold"] = 2
	hc["unhealthy_threshold"] = 3
	hc["http"].([]interface{})[0].(map[string]interface{})["expected_statuses"] = []interface{}{200}
	d := schema.TestResourceDataRaw(t, res.Schema, state)
	d.SetId("test-svc")

	diff, err := res.Diff(context.Background(), d.State(), terraform.NewResourceConfigRaw(cfg()), nil)
	if err != nil {
		t.Fatal(err)
	}
	if diff != nil {
		for key, attr := range diff.Attributes {
			if key == "system_labels.%" || key == "backends.#" {
				continue // computed-only, not set in the test state
			}
			t.Errorf("unexpected diff: %s = %#v", key, attr)
		}
	}
}

func TestLBBackendServiceTargetPortDefault(t *testing.T) {
	res := resourceLBBackendService()
	cfg := func(port, targetPort int) map[string]interface{} {
		hc := map[string]interface{}{"tcp": []interface{}{map[string]interface{}{}}}
		if targetPort != 0 {
			hc["target_port"] = targetPort
		}
		return map[string]interface{}{
			"name":             "test-svc",
			"port":             port,
			"backend_pool_ref": "/loadbalancer/projects/test-project/regions/se-sto/backendPools/test-pool",
			"health_check":     []interface{}{hc},
		}
	}

	t.Run("expand defaults to service port", func(t *testing.T) {
		d := schema.TestResourceDataRaw(t, res.Schema, cfg(8080, 0))
		if hc := expandHealthCheck(d); hc.TargetPort == nil || *hc.TargetPort != 8080 {
			t.Fatalf("TargetPort = %v, want 8080", hc.TargetPort)
		}
		d = schema.TestResourceDataRaw(t, res.Schema, cfg(8080, 9000))
		if hc := expandHealthCheck(d); hc.TargetPort == nil || *hc.TargetPort != 9000 {
			t.Fatalf("TargetPort = %v, want 9000", hc.TargetPort)
		}
	})

	t.Run("explicit zero rejected", func(t *testing.T) {
		c := cfg(8080, 0)
		c["health_check"].([]interface{})[0].(map[string]interface{})["target_port"] = 0
		if diags := res.Validate(terraform.NewResourceConfigRaw(c)); !diags.HasError() {
			t.Fatal("expected target_port = 0 to be rejected")
		}
	})

	for _, tc := range []struct {
		name                 string
		statePort, stateTP   int
		configPort, configTP int
		wantDiff             bool
	}{
		{"defaulted, unchanged", 8080, 8080, 8080, 0, false},
		{"port changed moves default", 8080, 8080, 9090, 0, true},
		// A stored 0 and an unset value look the same to the SDK, so a service
		// created before the default is repaired by setting target_port.
		{"stored zero, still unset", 8080, 0, 8080, 0, false},
		{"stored zero, explicitly set", 8080, 0, 8080, 8080, true},
		{"explicit equal to port", 8080, 8080, 8080, 8080, false},
		{"explicit removed", 8080, 9000, 8080, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := schema.TestResourceDataRaw(t, res.Schema, cfg(tc.statePort, 0))
			d.SetId("test-svc")
			if err := d.Set("health_check", []interface{}{map[string]interface{}{"target_port": tc.stateTP, "tcp": []interface{}{map[string]interface{}{}}}}); err != nil {
				t.Fatal(err)
			}
			diff, err := res.Diff(context.Background(), d.State(), terraform.NewResourceConfigRaw(cfg(tc.configPort, tc.configTP)), nil)
			if err != nil {
				t.Fatal(err)
			}
			got := diff != nil && diff.Attributes["health_check.0.target_port"] != nil
			if got != tc.wantDiff {
				t.Fatalf("target_port diff = %v, want %v; diff=%#v", got, tc.wantDiff, diff)
			}
		})
	}
}
