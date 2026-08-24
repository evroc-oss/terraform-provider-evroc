// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceBucket() *schema.Resource {
	return &schema.Resource{
		Description: "Retrieves information about an existing evroc storage bucket.",

		ReadContext: dataSourceBucketRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the bucket to query.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Project this resource belongs to. Defaults to the provider project.",
			},
			"object_retention_mode": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Object retention mode of the bucket.",
			},
			"object_locking": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Default object locking configuration.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"mode": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Lock mode: Soft or Immutable.",
						},
						"duration_days": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Duration in days for the default lock.",
						},
					},
				},
			},
			"lifecycle_rule": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Lifecycle rules that determine how and when objects or object versions are automatically deleted.",
				Elem:        dataSourceBucketLifecycleRuleSchema(),
			},
			"region": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Region where the bucket is located.",
			},
			"bucket_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier of the bucket.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the bucket was created.",
			},
		},
	}
}

func dataSourceBucketRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	var diags diag.Diagnostics
	name := d.Get("name").(string)

	bucket, err := client.Storage().Buckets().Get(ctx, name)
	if err != nil {
		return diag.Errorf("error getting bucket: %s", err)
	}

	d.SetId(bucket.Metadata.Id)
	diags = setDiag(d, "name", bucket.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "region", derefString(bucket.Metadata.Region), diags)
	diags = setDiag(d, "bucket_id", bucket.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", bucket.Metadata.CreationTimestamp.Format(time.RFC3339), diags)

	if bucket.Spec.ObjectRetentionMode != nil {
		diags = setDiag(d, "object_retention_mode", string(*bucket.Spec.ObjectRetentionMode), diags)
	}

	if bucket.Spec.DefaultObjectLocking != nil {
		locking := make(map[string]interface{})
		locking["mode"] = string(bucket.Spec.DefaultObjectLocking.Mode)
		locking["duration_days"] = int(bucket.Spec.DefaultObjectLocking.DurationDays)
		diags = setDiag(d, "object_locking", []interface{}{locking}, diags)
	}

	diags = setDiag(d, "lifecycle_rule", flattenBucketLifecyclePolicy(bucket.Spec.LifecyclePolicy), diags)

	return diags
}

// dataSourceBucketLifecycleRuleSchema mirrors bucketLifecycleRuleSchema with all
// fields computed, since data source attributes are read-only.
func dataSourceBucketLifecycleRuleSchema() *schema.Resource {
	return computedSchemaFromResource(bucketLifecycleRuleSchema())
}

// computedSchemaFromResource converts a resource schema into a read-only variant
// where every attribute (including nested blocks) is computed.
func computedSchemaFromResource(r *schema.Resource) *schema.Resource {
	out := &schema.Resource{Schema: make(map[string]*schema.Schema, len(r.Schema))}
	for name, s := range r.Schema {
		cp := *s
		cp.Required = false
		cp.Optional = false
		cp.Computed = true
		cp.Default = nil
		cp.ValidateDiagFunc = nil
		cp.MaxItems = 0
		cp.MinItems = 0
		if nested, ok := cp.Elem.(*schema.Resource); ok {
			cp.Elem = computedSchemaFromResource(nested)
		}
		out.Schema[name] = &cp
	}
	return out
}
