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
							Description: "Lock mode: GOVERNANCE or COMPLIANCE.",
						},
						"duration_days": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Duration in days for the default lock.",
						},
					},
				},
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

	return diags
}
