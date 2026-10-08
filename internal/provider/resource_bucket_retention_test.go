// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBucketRetentionForceNewOnDisable(t *testing.T) {
	res := resourceBucket()

	for _, tc := range []struct {
		from, to string
		forceNew bool
	}{
		{"Disabled", "Versioned", false},
		{"Versioned", "Suspended", false},
		{"Versioned", "Versioned", false},
		{"Locking", "Locking", false},
		{"Versioned", "Disabled", true},
		{"Locking", "Disabled", true},
		{"Locking", "Versioned", true},
	} {
		state := &terraform.InstanceState{
			ID:         "test-bucket",
			Attributes: map[string]string{"name": "test-bucket", "object_retention_mode": tc.from},
		}
		cfg := terraform.NewResourceConfigRaw(map[string]interface{}{"name": "test-bucket", "object_retention_mode": tc.to})
		diff, err := res.Diff(context.Background(), state, cfg, nil)
		if err != nil {
			t.Fatalf("%s -> %s: unexpected error: %v", tc.from, tc.to, err)
		}
		got := diff != nil && diff.RequiresNew()
		if got != tc.forceNew {
			t.Errorf("%s -> %s: RequiresNew = %v, want %v", tc.from, tc.to, got, tc.forceNew)
		}
	}
}
