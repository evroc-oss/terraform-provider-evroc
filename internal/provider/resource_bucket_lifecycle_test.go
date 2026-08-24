// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"reflect"
	"testing"
)

func TestExpandBucketLifecyclePolicy(t *testing.T) {
	rules := []interface{}{
		map[string]interface{}{
			"id":       "expire-logs",
			"disabled": false,
			"expire_current_version": []interface{}{
				map[string]interface{}{
					"days":                             30,
					"date":                             "",
					"expire_orphaned_deletion_markers": true,
				},
			},
			"expire_non_current_version": []interface{}{
				map[string]interface{}{
					"days":             7,
					"max_num_versions": 3,
				},
			},
			"abort_incomplete_multipart": []interface{}{
				map[string]interface{}{
					"days": 5,
				},
			},
			"filter": []interface{}{
				map[string]interface{}{
					"prefix":            "logs/",
					"size_greater_than": 1024,
					"size_less_than":    0,
					"tag": []interface{}{
						map[string]interface{}{"key": "env", "value": "dev"},
					},
				},
			},
		},
	}

	policy, err := expandBucketLifecyclePolicy(rules)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if policy == nil || len(policy.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %+v", policy)
	}

	rule := policy.Rules[0]
	if rule.Id != "expire-logs" {
		t.Errorf("expected id %q, got %q", "expire-logs", rule.Id)
	}
	if rule.Disabled != nil {
		t.Errorf("expected Disabled to be omitted when false, got %v", *rule.Disabled)
	}
	if rule.ExpireCurrentVersion == nil || rule.ExpireCurrentVersion.Days == nil || *rule.ExpireCurrentVersion.Days != 30 {
		t.Errorf("expected ExpireCurrentVersion.Days 30, got %+v", rule.ExpireCurrentVersion)
	}
	if rule.ExpireCurrentVersion.Date != nil {
		t.Errorf("expected ExpireCurrentVersion.Date to be nil, got %v", rule.ExpireCurrentVersion.Date)
	}
	if rule.ExpireCurrentVersion.ExpireOrphanedDeletionMarkers == nil || !*rule.ExpireCurrentVersion.ExpireOrphanedDeletionMarkers {
		t.Errorf("expected ExpireOrphanedDeletionMarkers true, got %+v", rule.ExpireCurrentVersion.ExpireOrphanedDeletionMarkers)
	}
	if rule.ExpireNonCurrentVersion == nil || *rule.ExpireNonCurrentVersion.Days != 7 || *rule.ExpireNonCurrentVersion.MaxNumVersions != 3 {
		t.Errorf("unexpected ExpireNonCurrentVersion: %+v", rule.ExpireNonCurrentVersion)
	}
	if rule.AbortIncompleteMultipart == nil || rule.AbortIncompleteMultipart.Days != 5 {
		t.Errorf("unexpected AbortIncompleteMultipart: %+v", rule.AbortIncompleteMultipart)
	}
	if rule.Filter == nil || rule.Filter.Prefix == nil || *rule.Filter.Prefix != "logs/" {
		t.Fatalf("unexpected Filter: %+v", rule.Filter)
	}
	if rule.Filter.SizeGreaterThan == nil || *rule.Filter.SizeGreaterThan != 1024 {
		t.Errorf("expected SizeGreaterThan 1024, got %+v", rule.Filter.SizeGreaterThan)
	}
	if rule.Filter.SizeLessThan != nil {
		t.Errorf("expected SizeLessThan to be omitted when 0, got %v", *rule.Filter.SizeLessThan)
	}
	if rule.Filter.Tag == nil || len(*rule.Filter.Tag) != 1 || (*rule.Filter.Tag)[0].Key != "env" || (*rule.Filter.Tag)[0].Value != "dev" {
		t.Errorf("unexpected Filter.Tag: %+v", rule.Filter.Tag)
	}
}

func TestExpandBucketLifecyclePolicyEmpty(t *testing.T) {
	policy, err := expandBucketLifecyclePolicy(nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if policy != nil {
		t.Errorf("expected nil policy for empty rules, got %+v", policy)
	}
}

func TestExpandBucketLifecyclePolicyInvalidDate(t *testing.T) {
	rules := []interface{}{
		map[string]interface{}{
			"id": "bad-date",
			"expire_current_version": []interface{}{
				map[string]interface{}{
					"days":                             0,
					"date":                             "not-a-date",
					"expire_orphaned_deletion_markers": false,
				},
			},
		},
	}

	if _, err := expandBucketLifecyclePolicy(rules); err == nil {
		t.Fatal("expected error for invalid date, got nil")
	}
}

func TestFlattenBucketLifecyclePolicyRoundtrip(t *testing.T) {
	rules := []interface{}{
		map[string]interface{}{
			"id":       "roundtrip",
			"disabled": false,
			"expire_current_version": []interface{}{
				map[string]interface{}{
					"days":                             90,
					"date":                             "2027-01-01T00:00:00Z",
					"expire_orphaned_deletion_markers": true,
				},
			},
			"expire_non_current_version": []interface{}{
				map[string]interface{}{
					"days":             14,
					"max_num_versions": 5,
				},
			},
			"abort_incomplete_multipart": []interface{}{
				map[string]interface{}{
					"days": 3,
				},
			},
			"filter": []interface{}{
				map[string]interface{}{
					"prefix":            "tmp/",
					"size_greater_than": 100,
					"size_less_than":    1000,
					"tag": []interface{}{
						map[string]interface{}{"key": "team", "value": "storage"},
					},
				},
			},
		},
	}

	policy, err := expandBucketLifecyclePolicy(rules)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	flattened := flattenBucketLifecyclePolicy(policy)
	if !reflect.DeepEqual(rules, flattened) {
		t.Errorf("roundtrip mismatch:\nexpected: %#v\ngot:      %#v", rules, flattened)
	}
}

func TestFlattenBucketLifecyclePolicyNil(t *testing.T) {
	if got := flattenBucketLifecyclePolicy(nil); got != nil {
		t.Errorf("expected nil for nil policy, got %#v", got)
	}
}
