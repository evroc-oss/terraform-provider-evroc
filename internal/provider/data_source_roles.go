// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceRoles() *schema.Resource {
	return &schema.Resource{
		Description: "Lists all available IAM roles from the role catalog.",

		ReadContext: dataSourceRolesRead,

		Schema: map[string]*schema.Schema{
			"roles": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Role FQID (e.g., /iam/roles/computeOperator).",
						},
						"description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Human-readable description of the role.",
						},
						"scope": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Role scope: \"organization\" or \"project\".",
						},
					},
				},
			},
		},
	}
}

func dataSourceRolesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	roles, err := config.Client.IAM().RoleBindings().ListRoles(ctx)
	if err != nil {
		return diag.Errorf("error listing roles: %s", err)
	}

	d.SetId("iam-roles")

	items := make([]map[string]interface{}, len(roles.Items))
	for i, r := range roles.Items {
		items[i] = map[string]interface{}{
			"id":          r.ID,
			"description": r.Description,
			"scope":       r.Scope,
		}
	}

	d.Set("roles", items)

	return nil
}
