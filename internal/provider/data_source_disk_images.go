// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDiskImages() *schema.Resource {
	return &schema.Resource{
		Description: "Lists the disk images currently available in the evroc platform, queried live from the API. " +
			"Exposes both a list and individual named attributes for easy reference; a named attribute is empty when that image is not offered.",

		ReadContext: dataSourceDiskImagesRead,

		Schema: map[string]*schema.Schema{
			"images": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of all available disk image names.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"details": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Details for each available disk image, in the same order as the images list.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Image name (e.g., ubuntu-minimal.24-04.1).",
						},
						"os_image": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Operating system image family.",
						},
						"os_version": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Operating system version.",
						},
						"os_arch": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Processor architecture the image is built for (amd64 or arm64).",
						},
						"version": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Image version number.",
						},
						"default_size_amount": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Default disk size amount, in the unit given by default_size_unit, used when a disk does not specify a size.",
						},
						"default_size_unit": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Default disk size unit.",
						},
						"gpu_affinities": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "GPU models this image is intended for. All images can be used for CPU VMs.",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
			// Individual image names as computed fields for easy reference
			"ubuntu_minimal_24_04_1": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Ubuntu 24.04.1 Minimal image.",
			},
			"ubuntu_24_04_1": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Ubuntu 24.04.1 image.",
			},
			"ubuntu_22_04_1": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Ubuntu 22.04.1 image.",
			},
			"rocky_10_0_1": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Rocky Linux 10.0.1 image.",
			},
			"rocky_9_6_1": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Rocky Linux 9.6.1 image.",
			},
			"rocky_9_5_1": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Rocky Linux 9.5.1 image.",
			},
			"rocky_8_10_1": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Rocky Linux 8.10.1 image.",
			},
			"opensuse_15_6_1": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "OpenSUSE 15.6.1 image.",
			},
			"opensuse_15_5_1": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "OpenSUSE 15.5.1 image.",
			},
			"sles_15_6_1": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "SUSE Linux Enterprise Server 15.6.1 image.",
			},
			"sles_15_5_1": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "SUSE Linux Enterprise Server 15.5.1 image.",
			},
			"sl_micro_6_1_1": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "SUSE Linux Micro 6.1.1 image.",
			},
		},
	}
}

func dataSourceDiskImagesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	list, err := client.Compute().DiskImages().List(ctx)
	if err != nil {
		return diag.Errorf("error listing disk images: %s", err)
	}

	items := list.Items
	sort.Slice(items, func(i, j int) bool { return items[i].Metadata.Id < items[j].Metadata.Id })

	images := make([]string, 0, len(items))
	details := make([]interface{}, 0, len(items))
	for _, img := range items {
		images = append(images, img.Metadata.Id)

		gpuAffinities := []string{}
		if img.Spec.GpuAffinities != nil {
			gpuAffinities = *img.Spec.GpuAffinities
		}
		osArch := ""
		if img.Spec.OsArch != nil {
			osArch = string(*img.Spec.OsArch)
		}
		details = append(details, map[string]interface{}{
			"name":                img.Metadata.Id,
			"os_image":            img.Spec.OsImage,
			"os_version":          img.Spec.OsVersion,
			"os_arch":             osArch,
			"version":             int(img.Spec.Version),
			"default_size_amount": int(img.Spec.DefaultSize.Amount),
			"default_size_unit":   string(img.Spec.DefaultSize.Unit),
			"gpu_affinities":      gpuAffinities,
		})
	}

	d.SetId("disk-images")
	diags = setDiag(d, "images", images, diags)
	diags = setDiag(d, "details", details, diags)

	// Set the named convenience field for each image that has one declared in
	// the schema (ubuntu-minimal.24-04.1 -> ubuntu_minimal_24_04_1). Images
	// without a declared field are still available through the images list.
	schemaFields := dataSourceDiskImages().Schema
	for _, img := range images {
		fieldName := strings.ReplaceAll(img, ".", "_")
		fieldName = strings.ReplaceAll(fieldName, "-", "_")
		if _, ok := schemaFields[fieldName]; ok {
			diags = setDiag(d, fieldName, img, diags)
		}
	}

	return diags
}
