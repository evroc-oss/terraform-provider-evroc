// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"time"
)

func dataSourceThinkSharedModels() *schema.Resource {
	return &schema.Resource{
		Description: "Lists Think shared models, their invocation aliases, capabilities and planned sunset dates. Use an alias as the model name in inference requests. A sunset value is an RFC3339 timestamp when the model will cease to be served, or an empty string when no sunset is announced.",
		Schema: map[string]*schema.Schema{
			"project": {Type: schema.TypeString, Optional: true, Computed: true, Description: "Project to query. Defaults to the provider project."},
			"models": {Type: schema.TypeList, Computed: true, Description: "Available shared models.", Elem: &schema.Resource{Schema: map[string]*schema.Schema{
				"name": {Type: schema.TypeString, Computed: true}, "description": {Type: schema.TypeString, Computed: true},
				"license": {Type: schema.TypeString, Computed: true}, "license_url": {Type: schema.TypeString, Computed: true},
				"sunset":       {Type: schema.TypeString, Computed: true, Description: "RFC3339 time when the model will cease to be served, or empty if unset."},
				"aliases":      {Type: schema.TypeList, Computed: true, Elem: &schema.Schema{Type: schema.TypeString}},
				"capabilities": {Type: schema.TypeList, Computed: true, Elem: &schema.Schema{Type: schema.TypeString}},
			}}},
		},
		ReadContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
			client, diags := resolveClient(d, meta.(*ProviderConfig))
			if diags.HasError() {
				return diags
			}
			result, err := client.Think().SharedModels().List(ctx)
			if err != nil {
				return diag.FromErr(err)
			}
			models := make([]interface{}, 0)
			if result.Items != nil {
				for _, m := range result.Items {
					aliases := []string{}
					if m.Spec.Aliases != nil {
						aliases = *m.Spec.Aliases
					}
					capabilities := []string{}
					if m.Spec.Capabilities != nil {
						for _, c := range *m.Spec.Capabilities {
							capabilities = append(capabilities, string(c))
						}
					}
					sunset := ""
					if m.Spec.Sunset != nil {
						sunset = m.Spec.Sunset.Format(time.RFC3339)
					}
					models = append(models, map[string]interface{}{"name": m.Metadata.Id, "description": derefString(m.Spec.Description), "license": derefString(m.Spec.License), "license_url": derefString(m.Spec.LicenseUrl), "sunset": sunset, "aliases": aliases, "capabilities": capabilities})
				}
			}
			d.SetId("think-shared-models")
			diags = setDiag(d, "project", client.DefaultProject(), diags)
			return setDiag(d, "models", models, diags)
		},
	}
}
