// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceLBBackendPool() *schema.Resource {
	return &schema.Resource{
		Description: "Get information about an existing evroc load balancer backend pool.",

		ReadContext: dataSourceLBBackendPoolRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the backend pool to look up.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Project this resource belongs to. Defaults to the provider project.",
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Region where the backend pool is located.",
			},
			// Computed fields
			"pool_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier (UUID) of the backend pool.",
			},
			"backend_refs": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "VM references in the backend pool.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"user_labels": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "User-defined labels (key/value pairs).",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"system_labels": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "System-managed labels (read-only).",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the backend pool was created (RFC3339 format).",
			},
			"fqid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified resource ID (FQID).",
			},
		},
	}
}

func dataSourceLBBackendPoolRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	var diags diag.Diagnostics

	name := d.Get("name").(string)

	pool, err := client.LoadBalancer().BackendPools().Get(ctx, name)
	if err != nil {
		return diag.Errorf("error reading backend pool %s: %s", name, err)
	}

	d.SetId(pool.Metadata.Id)
	diags = setDiag(d, "name", pool.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "region", derefString(pool.Metadata.Region), diags)
	diags = setDiag(d, "pool_id", pool.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", pool.Metadata.CreationTimestamp.Format(time.RFC3339), diags)

	if pool.Spec.BackendRefs != nil {
		diags = setDiag(d, "backend_refs", *pool.Spec.BackendRefs, diags)
	}

	diags = setDiag(d, "user_labels", flattenLabels(pool.Metadata.UserLabels), diags)
	diags = setDiag(d, "system_labels", flattenLabels(pool.Metadata.SystemLabels), diags)
	diags = setDiag(d, "fqid", client.LoadBalancer().BackendPoolRef(pool.Metadata.Id), diags)

	return diags
}
