// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceThinkModels() *schema.Resource {
	return &schema.Resource{
		Description: "Lists all available models for evroc Think dedicated instances.",

		ReadContext: dataSourceThinkModelsRead,

		Schema: map[string]*schema.Schema{
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Project to query. Defaults to the provider project.",
			},
			"models": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of available models.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Model identifier.",
						},
						"description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Model description.",
						},
						"handle": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "HuggingFace model handle.",
						},
						"license": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Model license.",
						},
						"default_size": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Suggested default instance size for this model.",
						},
					},
				},
			},
		},
	}
}

func dataSourceThinkModelsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	var diags diag.Diagnostics

	modelList, err := client.Think().Models().List(ctx)
	if err != nil {
		return diag.Errorf("error listing Think models: %s", err)
	}

	var models []interface{}
	if modelList.Items != nil {
		for _, m := range modelList.Items {
			model := map[string]interface{}{
				"name":         m.Metadata.Id,
				"description":  derefString(m.Spec.Description),
				"handle":       derefString(m.Spec.Handle),
				"license":      derefString(m.Spec.License),
				"default_size": "",
			}
			if m.Spec.DefaultSize != nil {
				if s, ok := m.Spec.DefaultSize.(string); ok {
					model["default_size"] = s
				}
			}
			models = append(models, model)
		}
	}

	d.SetId("think-models")
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "models", models, diags)

	return diags
}
