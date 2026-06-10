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

func resourceLBBackendPool() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an evroc load balancer backend pool resource. A backend pool groups VM references that receive traffic from backend services.",

		CreateContext: resourceLBBackendPoolCreate,
		ReadContext:   resourceLBBackendPoolRead,
		UpdateContext: resourceLBBackendPoolUpdate,
		DeleteContext: resourceLBBackendPoolDelete,

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
				Description: "Name of the backend pool. Must be unique within the project.",
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
				Description: "Region where the backend pool is created. Defaults to provider region.",
			},
			"backend_refs": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Set of fully qualified VM references to use as backends (e.g., evroc_virtual_machine.my_vm.fqid).",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
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
			"pool_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier (UUID) of the backend pool.",
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
				Description: "Timestamp when the backend pool was created (RFC3339 format).",
			},
			"fqid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified resource ID (FQID). Use this to reference this resource from other resources.",
			},
		},
	}
}

func resourceLBBackendPoolCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)

	var backendRefs []string
	if refs, ok := d.GetOk("backend_refs"); ok {
		for _, r := range refs.(*schema.Set).List() {
			backendRefs = append(backendRefs, r.(string))
		}
	}

	var userLabels map[string]string
	if labels, ok := d.GetOk("user_labels"); ok {
		userLabels = make(map[string]string)
		for k, v := range labels.(map[string]interface{}) {
			userLabels[k] = v.(string)
		}
	}

	req := BuildBackendPoolCreateRequest(name, backendRefs, userLabels)

	pool, err := client.LoadBalancer().BackendPools().Create(ctx, req)
	if err != nil {
		return diag.Errorf("error creating backend pool %s: %s", name, err)
	}

	d.SetId(pool.Metadata.Id)
	d.Set("project", resolveProject(d, config))

	return resourceLBBackendPoolRead(ctx, d, meta)
}

func resourceLBBackendPoolRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	pool, err := client.LoadBalancer().BackendPools().Get(ctx, d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading backend pool: %s", err)
	}

	diags = setDiag(d, "name", pool.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "region", derefString(pool.Metadata.Region), diags)
	diags = setDiag(d, "pool_id", pool.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", pool.Metadata.CreationTimestamp.Format(time.RFC3339), diags)

	if pool.Spec.BackendRefs != nil && len(*pool.Spec.BackendRefs) > 0 {
		diags = setDiag(d, "backend_refs", *pool.Spec.BackendRefs, diags)
	}

	if pool.Metadata.UserLabels != nil && len(*pool.Metadata.UserLabels) > 0 {
		diags = setDiag(d, "user_labels", flattenLabels(pool.Metadata.UserLabels), diags)
	}

	diags = setDiag(d, "system_labels", flattenLabels(pool.Metadata.SystemLabels), diags)
	diags = setDiag(d, "fqid", client.LoadBalancer().BackendPoolRef(pool.Metadata.Id), diags)

	return diags
}

func resourceLBBackendPoolUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	lbClient := client.LoadBalancer()

	if d.HasChange("backend_refs") {
		var desiredRefs []string
		if refs, ok := d.GetOk("backend_refs"); ok {
			for _, r := range refs.(*schema.Set).List() {
				desiredRefs = append(desiredRefs, r.(string))
			}
		}
		_, err := lbClient.BackendPools().Patch(ctx, d.Id(), map[string]interface{}{
			"spec": lbtypes.BackendpoolSpec{BackendRefs: &desiredRefs},
		})
		if err != nil {
			return diag.Errorf("error updating backend pool %s backends: %s", d.Id(), err)
		}
	}

	if d.HasChange("user_labels") {
		pool, err := lbClient.BackendPools().Get(ctx, d.Id())
		if err != nil {
			return diag.Errorf("error reading backend pool %s: %s", d.Id(), err)
		}

		if labels, ok := d.GetOk("user_labels"); ok {
			userLabels := make(lbtypes.UserLabels)
			for k, v := range labels.(map[string]interface{}) {
				userLabels[k] = v.(string)
			}
			pool.Metadata.UserLabels = &userLabels
		} else {
			pool.Metadata.UserLabels = nil
		}

		if _, err := lbClient.BackendPools().Patch(ctx, d.Id(), pool); err != nil {
			return diag.Errorf("error updating backend pool %s labels: %s", d.Id(), err)
		}
	}

	return resourceLBBackendPoolRead(ctx, d, meta)
}

func resourceLBBackendPoolDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	err := client.LoadBalancer().BackendPools().Delete(ctx, d.Id())
	if err != nil {
		if !isNotFoundError(err) {
			return diag.Errorf("error deleting backend pool %s: %s", d.Id(), err)
		}
		d.SetId("")
		return nil
	}

	timeout := d.Timeout(schema.TimeoutDelete)
	if err := client.LoadBalancer().BackendPools().WaitForDeleted(ctx, d.Id(), timeout); err != nil {
		return diag.Errorf("error waiting for backend pool %s to be deleted: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}
