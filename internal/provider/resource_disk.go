// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"path"
	"time"

	computetypes "github.com/evroc-oss/evroc-go-sdk/types/compute"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDisk() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an evroc disk resource for persistent block storage.",

		CreateContext: resourceDiskCreate,
		ReadContext:   resourceDiskRead,
		UpdateContext: resourceDiskUpdate,
		DeleteContext: resourceDiskDelete,

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
				Description: "Name of the disk. Must be unique within the project.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Project this resource belongs to. Defaults to the provider project.",
			},
			"size": {
				Type:             schema.TypeInt,
				Optional:         true,
				Computed:         true,
				ForceNew:         true,
				ValidateDiagFunc: validatePositiveInt(),
				Description:      "Size of the disk in GB (changes force recreation).",
			},
			"image": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"snapshot"},
				Description:   "OS image for the disk (e.g., ubuntu-24.04, rocky-9-6). Mutually exclusive with snapshot.",
			},
			"snapshot": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				ConflictsWith:    []string{"image"},
				DiffSuppressFunc: suppressFQIDDiff,
				Description:      "Snapshot to create the disk from. Accepts FQID or plain name. Mutually exclusive with image.",
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Computed:    true,
				Description: "Region where the disk is created. Defaults to provider region.",
			},
			"zone": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validateZone(),
				Description:      "Zone (e.g., a, b, c).",
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
			"disk_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier (UUID) of the disk.",
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
				Description: "Timestamp when the disk was created (RFC3339 format).",
			},
			"fqid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified resource ID (FQID). Use this to reference this resource from other resources.",
			},
		},
	}
}

func resourceDiskCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)
	size := d.Get("size").(int)
	image := d.Get("image").(string)
	zone := d.Get("zone").(string)

	var snapshot string
	if s, ok := d.GetOk("snapshot"); ok {
		snapshot = s.(string)
		if isFQID(snapshot) {
			// pass through
		} else {
			snapshot = string(client.Compute().SnapshotRef(snapshot))
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

	req := BuildDiskCreateRequest(name, size, image, snapshot, zone, userLabels)

	disk, err := client.Compute().Disks().Create(ctx, req)
	if err != nil {
		return diag.Errorf("error creating disk %s: %s", name, err)
	}

	d.SetId(disk.Metadata.Id)
	d.Set("project", resolveProject(d, config))

	// Wait for disk to be ready and capture the ready resource
	timeout := d.Timeout(schema.TimeoutCreate)
	readyDisk, err := client.Compute().Disks().WaitForReady(ctx, name, timeout)
	if err != nil {
		return diag.Errorf("error waiting for disk %s to be ready: %s", name, err)
	}

	// Use the ready resource's Ref() for consistent FQID-based identity
	d.SetId(readyDisk.Metadata.Id)

	return resourceDiskRead(ctx, d, meta)
}

func resourceDiskRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	disk, err := client.Compute().Disks().Get(ctx, d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading disk: %s", err)
	}

	diags = setDiag(d, "name", disk.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "region", derefString(disk.Metadata.Region), diags)
	diags = setDiag(d, "disk_id", disk.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", disk.Metadata.CreationTimestamp.Format(time.RFC3339), diags)

	// Set disk size
	if disk.Spec.DiskSize != nil {
		diags = setDiag(d, "size", int(disk.Spec.DiskSize.Amount), diags)
	}

	// Set image if present (DiskImageRef is a string reference path)
	// API returns full path like "/compute/global/diskImages/evroc/ubuntu-minimal.24-04.1"
	// Extract just the image name (last part of path)
	if disk.Spec.Source != nil {
		if disk.Spec.Source.DiskImageRef != nil {
			imageName := path.Base(*disk.Spec.Source.DiskImageRef)
			diags = setDiag(d, "image", imageName, diags)
		}
		if disk.Spec.Source.SnapshotRef != nil {
			diags = setDiag(d, "snapshot", *disk.Spec.Source.SnapshotRef, diags)
		}
	}

	// Set zone if present
	if disk.Spec.Placement.Zone != nil {
		diags = setDiag(d, "zone", *disk.Spec.Placement.Zone, diags)
	}

	if disk.Metadata.UserLabels != nil && len(*disk.Metadata.UserLabels) > 0 {
		diags = setDiag(d, "user_labels", flattenLabels(disk.Metadata.UserLabels), diags)
	}

	diags = setDiag(d, "system_labels", flattenLabels(disk.Metadata.SystemLabels), diags)
	diags = setDiag(d, "fqid", string(client.Compute().DiskRef(disk.Metadata.Id)), diags)

	return diags
}

func resourceDiskUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	if d.HasChange("user_labels") {
		disk, err := client.Compute().Disks().Get(ctx, d.Id())
		if err != nil {
			return diag.Errorf("error reading disk %s: %s", d.Id(), err)
		}

		if labels, ok := d.GetOk("user_labels"); ok {
			userLabels := make(computetypes.UserLabels)
			for k, v := range labels.(map[string]interface{}) {
				userLabels[k] = v.(string)
			}
			disk.Metadata.UserLabels = &userLabels
		} else {
			disk.Metadata.UserLabels = nil
		}

		_, err = client.Compute().Disks().Patch(ctx, d.Id(), disk)
		if err != nil {
			return diag.Errorf("error updating disk %s: %s", d.Id(), err)
		}
	}

	return resourceDiskRead(ctx, d, meta)
}

func resourceDiskDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	err := client.Compute().Disks().Delete(ctx, d.Id())
	if err != nil {
		if !isNotFoundError(err) {
			return diag.Errorf("error deleting disk %s: %s", d.Id(), err)
		}
		d.SetId("")
		return nil
	}

	timeout := d.Timeout(schema.TimeoutDelete)
	if err := client.Compute().Disks().WaitForDeleted(ctx, d.Id(), timeout); err != nil {
		return diag.Errorf("error waiting for disk %s to be deleted: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}
