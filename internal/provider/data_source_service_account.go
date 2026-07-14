// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceServiceAccount() *schema.Resource {
	return &schema.Resource{
		Description: "Retrieves information about an existing evroc IAM service account.",

		ReadContext: dataSourceServiceAccountRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the service account to look up.",
			},
			"project": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Project this service account belongs to.",
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Human-readable description of the service account's purpose.",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the service account is enabled.",
			},
			"user_labels": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "User-defined labels attached to the service account.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"service_account_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "System-assigned unique identifier (UUID) of the service account.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the service account was created (RFC3339 format).",
			},
			"oauth_client_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "OAuth client ID assigned to the service account.",
			},
			"fqid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified identifier for the service account.",
			},
		},
	}
}

func dataSourceServiceAccountRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	name := d.Get("name").(string)

	sa, err := config.Client.IAM().ServiceAccounts().Get(ctx, name)
	if err != nil {
		return diag.Errorf("error reading service account %s: %s", name, err)
	}

	d.SetId(sa.Metadata.Id)

	diags = setDiag(d, "name", sa.Metadata.Id, diags)
	diags = setDiag(d, "service_account_id", sa.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", sa.Metadata.CreationTimestamp.Format(time.RFC3339), diags)

	if sa.Spec.Description != nil {
		diags = setDiag(d, "description", *sa.Spec.Description, diags)
	}

	enabled := true
	if sa.Spec.Enabled != nil {
		enabled = *sa.Spec.Enabled
	}
	diags = setDiag(d, "enabled", enabled, diags)

	if sa.Status.OauthClientId != nil {
		diags = setDiag(d, "oauth_client_id", *sa.Status.OauthClientId, diags)
	}

	diags = setDiag(d, "user_labels", flattenLabels(sa.Metadata.UserLabels), diags)

	fqid := fmt.Sprintf("/iam/projects/%s/serviceAccounts/%s", config.Project, sa.Metadata.Id)
	diags = setDiag(d, "fqid", fqid, diags)

	return diags
}
