// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"sort"
	"strings"

	"github.com/evroc-oss/evroc-go-sdk/compute"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceComputeProfiles() *schema.Resource {
	return &schema.Resource{
		Description: "Lists the VM compute profiles (sizes) currently available in the evroc platform, queried live from the API. " +
			"Exposes both lists and individual named attributes for easy reference; a named attribute is empty when that profile is not offered.",

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
			"details": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Details for each available compute profile, in the same order as the profiles list.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Profile name (e.g., a1a.s).",
						},
						"processor_architecture": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Processor architecture (amd64 or arm64).",
						},
						"vcpus": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Number of vCPUs.",
						},
						"memory_amount": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Amount of memory, in the unit given by memory_unit.",
						},
						"memory_unit": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Memory unit (KB, MB, or GB).",
						},
						"gpu_model": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "GPU model. Empty for profiles without GPUs.",
						},
						"gpu_quantity": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Number of GPUs. Zero for profiles without GPUs.",
						},
						"gpu_local_disk_gb": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Size in GB of the local disk created automatically for GPU profiles. Zero for profiles without GPUs.",
						},
					},
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
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	list, err := client.Compute().ComputeProfiles().List(ctx)
	if err != nil {
		return diag.Errorf("error listing compute profiles: %s", err)
	}

	items := list.Items
	sort.Slice(items, func(i, j int) bool { return items[i].Metadata.Id < items[j].Metadata.Id })

	profiles := make([]string, 0, len(items))
	details := make([]interface{}, 0, len(items))
	for _, p := range items {
		profiles = append(profiles, p.Metadata.Id)

		detail := map[string]interface{}{
			"name":                   p.Metadata.Id,
			"processor_architecture": p.Spec.ProcessorArchitecture,
			"vcpus":                  p.Spec.VCPUs,
			"memory_amount":          int(p.Spec.Memory.Amount),
			"memory_unit":            string(p.Spec.Memory.Unit),
			"gpu_model":              "",
			"gpu_quantity":           0,
			"gpu_local_disk_gb":      0,
		}
		if p.Spec.Gpus != nil {
			detail["gpu_model"] = p.Spec.Gpus.Model
			detail["gpu_quantity"] = int(p.Spec.Gpus.Quantity)
			detail["gpu_local_disk_gb"] = int(p.Spec.Gpus.LocalDisk)
		}
		details = append(details, detail)
	}

	// Group profiles into series by the prefix before the first dot (a1a.s -> a1a).
	// Descriptions for known series come from the SDK catalog.
	seriesDescriptions := make(map[string]string, len(compute.AllVMSizeSeries))
	for _, s := range compute.AllVMSizeSeries {
		seriesDescriptions[s.Name] = s.Description
	}

	seriesSizes := make(map[string][]string)
	var seriesNames []string
	for _, p := range profiles {
		name := p
		if i := strings.Index(p, "."); i > 0 {
			name = p[:i]
		}
		if _, ok := seriesSizes[name]; !ok {
			seriesNames = append(seriesNames, name)
		}
		seriesSizes[name] = append(seriesSizes[name], p)
	}

	series := make([]interface{}, 0, len(seriesNames))
	for _, name := range seriesNames {
		series = append(series, map[string]interface{}{
			"name":        name,
			"description": seriesDescriptions[name],
			"sizes":       seriesSizes[name],
		})
	}

	d.SetId("compute-profiles")
	diags = setDiag(d, "profiles", profiles, diags)
	diags = setDiag(d, "details", details, diags)
	diags = setDiag(d, "series", series, diags)

	// Set the named convenience field for each profile that has one declared in
	// the schema (a1a.s -> a1a_s). Profiles without a declared field are still
	// available through the profiles list.
	schemaFields := dataSourceComputeProfiles().Schema
	for _, p := range profiles {
		fieldName := strings.ReplaceAll(p, ".", "_")
		fieldName = strings.ReplaceAll(fieldName, "-", "_")
		if _, ok := schemaFields[fieldName]; ok {
			diags = setDiag(d, fieldName, p, diags)
		}
	}

	return diags
}
