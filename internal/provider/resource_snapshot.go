// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import (
	"context"
	"time"

	"github.com/evroc-oss/evroc-go-sdk/compute"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceSnapshot() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an evroc snapshot resource. Snapshots capture the state of a disk at a point in time and can be used to create new disks.",

		CreateContext: resourceSnapshotCreate,
		ReadContext:   resourceSnapshotRead,
		DeleteContext: resourceSnapshotDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the snapshot. Must be unique within the project.",
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
				ForceNew:    true,
				Computed:    true,
				Description: "Region where the snapshot is created. Defaults to provider region.",
			},
			"disk_ref": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				DiffSuppressFunc: suppressFQIDDiff,
				Description:      "Reference to the disk to snapshot. Accepts FQID or plain name.",
			},
			// Computed fields
			"snapshot_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier (UUID) of the snapshot.",
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
				Description: "Timestamp when the snapshot was created (RFC3339 format).",
			},
			"fqid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified resource ID (FQID). Use this to reference this resource from other resources.",
			},
			"restore_size": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Size of the snapshot in GB.",
			},
		},
	}
}

func resourceSnapshotCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)
	diskRef := d.Get("disk_ref").(string)

	diskFQID := string(client.Compute().DiskRef(diskRef))
	if isFQID(diskRef) {
		diskFQID = diskRef
	}

	snap, err := compute.NewSnapshotBuilder(name).
		WithDiskRef(diskFQID).
		Create(ctx, client.Compute().Snapshots())
	if err != nil {
		return diag.Errorf("error creating snapshot %s: %s", name, err)
	}

	d.SetId(snap.Metadata.Id)
	d.Set("project", resolveProject(d, config))

	timeout := d.Timeout(schema.TimeoutCreate)
	readySnap, err := client.Compute().Snapshots().WaitForReady(ctx, name, timeout)
	if err != nil {
		return diag.Errorf("error waiting for snapshot %s to be ready: %s", name, err)
	}
	d.SetId(readySnap.Metadata.Id)

	return resourceSnapshotRead(ctx, d, meta)
}

func resourceSnapshotRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	snap, err := client.Compute().Snapshots().Get(ctx, d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading snapshot: %s", err)
	}

	diags = setDiag(d, "name", snap.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "region", derefString(snap.Metadata.Region), diags)
	diags = setDiag(d, "snapshot_id", snap.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", snap.Metadata.CreationTimestamp.Format(time.RFC3339), diags)

	if snap.Spec.DiskRef != nil {
		diags = setDiag(d, "disk_ref", *snap.Spec.DiskRef, diags)
	}

	if snap.Status.RestoreSize != nil {
		diags = setDiag(d, "restore_size", int(snap.Status.RestoreSize.Amount), diags)
	}

	diags = setDiag(d, "system_labels", flattenLabels(snap.Metadata.SystemLabels), diags)
	diags = setDiag(d, "fqid", client.Compute().SnapshotRef(snap.Metadata.Id), diags)

	return diags
}

func resourceSnapshotDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	err := client.Compute().Snapshots().Delete(ctx, d.Id())
	if err != nil {
		if !isNotFoundError(err) {
			return diag.Errorf("error deleting snapshot %s: %s", d.Id(), err)
		}
		d.SetId("")
		return nil
	}

	timeout := d.Timeout(schema.TimeoutDelete)
	if err := client.Compute().Snapshots().WaitForDeleted(ctx, d.Id(), timeout); err != nil {
		return diag.Errorf("error waiting for snapshot %s to be deleted: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}
