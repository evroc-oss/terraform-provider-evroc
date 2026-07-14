// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceOrganizationQuota() *schema.Resource {
	return &schema.Resource{
		Description: "Retrieves quota limits and current usage for the organization.",

		ReadContext: dataSourceOrganizationQuotaRead,

		Schema: map[string]*schema.Schema{
			// Compute limits
			"compute_vcpus": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Maximum vCPUs allowed.",
			},
			"compute_memory": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Maximum memory allowed (e.g., \"600 GB\").",
			},
			"compute_block_storage": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Maximum block storage allowed (e.g., \"600 GB\").",
			},
			// Networking limits
			"networking_public_ips": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Maximum public IPs allowed.",
			},
			// Load balancer limits
			"load_balancer_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Maximum load balancers allowed.",
			},
			// Usage
			"usage_vcpus": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Current vCPU usage.",
			},
			"usage_memory": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Current memory usage.",
			},
			"usage_block_storage": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Current block storage usage.",
			},
			"usage_public_ips": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Current public IP usage.",
			},
			"usage_load_balancers": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Current load balancer usage.",
			},
		},
	}
}

func dataSourceOrganizationQuotaRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	quota, err := config.Client.Quotas().OrganizationQuotas().Get(ctx)
	if err != nil {
		return diag.Errorf("error reading organization quota: %s", err)
	}

	d.SetId("org-quota")

	if q := quota.Spec.Quotas; q != nil {
		if q.Compute != nil {
			if q.Compute.VCPUs != nil {
				diags = setDiag(d, "compute_vcpus", int(*q.Compute.VCPUs), diags)
			}
			if q.Compute.Memory != nil {
				diags = setDiag(d, "compute_memory", *q.Compute.Memory, diags)
			}
			if q.Compute.BlockStorage != nil {
				diags = setDiag(d, "compute_block_storage", *q.Compute.BlockStorage, diags)
			}
		}
		if q.Networking != nil && q.Networking.PublicIPs != nil {
			diags = setDiag(d, "networking_public_ips", int(*q.Networking.PublicIPs), diags)
		}
		if q.LoadBalancer != nil && q.LoadBalancer.LoadBalancers != nil {
			diags = setDiag(d, "load_balancer_count", int(*q.LoadBalancer.LoadBalancers), diags)
		}
	}

	if u := quota.Status.QuotaUsage; u != nil {
		if u.Compute != nil {
			if u.Compute.VCPUs != nil {
				diags = setDiag(d, "usage_vcpus", int(*u.Compute.VCPUs), diags)
			}
			if u.Compute.Memory != nil {
				diags = setDiag(d, "usage_memory", *u.Compute.Memory, diags)
			}
			if u.Compute.BlockStorage != nil {
				diags = setDiag(d, "usage_block_storage", *u.Compute.BlockStorage, diags)
			}
		}
		if u.Networking != nil && u.Networking.PublicIPs != nil {
			diags = setDiag(d, "usage_public_ips", int(*u.Networking.PublicIPs), diags)
		}
		if u.LoadBalancer != nil && u.LoadBalancer.LoadBalancers != nil {
			diags = setDiag(d, "usage_load_balancers", int(*u.LoadBalancer.LoadBalancers), diags)
		}
	}

	return diags
}
