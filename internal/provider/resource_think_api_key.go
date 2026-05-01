// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceThinkAPIKey() *schema.Resource {
	return &schema.Resource{
		Description: "Manages an evroc Think API key for authenticating requests to shared and dedicated model endpoints.",

		CreateContext: resourceThinkAPIKeyCreate,
		ReadContext:   resourceThinkAPIKeyRead,
		DeleteContext: resourceThinkAPIKeyDelete,

		// API keys are immutable after creation — no UpdateContext needed.

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the API key. Must be unique within the project.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Project this resource belongs to. Defaults to the provider project.",
			},
			"expiry": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Expiry timestamp in RFC3339 format (e.g., 2026-12-31T23:59:59Z). If omitted, the key does not expire.",
			},
			// Computed fields
			"token": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "The full API key secret. The API only returns this value at creation time; Terraform persists it in state for subsequent reads.",
			},
			"token_prefix": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Short prefix of the API key for identification.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the API key was created (RFC3339 format).",
			},
		},
	}
}

func resourceThinkAPIKeyCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)

	var expiryStr string
	if v, ok := d.GetOk("expiry"); ok {
		expiryStr = v.(string)
	}

	req, err := BuildThinkAPIKeyCreateRequest(name, expiryStr)
	if err != nil {
		return diag.Errorf("error building API key request: %s", err)
	}

	apiKey, err := client.Think().ApiKeys().Create(ctx, req)
	if err != nil {
		return diag.Errorf("error creating Think API key %s: %s", name, err)
	}

	d.SetId(apiKey.Metadata.Id)
	d.Set("project", resolveProject(d, config))

	// Token is only returned on creation — capture it now
	if apiKey.Status.Token != nil {
		d.Set("token", *apiKey.Status.Token)
	}
	if apiKey.Status.TokenPrefix != nil {
		d.Set("token_prefix", *apiKey.Status.TokenPrefix)
	}
	d.Set("created_at", apiKey.Metadata.CreationTimestamp.Format(time.RFC3339))

	return nil
}

func resourceThinkAPIKeyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	// API keys don't have a Get endpoint — use List and filter
	apiKeys, err := client.Think().ApiKeys().List(ctx)
	if err != nil {
		return diag.Errorf("error listing Think API keys: %s", err)
	}

	if apiKeys.Items != nil {
		for _, key := range apiKeys.Items {
			if key.Metadata.Id == d.Id() {
				diags = setDiag(d, "name", key.Metadata.Id, diags)
				diags = setDiag(d, "project", resolveProject(d, config), diags)
				diags = setDiag(d, "created_at", key.Metadata.CreationTimestamp.Format(time.RFC3339), diags)
				if key.Status.TokenPrefix != nil {
					diags = setDiag(d, "token_prefix", *key.Status.TokenPrefix, diags)
				}
				if key.Spec.ExpiryTimestamp != nil {
					diags = setDiag(d, "expiry", key.Spec.ExpiryTimestamp.Format(time.RFC3339), diags)
				}
				return diags
			}
		}
	}

	// Not found
	d.SetId("")
	return nil
}

func resourceThinkAPIKeyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	err := client.Think().ApiKeys().Delete(ctx, d.Id())
	if err != nil {
		if !isNotFoundError(err) {
			return diag.Errorf("error deleting Think API key %s: %s", d.Id(), err)
		}
	}

	d.SetId("")
	return nil
}
