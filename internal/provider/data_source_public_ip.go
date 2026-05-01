// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourcePublicIP() *schema.Resource {
	return &schema.Resource{
		Description: "Get information about an existing evroc public IP.",

		ReadContext: dataSourcePublicIPRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the public IP to look up.",
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
				Description: "Region where the public IP is allocated.",
			},
			// Computed fields
			"ip_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier (UUID) of the public IP.",
			},
			"ip_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The allocated IPv4 address.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the public IP was created (RFC3339 format).",
			},
			"fqid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified resource ID (FQID).",
			},
		},
	}
}

func dataSourcePublicIPRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	var diags diag.Diagnostics

	name := d.Get("name").(string)

	publicIP, err := client.Networking().PublicIPs().Get(ctx, name)
	if err != nil {
		return diag.Errorf("error reading public IP %s: %s", name, err)
	}

	d.SetId(publicIP.Metadata.Id)
	diags = setDiag(d, "name", publicIP.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "region", derefString(publicIP.Metadata.Region), diags)
	diags = setDiag(d, "ip_id", publicIP.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", publicIP.Metadata.CreationTimestamp.Format(time.RFC3339), diags)

	// Set IP address if allocated
	if publicIP.Status.PublicIPv4Address != nil {
		diags = setDiag(d, "ip_address", *publicIP.Status.PublicIPv4Address, diags)
	}

	diags = setDiag(d, "fqid", string(client.Networking().PublicIPRef(publicIP.Metadata.Id)), diags)

	return diags
}
