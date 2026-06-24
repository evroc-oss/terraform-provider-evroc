// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceSnapshot() *schema.Resource {
	return &schema.Resource{
		Description: "Get information about an existing evroc snapshot.",

		ReadContext: dataSourceSnapshotRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the snapshot to look up.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Project this resource belongs to. Defaults to the provider project.",
			},
			// Computed fields
			"snapshot_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier (UUID) of the snapshot.",
			},
			"region": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Region where the snapshot is located.",
			},
			"disk_ref": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Reference to the source disk.",
			},
			"restore_size": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Size of the snapshot in GB.",
			},
			"system_labels": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "System-managed labels (read-only).",
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
				Description: "Fully qualified resource ID (FQID).",
			},
		},
	}
}

func dataSourceSnapshotRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	var diags diag.Diagnostics

	name := d.Get("name").(string)

	snap, err := client.Compute().Snapshots().Get(ctx, name)
	if err != nil {
		return diag.Errorf("error reading snapshot %s: %s", name, err)
	}

	d.SetId(snap.Metadata.Id)
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
