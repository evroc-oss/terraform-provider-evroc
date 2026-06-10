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

func resourceLBBackendService() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an evroc load balancer backend service resource. A backend service defines the port, protocol options, and backend pool reference for routing traffic.",

		CreateContext: resourceLBBackendServiceCreate,
		ReadContext:   resourceLBBackendServiceRead,
		UpdateContext: resourceLBBackendServiceUpdate,
		DeleteContext: resourceLBBackendServiceDelete,

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
				Description: "Name of the backend service. Must be unique within the project.",
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
				Description: "Region where the backend service is created. Defaults to provider region.",
			},
			"port": {
				Type:             schema.TypeInt,
				Required:         true,
				ValidateDiagFunc: validatePort(),
				Description:      "Backend port to forward traffic to on the target instances.",
			},
			"backend_pool_ref": {
				Type:             schema.TypeString,
				Required:         true,
				DiffSuppressFunc: suppressFQIDDiff,
				Description:      "Fully qualified reference to the backend pool (e.g., evroc_lb_backend_pool.my_pool.fqid).",
			},
			"proxy_protocol": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Enable PROXY protocol to pass the real client IP to backends.",
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
			"service_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier (UUID) of the backend service.",
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
				Description: "Timestamp when the backend service was created (RFC3339 format).",
			},
			"fqid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified resource ID (FQID). Use this to reference this resource from other resources.",
			},
		},
	}
}

func resourceLBBackendServiceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)
	port := int32(d.Get("port").(int))
	backendPoolRef := d.Get("backend_pool_ref").(string)
	proxyProtocol := d.Get("proxy_protocol").(bool)

	var userLabels map[string]string
	if labels, ok := d.GetOk("user_labels"); ok {
		userLabels = make(map[string]string)
		for k, v := range labels.(map[string]interface{}) {
			userLabels[k] = v.(string)
		}
	}

	req := BuildBackendServiceCreateRequest(name, port, backendPoolRef, proxyProtocol, userLabels)

	svc, err := client.LoadBalancer().BackendServices().Create(ctx, req)
	if err != nil {
		return diag.Errorf("error creating backend service %s: %s", name, err)
	}

	d.SetId(svc.Metadata.Id)
	d.Set("project", resolveProject(d, config))

	return resourceLBBackendServiceRead(ctx, d, meta)
}

func resourceLBBackendServiceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	svc, err := client.LoadBalancer().BackendServices().Get(ctx, d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading backend service: %s", err)
	}

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

	if svc.Metadata.UserLabels != nil && len(*svc.Metadata.UserLabels) > 0 {
		diags = setDiag(d, "user_labels", flattenLabels(svc.Metadata.UserLabels), diags)
	}

	diags = setDiag(d, "system_labels", flattenLabels(svc.Metadata.SystemLabels), diags)
	diags = setDiag(d, "fqid", client.LoadBalancer().BackendServiceRef(svc.Metadata.Id), diags)

	return diags
}

func resourceLBBackendServiceUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	lbClient := client.LoadBalancer()

	svc, err := lbClient.BackendServices().Get(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error reading backend service %s: %s", d.Id(), err)
	}

	if d.HasChange("port") {
		svc.Spec.Port = int32(d.Get("port").(int))
	}

	if d.HasChange("backend_pool_ref") {
		ref := d.Get("backend_pool_ref").(string)
		svc.Spec.BackendPoolRef = &ref
	}

	if d.HasChange("proxy_protocol") {
		pp := d.Get("proxy_protocol").(bool)
		svc.Spec.ProxyProtocol = &pp
	}

	if d.HasChange("user_labels") {
		if labels, ok := d.GetOk("user_labels"); ok {
			userLabels := make(lbtypes.UserLabels)
			for k, v := range labels.(map[string]interface{}) {
				userLabels[k] = v.(string)
			}
			svc.Metadata.UserLabels = &userLabels
		} else {
			svc.Metadata.UserLabels = nil
		}
	}

	if _, err := lbClient.BackendServices().Patch(ctx, d.Id(), svc); err != nil {
		return diag.Errorf("error updating backend service %s: %s", d.Id(), err)
	}

	return resourceLBBackendServiceRead(ctx, d, meta)
}

func resourceLBBackendServiceDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	err := client.LoadBalancer().BackendServices().Delete(ctx, d.Id())
	if err != nil {
		if !isNotFoundError(err) {
			return diag.Errorf("error deleting backend service %s: %s", d.Id(), err)
		}
		d.SetId("")
		return nil
	}

	timeout := d.Timeout(schema.TimeoutDelete)
	if err := client.LoadBalancer().BackendServices().WaitForDeleted(ctx, d.Id(), timeout); err != nil {
		return diag.Errorf("error waiting for backend service %s to be deleted: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}
