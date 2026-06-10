// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceLoadBalancer() *schema.Resource {
	return &schema.Resource{
		Description: "Get information about an existing evroc load balancer.",

		ReadContext: dataSourceLoadBalancerRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the load balancer to look up.",
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
				Description: "Region where the load balancer is located.",
			},
			// Computed fields
			"lb_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier (UUID) of the load balancer.",
			},
			"public_ip_ref": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified reference to the public IP for the load balancer.",
			},
			"listener": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of listeners (port mappings) for the load balancer.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Name for this listener.",
						},
						"protocol": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Protocol for the listener.",
						},
						"port": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Frontend port that the load balancer listens on.",
						},
						"route_refs": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "L4 route references for this listener.",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
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
				Description: "Timestamp when the load balancer was created (RFC3339 format).",
			},
			"fqid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified resource ID (FQID).",
			},
		},
	}
}

func dataSourceLoadBalancerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	var diags diag.Diagnostics

	name := d.Get("name").(string)

	lb, err := client.LoadBalancer().LoadBalancers().Get(ctx, name)
	if err != nil {
		return diag.Errorf("error reading load balancer %s: %s", name, err)
	}

	d.SetId(lb.Metadata.Id)
	diags = setDiag(d, "name", lb.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "region", derefString(lb.Metadata.Region), diags)
	diags = setDiag(d, "lb_id", lb.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", lb.Metadata.CreationTimestamp.Format(time.RFC3339), diags)
	diags = setDiag(d, "public_ip_ref", lb.Spec.PublicIPRef, diags)
	diags = setDiag(d, "listener", flattenLoadBalancerListeners(lb.Spec.Listeners), diags)

	diags = setDiag(d, "user_labels", flattenLabels(lb.Metadata.UserLabels), diags)
	diags = setDiag(d, "system_labels", flattenLabels(lb.Metadata.SystemLabels), diags)
	diags = setDiag(d, "fqid", client.LoadBalancer().LoadBalancerRef(lb.Metadata.Id), diags)

	return diags
}
