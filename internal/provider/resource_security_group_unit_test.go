// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"testing"

	networkingtypes "github.com/evroc-oss/evroc-go-sdk/types/networking"
)

func TestExpandSecurityGroupRules(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)

	t.Run("empty rules", func(t *testing.T) {
		result := expandSecurityGroupRules(config.Client, []interface{}{})
		if len(result) != 0 {
			t.Errorf("expected 0 rules, got %d", len(result))
		}
	})

	t.Run("null entries from set diff are skipped", func(t *testing.T) {
		input := []interface{}{
			map[string]interface{}{
				"name":                  "allow-ssh",
				"direction":             "Ingress",
				"protocol":              "TCP",
				"port":                  22,
				"end_port":              0,
				"remote_ip":             "0.0.0.0/0",
				"remote_security_group": "",
				"remote_subnet":         "",
			},
			map[string]interface{}{
				"name":                  "",
				"direction":             "",
				"protocol":              "",
				"port":                  0,
				"end_port":              0,
				"remote_ip":             "",
				"remote_security_group": "",
				"remote_subnet":         "",
			},
		}

		result := expandSecurityGroupRules(config.Client, input)
		if len(result) != 1 {
			t.Errorf("expected 1 rule (null entry skipped), got %d", len(result))
		}
		if len(result) > 0 && *result[0].Name != "allow-ssh" {
			t.Errorf("expected surviving rule to be allow-ssh, got %q", *result[0].Name)
		}
	})

	t.Run("single ingress rule", func(t *testing.T) {
		input := []interface{}{
			map[string]interface{}{
				"name":      "allow-ssh",
				"direction": "Ingress",
				"protocol":  "TCP",
				"port":      22,
				"end_port":  0,
				"remote_ip": "0.0.0.0/0",
			},
		}

		result := expandSecurityGroupRules(config.Client, input)
		if len(result) != 1 {
			t.Fatalf("expected 1 rule, got %d", len(result))
		}

		rule := result[0]
		if *rule.Name != "allow-ssh" {
			t.Errorf("expected name %q, got %q", "allow-ssh", *rule.Name)
		}
		if rule.Direction != "Ingress" {
			t.Errorf("expected direction %q, got %q", "Ingress", rule.Direction)
		}
		if rule.Protocol == nil || *rule.Protocol != "TCP" {
			t.Error("expected protocol TCP")
		}
		if rule.Port == nil || *rule.Port != 22 {
			t.Error("expected port 22")
		}
		if rule.Remote.Address == nil || rule.Remote.Address.IpAddressOrCIDR != "0.0.0.0/0" {
			t.Error("expected remote IP 0.0.0.0/0")
		}
	})

	t.Run("egress rule without remote IP", func(t *testing.T) {
		input := []interface{}{
			map[string]interface{}{
				"name":      "allow-all-egress",
				"direction": "Egress",
				"protocol":  "TCP",
				"port":      0,
				"end_port":  0,
				"remote_ip": "",
			},
		}

		result := expandSecurityGroupRules(config.Client, input)
		if len(result) != 1 {
			t.Fatalf("expected 1 rule, got %d", len(result))
		}

		rule := result[0]
		if rule.Direction != "Egress" {
			t.Errorf("expected direction Egress, got %q", rule.Direction)
		}
		// Port 0 should not be set (means "all ports")
		if rule.Port != nil {
			t.Error("expected port to be nil for port 0")
		}
		// No remote IP set
		if rule.Remote.Address != nil {
			t.Error("expected no remote address for empty remote_ip")
		}
	})

	t.Run("rule with port range", func(t *testing.T) {
		input := []interface{}{
			map[string]interface{}{
				"name":      "allow-high-ports",
				"direction": "Ingress",
				"protocol":  "TCP",
				"port":      8000,
				"end_port":  9000,
				"remote_ip": "10.0.0.0/8",
			},
		}

		result := expandSecurityGroupRules(config.Client, input)
		if len(result) != 1 {
			t.Fatalf("expected 1 rule, got %d", len(result))
		}

		rule := result[0]
		if rule.Port == nil || *rule.Port != 8000 {
			t.Error("expected port 8000")
		}
		if rule.EndPort == nil || *rule.EndPort != 9000 {
			t.Error("expected end_port 9000")
		}
	})

	t.Run("multiple rules", func(t *testing.T) {
		input := []interface{}{
			map[string]interface{}{
				"name":      "ssh",
				"direction": "Ingress",
				"protocol":  "TCP",
				"port":      22,
				"end_port":  0,
				"remote_ip": "0.0.0.0/0",
			},
			map[string]interface{}{
				"name":      "http",
				"direction": "Ingress",
				"protocol":  "TCP",
				"port":      80,
				"end_port":  0,
				"remote_ip": "0.0.0.0/0",
			},
			map[string]interface{}{
				"name":      "icmp",
				"direction": "Ingress",
				"protocol":  "ICMP",
				"port":      0,
				"end_port":  0,
				"remote_ip": "",
			},
		}

		result := expandSecurityGroupRules(config.Client, input)
		if len(result) != 3 {
			t.Fatalf("expected 3 rules, got %d", len(result))
		}
	})

	t.Run("rule with remote_security_group plain name resolves to FQID", func(t *testing.T) {
		input := []interface{}{
			map[string]interface{}{
				"name":                  "allow-from-lb",
				"direction":             "Ingress",
				"protocol":              "TCP",
				"port":                  443,
				"end_port":              0,
				"remote_ip":             "",
				"remote_security_group": "lb-sg",
				"remote_subnet":         "",
			},
		}

		result := expandSecurityGroupRules(config.Client, input)
		if len(result) != 1 {
			t.Fatalf("expected 1 rule, got %d", len(result))
		}

		rule := result[0]
		if rule.Remote.SecurityGroupRef == nil {
			t.Fatal("expected remote security group ref to be set")
		}
		ref := *rule.Remote.SecurityGroupRef
		if !isFQID(ref) {
			t.Errorf("expected FQID, got plain name %q", ref)
		}
		if ref != string(config.Client.Networking().SecurityGroupRef("lb-sg")) {
			t.Errorf("expected resolved FQID, got %q", ref)
		}
	})

	t.Run("rule with remote_security_group FQID passes through", func(t *testing.T) {
		fqid := "/networking/projects/other-proj/regions/se-sto/securityGroups/lb-sg"
		input := []interface{}{
			map[string]interface{}{
				"name":                  "allow-from-lb",
				"direction":             "Ingress",
				"protocol":              "TCP",
				"port":                  443,
				"end_port":              0,
				"remote_ip":             "",
				"remote_security_group": fqid,
				"remote_subnet":         "",
			},
		}

		result := expandSecurityGroupRules(config.Client, input)
		if len(result) != 1 {
			t.Fatalf("expected 1 rule, got %d", len(result))
		}

		rule := result[0]
		if rule.Remote.SecurityGroupRef == nil {
			t.Fatal("expected remote security group ref to be set")
		}
		if *rule.Remote.SecurityGroupRef != fqid {
			t.Errorf("expected FQID pass-through %q, got %q", fqid, *rule.Remote.SecurityGroupRef)
		}
	})
}

func TestFlattenSecurityGroupRules(t *testing.T) {
	t.Run("empty rules", func(t *testing.T) {
		result := flattenSecurityGroupRules([]networkingtypes.SecurityGroupSpecRulesItem{})
		if len(result) != 0 {
			t.Errorf("expected 0 rules, got %d", len(result))
		}
	})

	t.Run("single rule", func(t *testing.T) {
		name := "allow-ssh"
		protocol := networkingtypes.SecurityGroupSpecRulesItemProtocol("TCP")
		port := int32(22)

		rules := []networkingtypes.SecurityGroupSpecRulesItem{
			{
				Name:      &name,
				Direction: "Ingress",
				Protocol:  &protocol,
				Port:      &port,
				Remote: struct {
					Address          *networkingtypes.SecurityGroupSpecRulesItemAddress `json:"address,omitempty"`
					SecurityGroupRef *string                                            `json:"securityGroupRef,omitempty"`
					SubnetRef        *string                                            `json:"subnetRef,omitempty"`
				}{
					Address: &networkingtypes.SecurityGroupSpecRulesItemAddress{
						IpAddressOrCIDR: "0.0.0.0/0",
					},
				},
			},
		}

		result := flattenSecurityGroupRules(rules)
		if len(result) != 1 {
			t.Fatalf("expected 1 rule, got %d", len(result))
		}

		r := result[0].(map[string]interface{})
		if r["name"] != "allow-ssh" {
			t.Errorf("expected name %q, got %q", "allow-ssh", r["name"])
		}
		if r["direction"] != "Ingress" {
			t.Errorf("expected direction Ingress, got %q", r["direction"])
		}
		if r["protocol"] != "TCP" {
			t.Errorf("expected protocol TCP, got %q", r["protocol"])
		}
		if r["port"] != 22 {
			t.Errorf("expected port 22, got %v", r["port"])
		}
		if r["end_port"] != 0 {
			t.Errorf("expected end_port 0, got %v", r["end_port"])
		}
		if r["remote_ip"] != "0.0.0.0/0" {
			t.Errorf("expected remote_ip 0.0.0.0/0, got %q", r["remote_ip"])
		}
	})

	t.Run("rule without optional fields", func(t *testing.T) {
		name := "egress-all"

		rules := []networkingtypes.SecurityGroupSpecRulesItem{
			{
				Name:      &name,
				Direction: "Egress",
			},
		}

		result := flattenSecurityGroupRules(rules)
		if len(result) != 1 {
			t.Fatalf("expected 1 rule, got %d", len(result))
		}

		r := result[0].(map[string]interface{})
		if proto, exists := r["protocol"]; !exists || proto != "All" {
			t.Errorf("expected protocol to be 'All' when nil, got %v", r["protocol"])
		}
		if _, exists := r["port"]; exists {
			t.Error("expected port to not be set")
		}
		if _, exists := r["remote_ip"]; exists {
			t.Error("expected remote_ip to not be set")
		}
	})

	t.Run("rule with end_port", func(t *testing.T) {
		name := "port-range"
		protocol := networkingtypes.SecurityGroupSpecRulesItemProtocol("TCP")
		port := int32(8000)
		endPort := int32(9000)

		rules := []networkingtypes.SecurityGroupSpecRulesItem{
			{
				Name:      &name,
				Direction: "Ingress",
				Protocol:  &protocol,
				Port:      &port,
				EndPort:   &endPort,
			},
		}

		result := flattenSecurityGroupRules(rules)
		r := result[0].(map[string]interface{})
		if r["port"] != 8000 {
			t.Errorf("expected port 8000, got %v", r["port"])
		}
		if r["end_port"] != 9000 {
			t.Errorf("expected end_port 9000, got %v", r["end_port"])
		}
	})

	t.Run("rule with security group ref", func(t *testing.T) {
		name := "allow-from-lb"
		protocol := networkingtypes.SecurityGroupSpecRulesItemProtocol("TCP")
		port := int32(443)
		sgRef := "/networking/projects/proj/regions/se-sto/securityGroups/lb-sg"

		rules := []networkingtypes.SecurityGroupSpecRulesItem{
			{
				Name:      &name,
				Direction: "Ingress",
				Protocol:  &protocol,
				Port:      &port,
				Remote: struct {
					Address          *networkingtypes.SecurityGroupSpecRulesItemAddress `json:"address,omitempty"`
					SecurityGroupRef *string                                            `json:"securityGroupRef,omitempty"`
					SubnetRef        *string                                            `json:"subnetRef,omitempty"`
				}{
					SecurityGroupRef: &sgRef,
				},
			},
		}

		result := flattenSecurityGroupRules(rules)
		r := result[0].(map[string]interface{})
		if r["remote_security_group"] != sgRef {
			t.Errorf("expected remote_security_group %q, got %q", sgRef, r["remote_security_group"])
		}
	})

	t.Run("roundtrip expand then flatten", func(t *testing.T) {
		ms := newMockServer()
		defer ms.close()
		setupCatchAll(ms)
		config := newTestProviderConfig(t, ms.server.URL)

		input := []interface{}{
			map[string]interface{}{
				"name":      "ssh",
				"direction": "Ingress",
				"protocol":  "TCP",
				"port":      22,
				"end_port":  0,
				"remote_ip": "10.0.0.0/8",
			},
		}

		expanded := expandSecurityGroupRules(config.Client, input)
		flattened := flattenSecurityGroupRules(expanded)

		if len(flattened) != 1 {
			t.Fatalf("expected 1 rule after roundtrip, got %d", len(flattened))
		}

		r := flattened[0].(map[string]interface{})
		if r["name"] != "ssh" {
			t.Errorf("name mismatch after roundtrip: got %q", r["name"])
		}
		if r["direction"] != "Ingress" {
			t.Errorf("direction mismatch after roundtrip: got %q", r["direction"])
		}
		if r["protocol"] != "TCP" {
			t.Errorf("protocol mismatch after roundtrip: got %q", r["protocol"])
		}
		if r["port"] != 22 {
			t.Errorf("port mismatch after roundtrip: got %v", r["port"])
		}
		if r["remote_ip"] != "10.0.0.0/8" {
			t.Errorf("remote_ip mismatch after roundtrip: got %q", r["remote_ip"])
		}
		if r["end_port"] != 0 {
			t.Errorf("end_port mismatch after roundtrip: got %v", r["end_port"])
		}
	})
}
