// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"path"
	"time"

	storagetypes "github.com/evroc-oss/evroc-go-sdk/types/storage"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceBucketServiceAccount() *schema.Resource {
	return &schema.Resource{
		Description: "Manages an evroc bucket service account for S3-compatible access credentials.",

		CreateContext: resourceBucketServiceAccountCreate,
		ReadContext:   resourceBucketServiceAccountRead,
		UpdateContext: resourceBucketServiceAccountUpdate,
		DeleteContext: resourceBucketServiceAccountDelete,

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
				Description: "Name of the service account.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Project this resource belongs to. Defaults to the provider project.",
			},
			"buckets": {
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    1,
				Description: "List of bucket names this service account can access.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Region where the service account will be created.",
			},
			"user_labels": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "User-defined labels (key/value pairs) for organizing and selecting resources.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"service_account_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier of the service account.",
			},
			"system_labels": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "System-managed labels automatically set by evroc (read-only).",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"credentials_secret": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the Kubernetes secret containing S3 credentials.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the service account was created.",
			},
		},
	}
}

func resourceBucketServiceAccountCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)
	bucketsRaw := d.Get("buckets").([]interface{})

	buckets := make([]string, len(bucketsRaw))
	for i, b := range bucketsRaw {
		buckets[i] = b.(string)
	}

	// Get user labels
	var userLabels map[string]string
	if labels, ok := d.GetOk("user_labels"); ok {
		userLabels = make(map[string]string)
		for k, v := range labels.(map[string]interface{}) {
			userLabels[k] = v.(string)
		}
	}

	req := BuildBucketServiceAccountCreateRequest(name, buckets, userLabels)

	sa, err := client.Storage().BucketServiceAccounts().Create(ctx, req)
	if err != nil {
		return diag.Errorf("error creating bucket service account: %s", err)
	}

	d.SetId(sa.Metadata.Id)
	d.Set("project", resolveProject(d, config)) //nolint:errcheck // project always valid here

	// Wait for bucket service account to be ready and capture the ready resource
	timeout := d.Timeout(schema.TimeoutCreate)
	readySA, err := client.Storage().BucketServiceAccounts().WaitForReady(ctx, name, timeout)
	if err != nil {
		return diag.Errorf("error waiting for bucket service account %s to be ready: %s", name, err)
	}

	// Use the ready resource's identity
	d.SetId(readySA.Metadata.Id)

	return resourceBucketServiceAccountRead(ctx, d, meta)
}

func resourceBucketServiceAccountRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	sa, err := client.Storage().BucketServiceAccounts().Get(ctx, d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading bucket service account: %s", err)
	}

	diags = setDiag(d, "name", sa.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "region", derefString(sa.Metadata.Region), diags)
	diags = setDiag(d, "service_account_id", sa.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", sa.Metadata.CreationTimestamp.Format(time.RFC3339), diags)

	if sa.Spec.Buckets != nil {
		// Normalize paths - API returns full paths like "/storage/projects/.../buckets/name", we want just "name"
		buckets := make([]string, 0, len(*sa.Spec.Buckets))
		for _, bucket := range *sa.Spec.Buckets {
			buckets = append(buckets, path.Base(bucket))
		}
		diags = setDiag(d, "buckets", buckets, diags)
	}

	if sa.Status.S3CredentialsSecretName != nil {
		diags = setDiag(d, "credentials_secret", *sa.Status.S3CredentialsSecretName, diags)
	}

	if sa.Metadata.UserLabels != nil && len(*sa.Metadata.UserLabels) > 0 {
		diags = setDiag(d, "user_labels", flattenLabels(sa.Metadata.UserLabels), diags)
	}

	diags = setDiag(d, "system_labels", flattenLabels(sa.Metadata.SystemLabels), diags)

	return diags
}

func resourceBucketServiceAccountUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	if d.HasChanges("buckets", "user_labels") {
		// Get the current service account
		sa, err := client.Storage().BucketServiceAccounts().Get(ctx, d.Id())
		if err != nil {
			return diag.Errorf("error getting bucket service account for update: %s", err)
		}

		// Update the buckets
		if d.HasChange("buckets") {
			bucketsRaw := d.Get("buckets").([]interface{})
			buckets := make([]string, len(bucketsRaw))
			for i, b := range bucketsRaw {
				buckets[i] = b.(string)
			}
			sa.Spec.Buckets = &buckets
		}

		// Update user labels
		if d.HasChange("user_labels") {
			if labels, ok := d.GetOk("user_labels"); ok {
				userLabels := make(storagetypes.UserLabels)
				for k, v := range labels.(map[string]interface{}) {
					userLabels[k] = v.(string)
				}
				sa.Metadata.UserLabels = &userLabels
			} else {
				sa.Metadata.UserLabels = nil
			}
		}

		_, err = client.Storage().BucketServiceAccounts().Patch(ctx, d.Id(), sa)
		if err != nil {
			return diag.Errorf("error updating bucket service account: %s", err)
		}
	}

	return resourceBucketServiceAccountRead(ctx, d, meta)
}

func resourceBucketServiceAccountDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	err := client.Storage().BucketServiceAccounts().Delete(ctx, d.Id())
	if err != nil {
		if !isNotFoundError(err) {
			return diag.Errorf("error deleting bucket service account: %s", err)
		}
		d.SetId("")
		return nil
	}

	timeout := d.Timeout(schema.TimeoutDelete)
	if err := client.Storage().BucketServiceAccounts().WaitForDeleted(ctx, d.Id(), timeout); err != nil {
		return diag.Errorf("error waiting for bucket service account %s to be deleted: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}
