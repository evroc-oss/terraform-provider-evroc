// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"

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
