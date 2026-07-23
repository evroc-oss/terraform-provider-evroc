// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceRoleBinding() *schema.Resource {
	return &schema.Resource{
		Description: "Retrieves information about an existing project-scoped IAM role binding.",

		ReadContext: dataSourceRoleBindingRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the role binding to look up.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Project the role binding belongs to. Defaults to the provider project.",
			},
			"principal": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Principal FQID.",
			},
			"roles": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     roleEntrySchema(),
			},
			"display_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Human-friendly display name for the binding.",
			},
			"user_labels": {
				Type:     schema.TypeMap,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"uid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "System-assigned unique identifier (UUID).",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the role binding was created (RFC3339 format).",
			},
		},
	}
}

func dataSourceRoleBindingRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	name := d.Get("name").(string)

	rb, err := client.IAM().RoleBindings().GetProjectRoleBinding(ctx, name)
	if err != nil {
		return diag.Errorf("error reading role binding %s: %s", name, err)
	}

	d.SetId(rb.Metadata.Id)

	diags = setDiag(d, "name", rb.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "uid", rb.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", rb.Metadata.CreationTimestamp.Format(time.RFC3339), diags)
	diags = setDiag(d, "principal", rb.Spec.Principal, diags)
	diags = setDiag(d, "roles", flattenRoleEntries(rb.Spec.Roles), diags)

	if rb.Spec.Name != nil {
		diags = setDiag(d, "display_name", *rb.Spec.Name, diags)
	}

	diags = setDiag(d, "user_labels", flattenLabels(rb.Metadata.UserLabels), diags)

	return diags
}
