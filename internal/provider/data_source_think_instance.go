// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceThinkInstance() *schema.Resource {
	return &schema.Resource{
		Description: "Get information about an existing evroc Think instance.",

		ReadContext: dataSourceThinkInstanceRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the Think instance to look up.",
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
				Description: "Region where the instance is located.",
			},
			// Computed fields
			"instance_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier (UUID) of the instance.",
			},
			"model": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Model being served by this instance.",
			},
			"size": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Instance size for GPU allocation.",
			},
			"phase": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Current lifecycle phase (Creating, Running, Stopped, Failed, etc.).",
			},
			"endpoint": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "OpenAI-compatible API endpoint URL.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the instance was created (RFC3339 format).",
			},
		},
	}
}

func dataSourceThinkInstanceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	var diags diag.Diagnostics

	name := d.Get("name").(string)

	instance, err := client.Think().Instances().Get(ctx, name)
	if err != nil {
		return diag.Errorf("error reading Think instance %s: %s", name, err)
	}

	d.SetId(instance.Metadata.Id)
	diags = setDiag(d, "name", instance.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "region", derefString(instance.Metadata.Region), diags)
	diags = setDiag(d, "instance_id", instance.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", instance.Metadata.CreationTimestamp.Format(time.RFC3339), diags)
	diags = setDiag(d, "model", instance.Spec.Model, diags)

	if instance.Spec.Size != nil {
		diags = setDiag(d, "size", *instance.Spec.Size, diags)
	}

	if instance.Status.Phase != nil {
		diags = setDiag(d, "phase", string(*instance.Status.Phase), diags)
	}

	if instance.Status.Endpoint != nil {
		diags = setDiag(d, "endpoint", *instance.Status.Endpoint, diags)
	}

	return diags
}
