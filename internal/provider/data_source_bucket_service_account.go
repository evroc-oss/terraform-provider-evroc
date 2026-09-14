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
		Description: "Retrieves information and S3-compatible access credentials for an existing evroc bucket service account. Credential values are sensitive but are stored in state.",

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
				Description: "Identifier of the generated S3 credentials. Retained for compatibility with the deprecated `evroc_bucket_service_account_secret` data source.",
			},
			"access_key_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "S3 access key ID.",
			},
			"secret_access_key": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "S3 secret access key.",
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
		credentialsSecret := *sa.Status.S3CredentialsSecretName
		diags = setDiag(d, "credentials_secret", credentialsSecret, diags)

		secret, err := client.Storage().BucketServiceAccountSecrets().Get(ctx, credentialsSecret)
		if err != nil {
			return diag.Errorf("error reading credentials for bucket service account %s: %s", d.Id(), err)
		}
		if secret.Data.AccessKeyID != nil {
			diags = setDiag(d, "access_key_id", *secret.Data.AccessKeyID, diags)
		}
		if secret.Data.SecretAccessKey != nil {
			diags = setDiag(d, "secret_access_key", *secret.Data.SecretAccessKey, diags)
		}
	}

	return diags
}
