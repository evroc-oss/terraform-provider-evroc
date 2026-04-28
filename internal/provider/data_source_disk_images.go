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

func dataSourceDiskImages() *schema.Resource {
	return &schema.Resource{
		Description: "Lists all available disk images in the evroc platform. Exposes both a list and individual named attributes for easy reference.",

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
	var diags diag.Diagnostics
	// Convert ValidDiskImages to strings
	images := make([]string, len(compute.ValidDiskImages))
	imageMap := make(map[string]string)

	for i, img := range compute.ValidDiskImages {
		imgStr := string(img)
		images[i] = imgStr

		// Map image names to schema field names (replace dots and dashes with underscores)
		// ubuntu-minimal.24-04.1 -> ubuntu_minimal_24_04_1
		fieldName := imgStr
		fieldName = strings.ReplaceAll(fieldName, ".", "_")
		fieldName = strings.ReplaceAll(fieldName, "-", "_")
		imageMap[fieldName] = imgStr
	}

	d.SetId("disk-images")
	diags = setDiag(d, "images", images, diags)

	// Set individual named fields
	for fieldName, imgValue := range imageMap {
		diags = setDiag(d, fieldName, imgValue, diags)
	}

	return diags
}
