// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourcePermissionSet() *schema.Resource {
	return &schema.Resource{
		Description: "Retrieves information about an existing evroc permission set.",

		ReadContext: dataSourcePermissionSetRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the permission set to look up.",
			},
			"project": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Project ID this permission set belongs to.",
			},
			"email": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Email address of the user.",
			},
			"admin": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether this permission set grants admin privileges.",
			},
			"user_labels": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "User-defined labels attached to the permission set.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"permission_set_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "System-assigned unique identifier (UUID) of the permission set.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the permission set was created (RFC3339 format).",
			},
		},
	}
}

func dataSourcePermissionSetRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	name := d.Get("name").(string)

	ps, err := config.Client.IAM().PermissionSets().Get(ctx, name)
	if err != nil {
		return diag.Errorf("error reading permission set %s: %s", name, err)
	}

	d.SetId(ps.Metadata.Id)

	diags = setDiag(d, "name", ps.Metadata.Id, diags)
	diags = setDiag(d, "permission_set_id", ps.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", ps.Metadata.CreationTimestamp.Format(time.RFC3339), diags)
	diags = setDiag(d, "email", ps.Spec.Subject.User.Email, diags)
	diags = setDiag(d, "admin", ps.Spec.Admin, diags)
	diags = setDiag(d, "user_labels", flattenLabels(ps.Metadata.UserLabels), diags)

	return diags
}
