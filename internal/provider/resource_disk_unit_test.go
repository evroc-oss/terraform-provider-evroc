// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestDiskSizeForceNewOnShrink(t *testing.T) {
	res := resourceDisk()
	state := &terraform.InstanceState{
		ID:         "test-disk",
		Attributes: map[string]string{"name": "test-disk", "zone": "a", "size": "100"},
	}

	for _, tc := range []struct {
		size     string
		forceNew bool
	}{
		{"120", false},
		{"100", false},
		{"80", true},
	} {
		cfg := terraform.NewResourceConfigRaw(map[string]interface{}{"name": "test-disk", "zone": "a", "size": tc.size})
		diff, err := res.Diff(context.Background(), state, cfg, nil)
		if err != nil {
			t.Fatalf("size %s: unexpected error: %v", tc.size, err)
		}
		got := diff != nil && diff.RequiresNew()
		if got != tc.forceNew {
			t.Errorf("size 100 -> %s: RequiresNew = %v, want %v", tc.size, got, tc.forceNew)
		}
	}
}
