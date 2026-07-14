// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceProjectQuota() *schema.Resource {
	return &schema.Resource{
		Description: "Retrieves quota limits and current usage for a project.",

		ReadContext: dataSourceProjectQuotaRead,

		Schema: map[string]*schema.Schema{
			"object_storage_total_size": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Maximum total object storage size (e.g., \"5TB\" or \"unlimited\").",
			},
			"object_storage_usage": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Current object storage usage.",
			},
		},
	}
}

func dataSourceProjectQuotaRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	quota, err := config.Client.Quotas().ProjectQuotas().Get(ctx)
	if err != nil {
		return diag.Errorf("error reading project quota: %s", err)
	}

	d.SetId("project-quota")

	if quota.Spec.ObjectStorage != nil && quota.Spec.ObjectStorage.TotalSize != nil {
		diags = setDiag(d, "object_storage_total_size", *quota.Spec.ObjectStorage.TotalSize, diags)
	}
	if quota.Status.QuotaUsage != nil && quota.Status.QuotaUsage.ObjectStorage != nil && quota.Status.QuotaUsage.ObjectStorage.TotalSize != nil {
		diags = setDiag(d, "object_storage_usage", *quota.Status.QuotaUsage.ObjectStorage.TotalSize, diags)
	}

	return diags
}
