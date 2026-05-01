// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"time"

	networkingtypes "github.com/evroc-oss/evroc-go-sdk/types/networking"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourcePublicIP() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an evroc public IP address resource.",

		CreateContext: resourcePublicIPCreate,
		ReadContext:   resourcePublicIPRead,
		UpdateContext: resourcePublicIPUpdate,
		DeleteContext: resourcePublicIPDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(5 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the public IP. Must be unique within the project.",
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
				Description: "Region where the public IP is allocated. Defaults to provider region.",
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
			"ip_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier (UUID) of the public IP.",
			},
			"system_labels": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "System-managed labels automatically set by evroc (read-only).",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"ip_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The allocated IPv4 address.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the public IP was created (RFC3339 format).",
			},
			"fqid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified resource ID (FQID). Use this to reference this resource from other resources.",
			},
		},
	}
}

func resourcePublicIPCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)

	// Get user labels
	var userLabels map[string]string
	if labels, ok := d.GetOk("user_labels"); ok {
		userLabels = make(map[string]string)
		for k, v := range labels.(map[string]interface{}) {
			userLabels[k] = v.(string)
		}
	}

	// Build public IP create request using SDK builder
	req := BuildPublicIPCreateRequest(name, userLabels)

	publicIP, err := client.Networking().PublicIPs().Create(ctx, req)
	if err != nil {
		return diag.Errorf("error creating public IP %s: %s", name, err)
	}

	d.SetId(publicIP.Metadata.Id)
	d.Set("project", resolveProject(d, config))

	// Wait for public IP to be ready and capture the ready resource
	timeout := d.Timeout(schema.TimeoutCreate)
	readyPIP, err := client.Networking().PublicIPs().WaitForReady(ctx, name, timeout)
	if err != nil {
		return diag.Errorf("error waiting for public IP %s to be ready: %s", name, err)
	}

	// Use the ready resource's identity
	d.SetId(readyPIP.Metadata.Id)

	return resourcePublicIPRead(ctx, d, meta)
}

func resourcePublicIPRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	publicIP, err := client.Networking().PublicIPs().Get(ctx, d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading public IP: %s", err)
	}

	diags = setDiag(d, "name", publicIP.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "region", derefString(publicIP.Metadata.Region), diags)
	diags = setDiag(d, "ip_id", publicIP.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", publicIP.Metadata.CreationTimestamp.Format(time.RFC3339), diags)

	// Set IP address (empty string if not yet allocated)
	ipAddress := ""
	if publicIP.Status.PublicIPv4Address != nil {
		ipAddress = *publicIP.Status.PublicIPv4Address
	}
	diags = setDiag(d, "ip_address", ipAddress, diags)

	if publicIP.Metadata.UserLabels != nil && len(*publicIP.Metadata.UserLabels) > 0 {
		diags = setDiag(d, "user_labels", flattenLabels(publicIP.Metadata.UserLabels), diags)
	}

	diags = setDiag(d, "system_labels", flattenLabels(publicIP.Metadata.SystemLabels), diags)
	diags = setDiag(d, "fqid", string(client.Networking().PublicIPRef(publicIP.Metadata.Id)), diags)

	return diags
}

func resourcePublicIPUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	if d.HasChange("user_labels") {
		publicIP, err := client.Networking().PublicIPs().Get(ctx, d.Id())
		if err != nil {
			return diag.Errorf("error reading public IP %s: %s", d.Id(), err)
		}

		if labels, ok := d.GetOk("user_labels"); ok {
			userLabels := make(networkingtypes.UserLabels)
			for k, v := range labels.(map[string]interface{}) {
				userLabels[k] = v.(string)
			}
			publicIP.Metadata.UserLabels = &userLabels
		} else {
			publicIP.Metadata.UserLabels = nil
		}

		_, err = client.Networking().PublicIPs().Patch(ctx, d.Id(), publicIP)
		if err != nil {
			return diag.Errorf("error updating public IP %s: %s", d.Id(), err)
		}
	}

	return resourcePublicIPRead(ctx, d, meta)
}

func resourcePublicIPDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	err := client.Networking().PublicIPs().Delete(ctx, d.Id())
	if err != nil {
		// If already deleted, that's okay
		if !isNotFoundError(err) {
			return diag.Errorf("error deleting public IP %s: %s", d.Id(), err)
		}
		// Already deleted, return early
		d.SetId("")
		return nil
	}

	// Wait for deletion to complete
	timeout := d.Timeout(schema.TimeoutDelete)
	err = client.Networking().PublicIPs().WaitForDeleted(ctx, d.Id(), timeout)
	if err != nil {
		return diag.Errorf("error waiting for public IP %s to be deleted: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}
