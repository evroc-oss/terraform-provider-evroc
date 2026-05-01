// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceHotswapDiskAttachment() *schema.Resource {
	return &schema.Resource{
		Description: "Retrieves information about an existing evroc disk attachment.",

		ReadContext: dataSourceHotswapDiskAttachmentRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the disk attachment to query.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Project this resource belongs to. Defaults to the provider project.",
			},
			"virtual_machine": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the virtual machine the disk is attached to.",
			},
			"disk": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the attached disk.",
			},
			"region": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Region where the disk attachment is located.",
			},
			"attachment_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier of the disk attachment.",
			},
			"serial": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Serial identifier of the attached disk.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the disk attachment was created.",
			},
		},
	}
}

func dataSourceHotswapDiskAttachmentRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	var diags diag.Diagnostics
	name := d.Get("name").(string)

	attachment, err := client.Compute().HotswapDiskAttachments().Get(ctx, name)
	if err != nil {
		return diag.Errorf("error getting disk attachment: %s", err)
	}

	d.SetId(attachment.Metadata.Id)
	diags = setDiag(d, "name", attachment.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "region", derefString(attachment.Metadata.Region), diags)
	diags = setDiag(d, "attachment_id", attachment.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", attachment.Metadata.CreationTimestamp.Format(time.RFC3339), diags)

	// Store full FQID references for VM and disk
	diags = setDiag(d, "virtual_machine", attachment.Spec.VirtualMachineRef, diags)
	diags = setDiag(d, "disk", attachment.Spec.DiskRef, diags)

	if attachment.Status.Serial != nil {
		diags = setDiag(d, "serial", *attachment.Status.Serial, diags)
	}

	return diags
}
