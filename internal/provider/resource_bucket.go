// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"time"

	storagetypes "github.com/evroc-oss/evroc-go-sdk/types/storage"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceBucket() *schema.Resource {
	return &schema.Resource{
		Description: "Manages an evroc S3-compatible storage bucket with object retention and locking support.",

		CreateContext: resourceBucketCreate,
		ReadContext:   resourceBucketRead,
		UpdateContext: resourceBucketUpdate,
		DeleteContext: resourceBucketDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the storage bucket.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Project this resource belongs to. Defaults to the provider project.",
			},
			"object_retention_mode": {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          "Disabled",
				ValidateDiagFunc: validateObjectRetentionMode(),
				Description:      "Object retention mode: Disabled, Versioned, or Locking.",
			},
			"object_locking": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Default object locking configuration.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"mode": {
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: validateObjectLockingMode(),
							Description:      "Lock mode: Soft or Immutable.",
						},
						"duration_days": {
							Type:             schema.TypeInt,
							Required:         true,
							ValidateDiagFunc: validatePositiveInt(),
							Description:      "Duration in days for the default lock.",
						},
					},
				},
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Region where the bucket will be created.",
			},
			"user_labels": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "User-defined labels (key/value pairs) for organizing and selecting resources.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"bucket_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier of the bucket.",
			},
			"system_labels": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "System-managed labels automatically set by evroc (read-only).",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the bucket was created.",
			},
		},
	}
}

func resourceBucketCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)
	retentionMode := d.Get("object_retention_mode").(string)

	var lockingMode string
	var lockingDuration int32

	if v, ok := d.GetOk("object_locking"); ok {
		locking := v.([]interface{})
		if len(locking) > 0 {
			lockingConfig := locking[0].(map[string]interface{})
			lockingMode = lockingConfig["mode"].(string)
			lockingDuration = int32(lockingConfig["duration_days"].(int))
		}
	}

	// Get user labels
	var userLabels map[string]string
	if labels, ok := d.GetOk("user_labels"); ok {
		userLabels = make(map[string]string)
		for k, v := range labels.(map[string]interface{}) {
			userLabels[k] = v.(string)
		}
	}

	req := BuildBucketCreateRequest(name, retentionMode, lockingMode, lockingDuration, userLabels)

	bucket, err := client.Storage().Buckets().Create(ctx, req)
	if err != nil {
		return diag.Errorf("error creating bucket: %s", err)
	}

	d.SetId(bucket.Metadata.Id)
	d.Set("project", resolveProject(d, config))

	// Wait for bucket to be ready and capture the ready resource
	timeout := d.Timeout(schema.TimeoutCreate)
	readyBucket, err := client.Storage().Buckets().WaitForReady(ctx, name, timeout)
	if err != nil {
		return diag.Errorf("error waiting for bucket %s to be ready: %s", name, err)
	}

	// Use the ready resource's identity
	d.SetId(readyBucket.Metadata.Id)

	return resourceBucketRead(ctx, d, meta)
}

func resourceBucketRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	bucket, err := client.Storage().Buckets().Get(ctx, d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading bucket: %s", err)
	}

	diags = setDiag(d, "name", bucket.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "region", derefString(bucket.Metadata.Region), diags)
	diags = setDiag(d, "bucket_id", bucket.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", bucket.Metadata.CreationTimestamp.Format(time.RFC3339), diags)

	// Always set object_retention_mode (default to "Disabled" if nil)
	retentionMode := "Disabled"
	if bucket.Spec.ObjectRetentionMode != nil {
		retentionMode = string(*bucket.Spec.ObjectRetentionMode)
	}
	diags = setDiag(d, "object_retention_mode", retentionMode, diags)

	if bucket.Spec.DefaultObjectLocking != nil {
		locking := make(map[string]interface{})
		locking["mode"] = string(bucket.Spec.DefaultObjectLocking.Mode)
		locking["duration_days"] = int(bucket.Spec.DefaultObjectLocking.DurationDays)
		diags = setDiag(d, "object_locking", []interface{}{locking}, diags)
	}

	if bucket.Metadata.UserLabels != nil && len(*bucket.Metadata.UserLabels) > 0 {
		diags = setDiag(d, "user_labels", flattenLabels(bucket.Metadata.UserLabels), diags)
	}

	diags = setDiag(d, "system_labels", flattenLabels(bucket.Metadata.SystemLabels), diags)

	return diags
}

func resourceBucketUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	if d.HasChanges("object_retention_mode", "object_locking", "user_labels") {
		// Get the current bucket
		bucket, err := client.Storage().Buckets().Get(ctx, d.Id())
		if err != nil {
			return diag.Errorf("error getting bucket for update: %s", err)
		}

		// Update the fields
		retentionMode := d.Get("object_retention_mode").(string)
		if retentionMode != "" {
			mode := storagetypes.BucketSpecObjectRetentionMode(retentionMode)
			bucket.Spec.ObjectRetentionMode = &mode
		}

		if d.HasChange("object_locking") {
			if v, ok := d.GetOk("object_locking"); ok {
				locking := v.([]interface{})
				if len(locking) > 0 {
					lockingConfig := locking[0].(map[string]interface{})
					lockingMode := storagetypes.BucketSpecDefaultObjectLockingMode(lockingConfig["mode"].(string))
					lockingDuration := int32(lockingConfig["duration_days"].(int))
					bucket.Spec.DefaultObjectLocking = &storagetypes.BucketSpecDefaultObjectLocking{
						Mode:         lockingMode,
						DurationDays: lockingDuration,
					}
				}
			} else {
				bucket.Spec.DefaultObjectLocking = nil
			}
		}

		// Update user labels
		if d.HasChange("user_labels") {
			if labels, ok := d.GetOk("user_labels"); ok {
				userLabels := make(storagetypes.UserLabels)
				for k, v := range labels.(map[string]interface{}) {
					userLabels[k] = v.(string)
				}
				bucket.Metadata.UserLabels = &userLabels
			} else {
				bucket.Metadata.UserLabels = nil
			}
		}

		_, err = client.Storage().Buckets().Patch(ctx, d.Id(), bucket)
		if err != nil {
			return diag.Errorf("error updating bucket: %s", err)
		}
	}

	return resourceBucketRead(ctx, d, meta)
}

func resourceBucketDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	err := client.Storage().Buckets().Delete(ctx, d.Id())
	if err != nil {
		if !isNotFoundError(err) {
			return diag.Errorf("error deleting bucket: %s", err)
		}
		d.SetId("")
		return nil
	}

	timeout := d.Timeout(schema.TimeoutDelete)
	if err := client.Storage().Buckets().WaitForDeleted(ctx, d.Id(), timeout); err != nil {
		return diag.Errorf("error waiting for bucket %s to be deleted: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}
