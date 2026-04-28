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

func dataSourceBucketServiceAccount() *schema.Resource {
	return &schema.Resource{
		Description: "Retrieves information about an existing evroc bucket service account.",

		ReadContext: dataSourceBucketServiceAccountRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the service account to query.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Project this resource belongs to. Defaults to the provider project.",
			},
			"buckets": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of bucket names this service account can access.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"region": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Region where the service account is located.",
			},
			"service_account_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier of the service account.",
			},
			"credentials_secret": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the Kubernetes secret containing S3 credentials.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the service account was created.",
			},
		},
	}
}

func dataSourceBucketServiceAccountRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	var diags diag.Diagnostics
	name := d.Get("name").(string)

	sa, err := client.Storage().BucketServiceAccounts().Get(ctx, name)
	if err != nil {
		return diag.Errorf("error getting bucket service account: %s", err)
	}

	d.SetId(sa.Metadata.Id)
	diags = setDiag(d, "name", sa.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "region", derefString(sa.Metadata.Region), diags)
	diags = setDiag(d, "service_account_id", sa.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", sa.Metadata.CreationTimestamp.Format(time.RFC3339), diags)

	if sa.Spec.Buckets != nil {
		buckets := make([]string, 0, len(*sa.Spec.Buckets))
		for _, bucket := range *sa.Spec.Buckets {
			buckets = append(buckets, path.Base(bucket))
		}
		diags = setDiag(d, "buckets", buckets, diags)
	}

	if sa.Status.S3CredentialsSecretName != nil {
		diags = setDiag(d, "credentials_secret", *sa.Status.S3CredentialsSecretName, diags)
	}

	return diags
}
