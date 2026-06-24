// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"
	"net"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// validateZone validates that the zone is one of the valid zones
func validateZone() schema.SchemaValidateDiagFunc {
	return validation.ToDiagFunc(validation.StringInSlice([]string{"a", "b", "c"}, false))
}

// validatePort validates that the port is in the valid range (0-65535)
// Port 0 is allowed and means "all ports"
func validatePort() schema.SchemaValidateDiagFunc {
	return validation.ToDiagFunc(validation.IntBetween(0, 65535))
}

// validateSecurityGroupDirection validates the security group rule direction
func validateSecurityGroupDirection() schema.SchemaValidateDiagFunc {
	return validation.ToDiagFunc(validation.StringInSlice([]string{"Ingress", "Egress"}, false))
}

// validateSecurityGroupProtocol validates the security group rule protocol
func validateSecurityGroupProtocol() schema.SchemaValidateDiagFunc {
	return validation.ToDiagFunc(validation.StringInSlice([]string{"TCP", "UDP", "ICMP", "All"}, false))
}

// validatePlacementStrategy validates the placement group strategy
func validatePlacementStrategy() schema.SchemaValidateDiagFunc {
	return validation.ToDiagFunc(validation.StringInSlice([]string{"spread", "cluster"}, false))
}

// validateObjectRetentionMode validates bucket object retention mode
func validateObjectRetentionMode() schema.SchemaValidateDiagFunc {
	return validation.ToDiagFunc(validation.StringInSlice([]string{"Disabled", "Versioned", "Suspended", "Locking"}, false))
}

// validateObjectLockingMode validates bucket object locking mode
func validateObjectLockingMode() schema.SchemaValidateDiagFunc {
	return validation.ToDiagFunc(validation.StringInSlice([]string{"Soft", "Immutable"}, false))
}

// validateCIDR validates that a string is a valid CIDR notation
func validateCIDR() schema.SchemaValidateDiagFunc {
	return validation.ToDiagFunc(validation.IsCIDR)
}

func validateVPCStackType() schema.SchemaValidateDiagFunc {
	return validation.ToDiagFunc(validation.StringInSlice([]string{"dual-stack", "ipv6-only"}, false))
}

func validateVMStackType() schema.SchemaValidateDiagFunc {
	return validation.ToDiagFunc(validation.StringInSlice([]string{"dual-stack", "ipv6-only", "ipv4-only"}, false))
}

var rfc1918Nets []*net.IPNet

func init() {
	for _, cidr := range []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"} {
		_, n, _ := net.ParseCIDR(cidr) //nolint:errcheck // constant CIDRs
		rfc1918Nets = append(rfc1918Nets, n)
	}
}

func validateVPCCIDR() schema.SchemaValidateDiagFunc {
	return validation.ToDiagFunc(func(i interface{}, k string) (warnings []string, errors []error) {
		v, ok := i.(string)
		if !ok {
			errors = append(errors, fmt.Errorf("expected type of %s to be string", k))
			return warnings, errors
		}

		ip, _, err := net.ParseCIDR(v)
		if err != nil {
			errors = append(errors, fmt.Errorf("expected %s to be a valid CIDR, got: %s", k, v))
			return warnings, errors
		}

		isRFC1918 := false
		for _, n := range rfc1918Nets {
			if n.Contains(ip) {
				isRFC1918 = true
				break
			}
		}
		if !isRFC1918 {
			errors = append(errors, fmt.Errorf("%s must be within RFC 1918 ranges (10.0.0.0/8, 172.16.0.0/12, or 192.168.0.0/16), got: %s", k, v))
		}
		return warnings, errors
	})
}

func validateSubnetCIDR() schema.SchemaValidateDiagFunc {
	return validation.ToDiagFunc(func(i interface{}, k string) (warnings []string, errors []error) {
		v, ok := i.(string)
		if !ok {
			errors = append(errors, fmt.Errorf("expected type of %s to be string", k))
			return warnings, errors
		}

		ip, ipNet, err := net.ParseCIDR(v)
		if err != nil {
			errors = append(errors, fmt.Errorf("%s is not a valid CIDR: %w", k, err))
			return warnings, errors
		}

		if !ip.Equal(ip.Mask(ipNet.Mask)) {
			errors = append(errors, fmt.Errorf("%s must be a canonical CIDR (host bits must be zero, e.g., 10.0.0.0/24 not 10.0.0.1/24)", k))
			return warnings, errors
		}

		ones, _ := ipNet.Mask.Size()
		if ones < 16 || ones > 29 {
			errors = append(errors, fmt.Errorf("%s prefix must be between /16 and /29, got /%d", k, ones))
			return warnings, errors
		}

		return warnings, errors
	})
}

// validatePositiveInt validates that an integer is positive (> 0)
func validatePositiveInt() schema.SchemaValidateDiagFunc {
	return validation.ToDiagFunc(func(i interface{}, k string) (warnings []string, errors []error) {
		v, ok := i.(int)
		if !ok {
			errors = append(errors, fmt.Errorf("expected type of %s to be int", k))
			return warnings, errors
		}

		if v <= 0 {
			errors = append(errors, fmt.Errorf("expected %s to be greater than 0, got %d", k, v))
			return warnings, errors
		}

		return warnings, errors
	})
}
