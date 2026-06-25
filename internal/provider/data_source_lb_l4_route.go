// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceLBL4Route() *schema.Resource {
	return &schema.Resource{
		Description: "Get information about an existing evroc load balancer L4 route.",

		ReadContext: dataSourceLBL4RouteRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the L4 route to look up.",
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
				Description: "Region where the L4 route is located.",
			},
			// Computed fields
			"route_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier (UUID) of the L4 route.",
			},
			"default_backend_service_ref": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified reference to the backend service.",
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
				Description: "Timestamp when the L4 route was created (RFC3339 format).",
			},
			"fqid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified resource ID (FQID).",
			},
		},
	}
}

func dataSourceLBL4RouteRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	var diags diag.Diagnostics

	name := d.Get("name").(string)

	route, err := client.LoadBalancer().L4Routes().Get(ctx, name)
	if err != nil {
		return diag.Errorf("error reading L4 route %s: %s", name, err)
	}

	d.SetId(route.Metadata.Id)
	diags = setDiag(d, "name", route.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "region", derefString(route.Metadata.Region), diags)
	diags = setDiag(d, "route_id", route.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", route.Metadata.CreationTimestamp.Format(time.RFC3339), diags)
	diags = setDiag(d, "default_backend_service_ref", route.Spec.DefaultBackendServiceRef, diags)

	diags = setDiag(d, "user_labels", flattenLabels(route.Metadata.UserLabels), diags)
	diags = setDiag(d, "system_labels", flattenLabels(route.Metadata.SystemLabels), diags)
	diags = setDiag(d, "fqid", client.LoadBalancer().L4RouteRef(route.Metadata.Id), diags)

	return diags
}
