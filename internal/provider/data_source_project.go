// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceProject() *schema.Resource {
	return &schema.Resource{
		Description: "Retrieves information about an existing evroc project.",

		ReadContext: dataSourceProjectRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the project to look up.",
			},
			"display_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Human-friendly display name of the project.",
			},
			"organization": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Organization ID this project belongs to.",
			},
			"user_labels": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "User-defined labels attached to the project.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"project_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "System-assigned unique identifier (UUID) of the project.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the project was created (RFC3339 format).",
			},
		},
	}
}

func dataSourceProjectRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	name := d.Get("name").(string)

	project, err := config.Client.IAM().Projects().Get(ctx, name)
	if err != nil {
		return diag.Errorf("error reading project %s: %s", name, err)
	}

	d.SetId(project.Metadata.Id)

	diags = setDiag(d, "name", project.Metadata.Id, diags)
	diags = setDiag(d, "project_id", project.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", project.Metadata.CreationTimestamp.Format(time.RFC3339), diags)
	diags = setDiag(d, "organization", project.Spec.Organization, diags)

	if project.Spec.Name != nil {
		diags = setDiag(d, "display_name", *project.Spec.Name, diags)
	}

	diags = setDiag(d, "user_labels", flattenLabels(project.Metadata.UserLabels), diags)

	return diags
}
