// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"testing"
)

func TestValidateZone(t *testing.T) {
	validValues := []string{"a", "b", "c"}
	invalidValues := []string{"d", "z", "A", "ab", "", "1"}

	validator := validateZone()

	for _, v := range validValues {
		diags := validator(v, nil)
		if diags.HasError() {
			t.Errorf("expected zone %q to be valid, got error: %s", v, diags[0].Summary)
		}
	}

	for _, v := range invalidValues {
		diags := validator(v, nil)
		if !diags.HasError() {
			t.Errorf("expected zone %q to be invalid, got no error", v)
		}
	}
}

func TestValidatePort(t *testing.T) {
	validPorts := []int{0, 1, 22, 80, 443, 8080, 65535}
	invalidPorts := []int{-1, -100, 65536, 100000}

	validator := validatePort()

	for _, v := range validPorts {
		diags := validator(v, nil)
		if diags.HasError() {
			t.Errorf("expected port %d to be valid, got error: %s", v, diags[0].Summary)
		}
	}

	for _, v := range invalidPorts {
		diags := validator(v, nil)
		if !diags.HasError() {
			t.Errorf("expected port %d to be invalid, got no error", v)
		}
	}
}

func TestValidateSecurityGroupDirection(t *testing.T) {
	validValues := []string{"Ingress", "Egress"}
	invalidValues := []string{"ingress", "egress", "INGRESS", "inbound", "outbound", ""}

	validator := validateSecurityGroupDirection()

	for _, v := range validValues {
		diags := validator(v, nil)
		if diags.HasError() {
			t.Errorf("expected direction %q to be valid, got error: %s", v, diags[0].Summary)
		}
	}

	for _, v := range invalidValues {
		diags := validator(v, nil)
		if !diags.HasError() {
			t.Errorf("expected direction %q to be invalid, got no error", v)
		}
	}
}

func TestValidateSecurityGroupProtocol(t *testing.T) {
	validValues := []string{"TCP", "UDP", "ICMP", "All"}
	invalidValues := []string{"tcp", "udp", "icmp", "all", "HTTP", ""}

	validator := validateSecurityGroupProtocol()

	for _, v := range validValues {
		diags := validator(v, nil)
		if diags.HasError() {
			t.Errorf("expected protocol %q to be valid, got error: %s", v, diags[0].Summary)
		}
	}

	for _, v := range invalidValues {
		diags := validator(v, nil)
		if !diags.HasError() {
			t.Errorf("expected protocol %q to be invalid, got no error", v)
		}
	}
}

func TestValidatePlacementStrategy(t *testing.T) {
	validValues := []string{"spread", "cluster"}
	invalidValues := []string{"Spread", "Cluster", "random", ""}

	validator := validatePlacementStrategy()

	for _, v := range validValues {
		diags := validator(v, nil)
		if diags.HasError() {
			t.Errorf("expected strategy %q to be valid, got error: %s", v, diags[0].Summary)
		}
	}

	for _, v := range invalidValues {
		diags := validator(v, nil)
		if !diags.HasError() {
			t.Errorf("expected strategy %q to be invalid, got no error", v)
		}
	}
}

func TestValidateObjectRetentionMode(t *testing.T) {
	validValues := []string{"Disabled", "Versioned", "Locking"}
	invalidValues := []string{"disabled", "versioned", "locking", "DISABLED", ""}

	validator := validateObjectRetentionMode()

	for _, v := range validValues {
		diags := validator(v, nil)
		if diags.HasError() {
			t.Errorf("expected retention mode %q to be valid, got error: %s", v, diags[0].Summary)
		}
	}

	for _, v := range invalidValues {
		diags := validator(v, nil)
		if !diags.HasError() {
			t.Errorf("expected retention mode %q to be invalid, got no error", v)
		}
	}
}

func TestValidateObjectLockingMode(t *testing.T) {
	validValues := []string{"Soft", "Immutable"}
	invalidValues := []string{"soft", "immutable", "governance", ""}

	validator := validateObjectLockingMode()

	for _, v := range validValues {
		diags := validator(v, nil)
		if diags.HasError() {
			t.Errorf("expected locking mode %q to be valid, got error: %s", v, diags[0].Summary)
		}
	}

	for _, v := range invalidValues {
		diags := validator(v, nil)
		if !diags.HasError() {
			t.Errorf("expected locking mode %q to be invalid, got no error", v)
		}
	}
}

func TestValidateCIDR(t *testing.T) {
	validValues := []string{"10.0.0.0/8", "192.168.1.0/24", "0.0.0.0/0", "172.16.0.0/12"}
	invalidValues := []string{"10.0.0.0", "not-a-cidr", "256.0.0.0/8", ""}

	validator := validateCIDR()

	for _, v := range validValues {
		diags := validator(v, nil)
		if diags.HasError() {
			t.Errorf("expected CIDR %q to be valid, got error: %s", v, diags[0].Summary)
		}
	}

	for _, v := range invalidValues {
		diags := validator(v, nil)
		if !diags.HasError() {
			t.Errorf("expected CIDR %q to be invalid, got no error", v)
		}
	}
}

func TestValidatePositiveInt(t *testing.T) {
	validValues := []int{1, 2, 100, 1000}
	invalidValues := []int{0, -1, -100}

	validator := validatePositiveInt()

	for _, v := range validValues {
		diags := validator(v, nil)
		if diags.HasError() {
			t.Errorf("expected %d to be valid, got error: %s", v, diags[0].Summary)
		}
	}

	for _, v := range invalidValues {
		diags := validator(v, nil)
		if !diags.HasError() {
			t.Errorf("expected %d to be invalid, got no error", v)
		}
	}

	// Test non-int input
	diags := validatePositiveInt()("not-an-int", nil)
	if !diags.HasError() {
		t.Error("expected non-int input to be invalid, got no error")
	}
}
