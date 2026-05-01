// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"strings"

	"github.com/evroc-oss/evroc-go-sdk/compute"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceComputeProfiles() *schema.Resource {
	return &schema.Resource{
		Description: "Lists all available VM compute profiles (sizes) in the evroc platform. Exposes both lists and individual named attributes for easy reference.",

		ReadContext: dataSourceComputeProfilesRead,

		Schema: map[string]*schema.Schema{
			"profiles": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of all available compute profile names.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"series": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Compute profile series with descriptions.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Series name (e.g., a1a, c1a, m1a).",
						},
						"description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Series description.",
						},
						"sizes": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Available sizes in this series.",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
			// General-purpose profiles (a1a series)
			"a1a_xs": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "General-purpose extra-small (a1a.xs) profile.",
			},
			"a1a_s": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "General-purpose small (a1a.s) profile.",
			},
			"a1a_m": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "General-purpose medium (a1a.m) profile.",
			},
			"a1a_l": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "General-purpose large (a1a.l) profile.",
			},
			"a1a_xl": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "General-purpose extra-large (a1a.xl) profile.",
			},
			"a1a_2xl": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "General-purpose 2x-large (a1a.2xl) profile.",
			},
			// Compute-optimized profiles (c1a series)
			"c1a_s": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Compute-optimized small (c1a.s) profile.",
			},
			"c1a_m": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Compute-optimized medium (c1a.m) profile.",
			},
			"c1a_l": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Compute-optimized large (c1a.l) profile.",
			},
			"c1a_xl": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Compute-optimized extra-large (c1a.xl) profile.",
			},
			"c1a_2xl": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Compute-optimized 2x-large (c1a.2xl) profile.",
			},
			// Memory-optimized profiles (m1a series)
			"m1a_s": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Memory-optimized small (m1a.s) profile.",
			},
			"m1a_m": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Memory-optimized medium (m1a.m) profile.",
			},
			"m1a_l": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Memory-optimized large (m1a.l) profile.",
			},
			"m1a_xl": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Memory-optimized extra-large (m1a.xl) profile.",
			},
			// GPU-enabled profiles: NVIDIA L40S (gn-l40s series)
			"gn_l40s_s": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "NVIDIA L40S GPU small (gn-l40s.s) profile.",
			},
			"gn_l40s_m": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "NVIDIA L40S GPU medium (gn-l40s.m) profile.",
			},
			"gn_l40s_l": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "NVIDIA L40S GPU large (gn-l40s.l) profile.",
			},
			// GPU-enabled profiles: NVIDIA B200 (gn-b200 series)
			"gn_b200_s": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "NVIDIA B200 GPU small (gn-b200.s) profile.",
			},
			"gn_b200_m": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "NVIDIA B200 GPU medium (gn-b200.m) profile.",
			},
			"gn_b200_l": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "NVIDIA B200 GPU large (gn-b200.l) profile.",
			},
			"gn_b200_xl": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "NVIDIA B200 GPU extra-large (gn-b200.xl) profile.",
			},
		},
	}
}

func dataSourceComputeProfilesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	// Get all valid VM sizes
	profiles := make([]string, len(compute.ValidVMSizes))
	profileMap := make(map[string]string)

	for i, size := range compute.ValidVMSizes {
		sizeStr := string(size)
		profiles[i] = sizeStr

		// Map profile names to schema field names (replace dots with underscores)
		// a1a.s -> a1a_s
		fieldName := strings.ReplaceAll(sizeStr, ".", "_")
		fieldName = strings.ReplaceAll(fieldName, "-", "_")
		profileMap[fieldName] = sizeStr
	}

	// Get series information
	series := make([]interface{}, len(compute.AllVMSizeSeries))
	for i, s := range compute.AllVMSizeSeries {
		sizes := make([]string, len(s.Sizes))
		for j, size := range s.Sizes {
			sizes[j] = string(size)
		}

		series[i] = map[string]interface{}{
			"name":        s.Name,
			"description": s.Description,
			"sizes":       sizes,
		}
	}

	d.SetId("compute-profiles")
	diags = setDiag(d, "profiles", profiles, diags)
	diags = setDiag(d, "series", series, diags)

	// Set individual named fields for all profiles defined in the schema.
	// Each key in profileMap must have a corresponding schema field.
	for fieldName, profileValue := range profileMap {
		diags = setDiag(d, fieldName, profileValue, diags)
	}

	return diags
}
