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
		Description: "Retrieves S3-compatible credentials for a bucket service account. These credentials can be used to access evroc object storage from any S3-compatible client.",

		ReadContext: dataSourceBucketServiceAccountSecretRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the bucket service account.",
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
