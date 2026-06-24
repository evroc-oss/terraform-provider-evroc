// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import (
	"context"
	"time"

	lbtypes "github.com/evroc-oss/evroc-go-sdk/types/loadbalancer"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceLBL4Route() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an evroc load balancer L4 route resource. An L4 route directs Layer 4 traffic to a backend service.",

		CreateContext: resourceLBL4RouteCreate,
		ReadContext:   resourceLBL4RouteRead,
		UpdateContext: resourceLBL4RouteUpdate,
		DeleteContext: resourceLBL4RouteDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the L4 route. Must be unique within the project.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Project this resource belongs to. Defaults to the provider project.",
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Region where the L4 route is created. Defaults to provider region.",
			},
			"default_backend_service_ref": {
				Type:             schema.TypeString,
				Required:         true,
				DiffSuppressFunc: suppressFQIDDiff,
				Description:      "Fully qualified reference to the backend service (e.g., evroc_lb_backend_service.my_svc.fqid).",
			},
			"user_labels": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "User-defined labels (key/value pairs) for organizing and selecting resources.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			// Computed fields
			"route_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier (UUID) of the L4 route.",
			},
			"system_labels": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "System-managed labels automatically set by evroc (read-only).",
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
				Description: "Fully qualified resource ID (FQID). Use this to reference this resource from other resources.",
			},
		},
	}
}

func resourceLBL4RouteCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)
	backendServiceRef := d.Get("default_backend_service_ref").(string)

	var userLabels map[string]string
	if labels, ok := d.GetOk("user_labels"); ok {
		userLabels = make(map[string]string)
		for k, v := range labels.(map[string]interface{}) {
			userLabels[k] = v.(string)
		}
	}

	req := BuildL4RouteCreateRequest(name, backendServiceRef, userLabels)

	route, err := client.LoadBalancer().L4Routes().Create(ctx, req)
	if err != nil {
		return diag.Errorf("error creating L4 route %s: %s", name, err)
	}

	d.SetId(route.Metadata.Id)
	d.Set("project", resolveProject(d, config))

	return resourceLBL4RouteRead(ctx, d, meta)
}

func resourceLBL4RouteRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	route, err := client.LoadBalancer().L4Routes().Get(ctx, d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading L4 route: %s", err)
	}

	diags = setDiag(d, "name", route.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "region", derefString(route.Metadata.Region), diags)
	diags = setDiag(d, "route_id", route.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", route.Metadata.CreationTimestamp.Format(time.RFC3339), diags)
	diags = setDiag(d, "default_backend_service_ref", route.Spec.DefaultBackendServiceRef, diags)

	if route.Metadata.UserLabels != nil && len(*route.Metadata.UserLabels) > 0 {
		diags = setDiag(d, "user_labels", flattenLabels(route.Metadata.UserLabels), diags)
	}

	diags = setDiag(d, "system_labels", flattenLabels(route.Metadata.SystemLabels), diags)
	diags = setDiag(d, "fqid", client.LoadBalancer().L4RouteRef(route.Metadata.Id), diags)

	return diags
}

func resourceLBL4RouteUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	lbClient := client.LoadBalancer()

	route, err := lbClient.L4Routes().Get(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error reading L4 route %s: %s", d.Id(), err)
	}

	if d.HasChange("default_backend_service_ref") {
		route.Spec.DefaultBackendServiceRef = d.Get("default_backend_service_ref").(string)
	}

	if d.HasChange("user_labels") {
		if labels, ok := d.GetOk("user_labels"); ok {
			userLabels := make(lbtypes.UserLabels)
			for k, v := range labels.(map[string]interface{}) {
				userLabels[k] = v.(string)
			}
			route.Metadata.UserLabels = &userLabels
		} else {
			route.Metadata.UserLabels = nil
		}
	}

	if _, err := lbClient.L4Routes().Patch(ctx, d.Id(), route); err != nil {
		return diag.Errorf("error updating L4 route %s: %s", d.Id(), err)
	}

	return resourceLBL4RouteRead(ctx, d, meta)
}

func resourceLBL4RouteDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	err := client.LoadBalancer().L4Routes().Delete(ctx, d.Id())
	if err != nil {
		if !isNotFoundError(err) {
			return diag.Errorf("error deleting L4 route %s: %s", d.Id(), err)
		}
		d.SetId("")
		return nil
	}

	timeout := d.Timeout(schema.TimeoutDelete)
	if err := client.LoadBalancer().L4Routes().WaitForDeleted(ctx, d.Id(), timeout); err != nil {
		return diag.Errorf("error waiting for L4 route %s to be deleted: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}
