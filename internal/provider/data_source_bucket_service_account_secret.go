// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceBucketServiceAccountSecret() *schema.Resource {
	return &schema.Resource{
		Description:        "Deprecated. Use the sensitive `access_key_id` and `secret_access_key` attributes on `evroc_bucket_service_account` instead.",
		DeprecationMessage: "The evroc_bucket_service_account_secret data source is deprecated; use evroc_bucket_service_account.access_key_id and evroc_bucket_service_account.secret_access_key instead.",

		ReadContext: dataSourceBucketServiceAccountSecretRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Credential identifier returned by `evroc_bucket_service_account.credentials_secret`. This is not the bucket service account name.",
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
		},
	}
}

func dataSourceBucketServiceAccountSecretRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	name := d.Get("name").(string)

	secret, err := config.Client.Storage().BucketServiceAccountSecrets().Get(ctx, name)
	if err != nil {
		return diag.Errorf("error reading bucket service account secret %s: %s", name, err)
	}

	d.SetId(name)

	if secret.Data.AccessKeyID != nil {
		diags = setDiag(d, "access_key_id", *secret.Data.AccessKeyID, diags)
	}
	if secret.Data.SecretAccessKey != nil {
		diags = setDiag(d, "secret_access_key", *secret.Data.SecretAccessKey, diags)
	}

	return diags
}
