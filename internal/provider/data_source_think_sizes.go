// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceThinkSizes() *schema.Resource {
	return &schema.Resource{
		Description: "Lists all available GPU instance sizes for evroc Think.",

		ReadContext: dataSourceThinkSizesRead,

		Schema: map[string]*schema.Schema{
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Project to query. Defaults to the provider project.",
			},
			"sizes": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of available sizes.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Size identifier.",
						},
						"description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Size description.",
						},
					},
				},
			},
		},
	}
}

func dataSourceThinkSizesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	var diags diag.Diagnostics

	sizeList, err := client.Think().Sizes().List(ctx)
	if err != nil {
		return diag.Errorf("error listing Think sizes: %s", err)
	}

	var sizes []interface{}
	if sizeList.Items != nil {
		for _, s := range sizeList.Items {
			sizes = append(sizes, map[string]interface{}{
				"name":        s.Metadata.Id,
				"description": derefString(s.Spec.Description),
			})
		}
	}

	d.SetId("think-sizes")
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "sizes", sizes, diags)

	return diags
}
