// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourcePlacementGroup() *schema.Resource {
	return &schema.Resource{
		Description: "Retrieves information about an existing evroc placement group.",

		ReadContext: dataSourcePlacementGroupRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the placement group to query.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Project this resource belongs to. Defaults to the provider project.",
			},
			"strategy": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Placement strategy of the placement group.",
			},
			"zone": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Zone of the placement group.",
			},
			"region": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Region where the placement group is located.",
			},
			"pg_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier of the placement group.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the placement group was created.",
			},
			"fqid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified resource ID (FQID).",
			},
		},
	}
}

func dataSourcePlacementGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	var diags diag.Diagnostics
	name := d.Get("name").(string)

	pg, err := client.Compute().PlacementGroups().Get(ctx, name)
	if err != nil {
		return diag.Errorf("error getting placement group: %s", err)
	}

	d.SetId(pg.Metadata.Id)
	diags = setDiag(d, "name", pg.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "region", derefString(pg.Metadata.Region), diags)
	diags = setDiag(d, "pg_id", pg.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", pg.Metadata.CreationTimestamp.Format(time.RFC3339), diags)

	diags = setDiag(d, "strategy", string(pg.Spec.Strategy.Type), diags)

	if pg.Spec.Placement.Zone != nil {
		diags = setDiag(d, "zone", *pg.Spec.Placement.Zone, diags)
	}

	diags = setDiag(d, "fqid", string(client.Compute().PlacementGroupRef(pg.Metadata.Id)), diags)

	return diags
}
