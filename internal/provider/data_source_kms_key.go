// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceKMSKey() *schema.Resource {
	fields := kmsKeySchema()
	for name, field := range fields {
		field.ForceNew = false
		if name == "name" || name == "project" {
			continue
		}
		field.Required = false
		field.Optional = false
		field.Computed = true
		field.Default = nil
		field.ValidateDiagFunc = nil
	}
	fields["algorithm"].Description = "Encryption algorithm reported by KMS."
	fields["auto_rotate_after_days"].Description = "Automatic rotation interval in days; 0 when unset."
	fields["expire_after_days"].Description = "Key-version expiry interval in days; 0 when unset."
	fields["enabled"].Description = "True when the observed key state is Active."
	fields["imported"].Description = "Whether the key uses externally imported material."
	return &schema.Resource{Description: "Looks up a generated or externally imported KMS key in the provider region, including lifecycle state and version metadata. No key material is exposed.", Schema: fields, ReadContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		client, diags := resolveClient(d, meta.(*ProviderConfig))
		if diags.HasError() {
			return diags
		}
		key, err := client.KMS().Keys().Get(ctx, d.Get("name").(string))
		if err != nil {
			return diag.Errorf("error reading KMS key: %s", err)
		}
		d.SetId(key.Metadata.Id)
		return flattenKMSKey(d, client, key)
	}}
}
