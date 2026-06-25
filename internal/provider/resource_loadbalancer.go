// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import (
	"context"
	"time"

	"github.com/evroc-oss/evroc-go-sdk/loadbalancer"
	lbtypes "github.com/evroc-oss/evroc-go-sdk/types/loadbalancer"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func loadbalancerListenerResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Name for this listener (e.g., 'web', 'api'). Must be unique per load balancer.",
			},
			"protocol": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Protocol for the listener. Currently only 'TCP' is supported.",
			},
			"port": {
				Type:             schema.TypeInt,
				Required:         true,
				ValidateDiagFunc: validatePort(),
				Description:      "Frontend port that the load balancer listens on.",
			},
			"route_refs": {
				Type:        schema.TypeSet,
				Required:    true,
				Description: "Set of fully qualified L4 route references (e.g., evroc_lb_l4_route.my_route.fqid).",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceLoadBalancer() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an evroc Layer 4 (TCP) load balancer resource for distributing traffic across backend targets.",

		CreateContext: resourceLoadBalancerCreate,
		ReadContext:   resourceLoadBalancerRead,
		UpdateContext: resourceLoadBalancerUpdate,
		DeleteContext: resourceLoadBalancerDelete,

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
				Description: "Name of the load balancer. Must be unique within the project.",
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
				ForceNew:    true,
				Computed:    true,
				Description: "Region where the load balancer is created. Defaults to provider region.",
			},
			"public_ip_ref": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				DiffSuppressFunc: suppressFQIDDiff,
				Description:      "Fully qualified reference to the public IP for the load balancer (e.g., evroc_public_ip.my_ip.fqid).",
			},
			"listener": {
				Type:        schema.TypeSet,
				Required:    true,
				MinItems:    1,
				Description: "List of listeners (port mappings) for the load balancer.",
				Set:         schema.HashResource(loadbalancerListenerResource()),
				Elem:        loadbalancerListenerResource(),
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
			"lb_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier (UUID) of the load balancer.",
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
				Description: "Timestamp when the load balancer was created (RFC3339 format).",
			},
			"fqid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified resource ID (FQID). Use this to reference this resource from other resources.",
			},
			"public_ipv4_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Assigned public IPv4 address of the load balancer.",
			},
		},
	}
}

func resourceLoadBalancerCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)
	publicIPRef := d.Get("public_ip_ref").(string)
	listeners := expandLoadBalancerListeners(d.Get("listener").(*schema.Set))

	var userLabels map[string]string
	if labels, ok := d.GetOk("user_labels"); ok {
		userLabels = make(map[string]string)
		for k, v := range labels.(map[string]interface{}) {
			userLabels[k] = v.(string)
		}
	}

	builder := loadbalancer.NewLoadBalancerBuilder(name).
		WithPublicIPRef(publicIPRef)

	if len(userLabels) > 0 {
		builder = builder.WithLabels(userLabels)
	}

	for _, l := range listeners {
		builder = builder.WithListener(l)
	}

	lb, err := builder.Create(ctx, client.LoadBalancer().LoadBalancers())
	if err != nil {
		return diag.Errorf("error creating load balancer %s: %s", name, err)
	}

	d.SetId(lb.Metadata.Id)
	d.Set("project", resolveProject(d, config))

	timeout := d.Timeout(schema.TimeoutCreate)
	readyLB, err := client.LoadBalancer().LoadBalancers().WaitForReady(ctx, name, timeout)
	if err != nil {
		return diag.Errorf("error waiting for load balancer %s to be ready: %s", name, err)
	}

	d.SetId(readyLB.Metadata.Id)

	return resourceLoadBalancerRead(ctx, d, meta)
}

func resourceLoadBalancerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	lb, err := client.LoadBalancer().LoadBalancers().Get(ctx, d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading load balancer: %s", err)
	}

	diags = setDiag(d, "name", lb.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "region", derefString(lb.Metadata.Region), diags)
	diags = setDiag(d, "lb_id", lb.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", lb.Metadata.CreationTimestamp.Format(time.RFC3339), diags)
	diags = setDiag(d, "public_ip_ref", lb.Spec.PublicIPRef, diags)
	diags = setDiag(d, "listener", flattenLoadBalancerListeners(lb.Spec.Listeners), diags)

	if lb.Metadata.UserLabels != nil && len(*lb.Metadata.UserLabels) > 0 {
		diags = setDiag(d, "user_labels", flattenLabels(lb.Metadata.UserLabels), diags)
	}

	diags = setDiag(d, "system_labels", flattenLabels(lb.Metadata.SystemLabels), diags)
	diags = setDiag(d, "fqid", client.LoadBalancer().LoadBalancerRef(lb.Metadata.Id), diags)

	if lb.Status.Networking != nil && lb.Status.Networking.PublicIPv4Address != nil {
		diags = setDiag(d, "public_ipv4_address", *lb.Status.Networking.PublicIPv4Address, diags)
	} else {
		diags = setDiag(d, "public_ipv4_address", "", diags)
	}

	return diags
}

func resourceLoadBalancerUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	lbClient := client.LoadBalancer()

	lb, err := lbClient.LoadBalancers().Get(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error reading load balancer %s: %s", d.Id(), err)
	}

	if d.HasChange("listener") {
		listeners := expandLoadBalancerListeners(d.Get("listener").(*schema.Set))
		lb.Spec.Listeners = &listeners
	}

	if d.HasChange("user_labels") {
		if labels, ok := d.GetOk("user_labels"); ok {
			userLabels := make(lbtypes.UserLabels)
			for k, v := range labels.(map[string]interface{}) {
				userLabels[k] = v.(string)
			}
			lb.Metadata.UserLabels = &userLabels
		} else {
			lb.Metadata.UserLabels = nil
		}
	}

	if _, err := lbClient.LoadBalancers().Patch(ctx, d.Id(), lb); err != nil {
		return diag.Errorf("error updating load balancer %s: %s", d.Id(), err)
	}

	return resourceLoadBalancerRead(ctx, d, meta)
}

func resourceLoadBalancerDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	err := client.LoadBalancer().LoadBalancers().Delete(ctx, d.Id())
	if err != nil {
		if !isNotFoundError(err) {
			return diag.Errorf("error deleting load balancer %s: %s", d.Id(), err)
		}
		d.SetId("")
		return nil
	}

	timeout := d.Timeout(schema.TimeoutDelete)
	if err := client.LoadBalancer().LoadBalancers().WaitForDeleted(ctx, d.Id(), timeout); err != nil {
		return diag.Errorf("error waiting for load balancer %s to be deleted: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}

func expandLoadBalancerListeners(listenersSet *schema.Set) []lbtypes.LoadbalancerSpecListenersItem {
	listeners := make([]lbtypes.LoadbalancerSpecListenersItem, 0, listenersSet.Len())

	for _, item := range listenersSet.List() {
		m := item.(map[string]interface{})

		l := lbtypes.LoadbalancerSpecListenersItem{
			Protocol: lbtypes.LoadbalancerSpecListenersItemProtocol(m["protocol"].(string)),
			Port:     int32(m["port"].(int)),
		}

		if name, ok := m["name"].(string); ok && name != "" {
			l.Name = &name
		}

		if routeRefs, ok := m["route_refs"]; ok {
			refs := make([]string, 0)
			for _, r := range routeRefs.(*schema.Set).List() {
				refs = append(refs, r.(string))
			}
			if len(refs) > 0 {
				l.RouteRefs = &refs
			}
		}

		listeners = append(listeners, l)
	}

	return listeners
}

func flattenLoadBalancerListeners(listeners *[]lbtypes.LoadbalancerSpecListenersItem) []interface{} {
	if listeners == nil {
		return nil
	}

	result := make([]interface{}, 0, len(*listeners))

	for _, l := range *listeners {
		m := map[string]interface{}{
			"protocol": string(l.Protocol),
			"port":     int(l.Port),
		}

		if l.Name != nil {
			m["name"] = *l.Name
		} else {
			m["name"] = ""
		}

		if l.RouteRefs != nil {
			m["route_refs"] = *l.RouteRefs
		} else {
			m["route_refs"] = []string{}
		}

		result = append(result, m)
	}

	return result
}
