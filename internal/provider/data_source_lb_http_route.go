// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceLBHTTPRoute() *schema.Resource {
	return &schema.Resource{
		Description: "Looks up an evroc load balancer HTTP route and its request matching rules.",
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "HTTP route name, unique within the project and region.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Project containing the route. Defaults to the provider project.",
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Region containing the route. Defaults to the provider region.",
			},
			"hostnames": {
				Type:        schema.TypeSet,
				Description: "Hostnames to match, including optional leading *. wildcards. Omit to accept all hostnames accepted by the listener.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed: true,
			},
			"rule": {
				Type:        schema.TypeList,
				Description: "Ordered routing rules. A rule without matches is a catch-all.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Description: "Unique rule name, stable when rules are reordered.",
							Computed:    true,
						},
						"backend_service_ref": {
							Type:        schema.TypeString,
							Description: "Backend service reference in the same project and region.",
							Computed:    true,
						},
						"match": {
							Type:        schema.TypeList,
							Description: "Alternative request matches (OR). All fields within one match must match (AND).",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"method": {
										Type:        schema.TypeString,
										Description: "HTTP request method to match. Omit to match any method.",
										Computed:    true,
									},
									"path": {
										Type:        schema.TypeList,
										Description: "Request path match. Omission matches the PathPrefix /.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"type": {
													Type:        schema.TypeString,
													Description: "Path matching type: Exact or PathPrefix. Prefixes match complete path segments.",
													Computed:    true,
												},
												"value": {
													Type:        schema.TypeString,
													Description: "Absolute request path to match.",
													Computed:    true,
												},
											},
										},
										Computed: true,
									},
									"header": {
										Description: "Request headers that must all match. Names are case-insensitive and must be unique.",
										Type:        schema.TypeSet,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"name": {
													Type:        schema.TypeString,
													Description: "Header or query parameter name.",
													Computed:    true,
												},
												"value": {
													Type:        schema.TypeString,
													Description: "Exact value to match.",
													Computed:    true,
												},
											},
										},
										Computed: true,
									},
									"query_param": {
										Description: "Request query parameters that must all match. Names must be unique.",
										Type:        schema.TypeSet,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"name": {
													Type:        schema.TypeString,
													Description: "Header or query parameter name.",
													Computed:    true,
												},
												"value": {
													Type:        schema.TypeString,
													Description: "Exact value to match.",
													Computed:    true,
												},
											},
										},
										Computed: true,
									},
								},
							},
							Computed: true,
						},
					},
				},
				Computed: true,
			},
			"user_labels": {
				Type:        schema.TypeMap,
				Description: "User-defined resource labels.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed: true,
			},
			"system_labels": {
				Type:        schema.TypeMap,
				Description: "System-managed resource labels.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed: true,
			},
			"route_id": {
				Type:        schema.TypeString,
				Description: "System-assigned route UUID.",
				Computed:    true,
			},
			"created_at": {
				Type:        schema.TypeString,
				Description: "Creation timestamp in RFC3339 format.",
				Computed:    true,
			},
			"fqid": {
				Type:        schema.TypeString,
				Description: "Fully qualified HTTP route reference.",
				Computed:    true,
			},
		},
		ReadContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
			client, diags := resolveClient(d, meta.(*ProviderConfig))
			if diags.HasError() {
				return diags
			}
			route, err := client.LoadBalancer().HTTPRoutes().Get(ctx, d.Get("name").(string))
			if err != nil {
				return diag.Errorf("error reading HTTP route: %s", err)
			}
			d.SetId(route.Metadata.Id)
			return flattenHTTPRoute(d, client, route)
		},
	}
}
