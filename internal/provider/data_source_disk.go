// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"path"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDisk() *schema.Resource {
	return &schema.Resource{
		Description: "Get information about an existing evroc disk.",

		ReadContext: dataSourceDiskRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the disk to look up.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Project this resource belongs to. Defaults to the provider project.",
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Region where the disk is located.",
			},
			// Computed fields
			"disk_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier (UUID) of the disk.",
			},
			"size": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Size of the disk in GB.",
			},
			"image": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "OS image used for the disk.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the disk was created (RFC3339 format).",
			},
			"fqid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified resource ID (FQID).",
			},
		},
	}
}

func dataSourceDiskRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	var diags diag.Diagnostics

	name := d.Get("name").(string)

	disk, err := client.Compute().Disks().Get(ctx, name)
	if err != nil {
		return diag.Errorf("error reading disk %s: %s", name, err)
	}

	d.SetId(disk.Metadata.Id)
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
	// API returns full path like "/compute/global/diskImages/ubuntu-22.04"
	// Extract just the image name (last part of path)
	if disk.Spec.Source != nil && disk.Spec.Source.DiskImageRef != nil {
		imageName := path.Base(*disk.Spec.Source.DiskImageRef)
		diags = setDiag(d, "image", imageName, diags)
	}

	diags = setDiag(d, "fqid", string(client.Compute().DiskRef(disk.Metadata.Id)), diags)

	return diags
}
