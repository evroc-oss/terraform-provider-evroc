// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceFilestore() *schema.Resource {
	return &schema.Resource{
		Description: "Retrieves information about an existing evroc FileStore.",

		ReadContext: dataSourceFilestoreRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the file store to query.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Project this resource belongs to. Defaults to the provider project.",
			},
			"zone": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Availability zone of the file store.",
			},
			"region": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Region where the file store is located.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Current lifecycle status.",
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
				Description: "NFS protocol version.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the file store was created.",
			},
		},
	}
}

func dataSourceFilestoreRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	var diags diag.Diagnostics
	name := d.Get("name").(string)

	fs, err := client.Storage().FileStores().Get(ctx, name)
	if err != nil {
		return diag.Errorf("error getting filestore: %s", err)
	}

	d.SetId(fs.Metadata.Id)
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

	return diags
}
