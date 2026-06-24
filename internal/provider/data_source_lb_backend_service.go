// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceLBBackendService() *schema.Resource {
	return &schema.Resource{
		Description: "Get information about an existing evroc load balancer backend service.",

		ReadContext: dataSourceLBBackendServiceRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the backend service to look up.",
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
				Description: "Region where the backend service is located.",
			},
			// Computed fields
			"service_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier (UUID) of the backend service.",
			},
			"port": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Backend port to forward traffic to.",
			},
			"backend_pool_ref": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified reference to the backend pool.",
			},
			"proxy_protocol": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether PROXY protocol is enabled.",
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
				Description: "Timestamp when the backend service was created (RFC3339 format).",
			},
			"fqid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified resource ID (FQID).",
			},
			"backend_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of resolved backend addresses for this service.",
			},
			"health_check": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Active health check configuration for this backend service.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"interval":            {Type: schema.TypeString, Computed: true},
						"timeout":             {Type: schema.TypeString, Computed: true},
						"target_port":         {Type: schema.TypeInt, Computed: true},
						"healthy_threshold":   {Type: schema.TypeInt, Computed: true},
						"unhealthy_threshold": {Type: schema.TypeInt, Computed: true},
						"tcp": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"send":    {Type: schema.TypeString, Computed: true},
									"receive": {Type: schema.TypeString, Computed: true},
								},
							},
						},
						"http": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"path":              {Type: schema.TypeString, Computed: true},
									"method":            {Type: schema.TypeString, Computed: true},
									"host":              {Type: schema.TypeString, Computed: true},
									"expected_statuses": {Type: schema.TypeList, Computed: true, Elem: &schema.Schema{Type: schema.TypeInt}},
								},
							},
						},
						"https": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"path":                 {Type: schema.TypeString, Computed: true},
									"method":               {Type: schema.TypeString, Computed: true},
									"host":                 {Type: schema.TypeString, Computed: true},
									"expected_statuses":    {Type: schema.TypeList, Computed: true, Elem: &schema.Schema{Type: schema.TypeInt}},
									"insecure_skip_verify": {Type: schema.TypeBool, Computed: true},
								},
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceLBBackendServiceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	var diags diag.Diagnostics

	name := d.Get("name").(string)

	svc, err := client.LoadBalancer().BackendServices().Get(ctx, name)
	if err != nil {
		return diag.Errorf("error reading backend service %s: %s", name, err)
	}

	d.SetId(svc.Metadata.Id)
	diags = setDiag(d, "name", svc.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "region", derefString(svc.Metadata.Region), diags)
	diags = setDiag(d, "service_id", svc.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", svc.Metadata.CreationTimestamp.Format(time.RFC3339), diags)
	diags = setDiag(d, "port", int(svc.Spec.Port), diags)
	diags = setDiag(d, "backend_pool_ref", derefString(svc.Spec.BackendPoolRef), diags)

	proxyProtocol := false
	if svc.Spec.ProxyProtocol != nil {
		proxyProtocol = *svc.Spec.ProxyProtocol
	}
	diags = setDiag(d, "proxy_protocol", proxyProtocol, diags)

	diags = setDiag(d, "user_labels", flattenLabels(svc.Metadata.UserLabels), diags)
	diags = setDiag(d, "system_labels", flattenLabels(svc.Metadata.SystemLabels), diags)
	diags = setDiag(d, "fqid", client.LoadBalancer().BackendServiceRef(svc.Metadata.Id), diags)

	diags = setDiag(d, "backend_count", svc.Status.Backends, diags)

	if svc.Spec.HealthCheck != nil {
		diags = setDiag(d, "health_check", flattenHealthCheck(svc.Spec.HealthCheck), diags)
	}

	return diags
}
