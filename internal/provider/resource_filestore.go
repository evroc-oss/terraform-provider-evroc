// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"time"

	"github.com/evroc-oss/evroc-go-sdk/storage"
	storagetypes "github.com/evroc-oss/evroc-go-sdk/types/storage"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceFilestore() *schema.Resource {
	return &schema.Resource{
		Description: "Manages an evroc FileStore — a managed file system for shared storage.",

		CreateContext: resourceFilestoreCreate,
		ReadContext:   resourceFilestoreRead,
		UpdateContext: resourceFilestoreUpdate,
		DeleteContext: resourceFilestoreDelete,

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
				Description: "Name of the file store.",
			},
			"zone": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Availability zone for the file store.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Project this resource belongs to. Defaults to the provider project.",
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Region where the file store will be created.",
			},
			"user_labels": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "User-defined labels (key/value pairs) for organizing and selecting resources.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			// Computed fields
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Current lifecycle status (Pending, Reconciling, Provisioning, Available, Released, Failed).",
			},
			"nfs_endpoint": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "NFS server IP or hostname for mounting.",
			},
			"nfs_export_path": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "NFS export path on the server.",
			},
			"nfs_version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "NFS protocol version (e.g. V4.1).",
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
				Description: "Timestamp when the file store was created.",
			},
		},
	}
}

func resourceFilestoreCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)
	zone := d.Get("zone").(string)

	builder := storage.NewFileStoreBuilder(name, zone)

	if labels, ok := d.GetOk("user_labels"); ok {
		userLabels := make(map[string]string)
		for k, v := range labels.(map[string]interface{}) {
			userLabels[k] = v.(string)
		}
		builder.WithLabels(userLabels)
	}

	fs, err := client.Storage().FileStores().Create(ctx, builder.Build())
	if err != nil {
		return diag.Errorf("error creating filestore: %s", err)
	}

	d.SetId(fs.Metadata.Id)
	d.Set("project", resolveProject(d, config))

	// Wait for filestore to become available
	timeout := d.Timeout(schema.TimeoutCreate)
	readyFS, err := client.Storage().FileStores().WaitForAvailable(ctx, name, timeout)
	if err != nil {
		return diag.Errorf("error waiting for filestore %s to be available: %s", name, err)
	}

	d.SetId(readyFS.Metadata.Id)

	return resourceFilestoreRead(ctx, d, meta)
}

func resourceFilestoreRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	fs, err := client.Storage().FileStores().Get(ctx, d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading filestore: %s", err)
	}

	diags = setDiag(d, "name", fs.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "region", derefString(fs.Metadata.Region), diags)
	diags = setDiag(d, "zone", fs.Spec.Placement.Zone, diags)
	diags = setDiag(d, "created_at", fs.Metadata.CreationTimestamp.Format(time.RFC3339), diags)

	if fs.Status.Status != nil {
		diags = setDiag(d, "status", string(*fs.Status.Status), diags)
	}

	if fs.Status.Nfs != nil {
		diags = setDiag(d, "nfs_endpoint", fs.Status.Nfs.Endpoint, diags)
		diags = setDiag(d, "nfs_export_path", fs.Status.Nfs.ExportPath, diags)
		diags = setDiag(d, "nfs_version", string(fs.Status.Nfs.Version), diags)
	}

	if fs.Metadata.UserLabels != nil && len(*fs.Metadata.UserLabels) > 0 {
		diags = setDiag(d, "user_labels", flattenLabels(fs.Metadata.UserLabels), diags)
	}

	diags = setDiag(d, "system_labels", flattenLabels(fs.Metadata.SystemLabels), diags)

	return diags
}

func resourceFilestoreUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	if d.HasChange("user_labels") {
		fs, err := client.Storage().FileStores().Get(ctx, d.Id())
		if err != nil {
			return diag.Errorf("error getting filestore for update: %s", err)
		}

		if labels, ok := d.GetOk("user_labels"); ok {
			userLabels := make(storagetypes.UserLabels)
			for k, v := range labels.(map[string]interface{}) {
				userLabels[k] = v.(string)
			}
			fs.Metadata.UserLabels = &userLabels
		} else {
			fs.Metadata.UserLabels = nil
		}

		_, err = client.Storage().FileStores().Patch(ctx, d.Id(), fs)
		if err != nil {
			return diag.Errorf("error updating filestore: %s", err)
		}
	}

	return resourceFilestoreRead(ctx, d, meta)
}

func resourceFilestoreDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	err := client.Storage().FileStores().Delete(ctx, d.Id())
	if err != nil {
		if !isNotFoundError(err) {
			return diag.Errorf("error deleting filestore: %s", err)
		}
		d.SetId("")
		return nil
	}

	timeout := d.Timeout(schema.TimeoutDelete)
	if err := client.Storage().FileStores().WaitForDeleted(ctx, d.Id(), timeout); err != nil {
		return diag.Errorf("error waiting for filestore %s to be deleted: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}
