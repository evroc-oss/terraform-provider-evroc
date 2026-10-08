// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"

	"github.com/evroc-oss/evroc-go-sdk/compute"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceCustomDiskImage() *schema.Resource {
	return &schema.Resource{
		Description: "Looks up a registered custom disk image by name in the provider region. Use fqid as the image of an evroc_disk.",
		ReadContext: dataSourceCustomDiskImageRead,
		Schema: map[string]*schema.Schema{
			"name":                   {Type: schema.TypeString, Required: true, ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotEmpty), Description: "Name of the registered custom disk image."},
			"project":                {Type: schema.TypeString, Optional: true, Computed: true, Description: "Project containing the image. Defaults to the provider project."},
			"region":                 {Type: schema.TypeString, Computed: true, Description: "Region containing the image, selected by the provider configuration."},
			"fqid":                   {Type: schema.TypeString, Computed: true, Description: "Fully qualified image reference. Pass this to evroc_disk.image."},
			"architecture":           {Type: schema.TypeString, Computed: true, Description: "CPU architecture of the image."},
			"bucket":                 {Type: schema.TypeString, Computed: true, Description: "FQID of the bucket holding the image object."},
			"bucket_service_account": {Type: schema.TypeString, Computed: true, Description: "FQID of the bucket service account that reads the object."},
			"object_path":            {Type: schema.TypeString, Computed: true, Description: "Path of the image object within the bucket."},
			"object_version":         {Type: schema.TypeString, Computed: true, Description: "Object version the registration is pinned to. Empty when unpinned, in which case new disks use whatever is at object_path."},
			"default_disk_size":      {Type: schema.TypeInt, Computed: true, Description: "Default disk size in GB for disks created from this image."},
			"ready":                  {Type: schema.TypeBool, Computed: true, Description: "Whether the registration is ready for use."},
			"description":            {Type: schema.TypeString, Computed: true, Description: "Description of the image."},
			"os_name":                {Type: schema.TypeString, Computed: true, Description: "Operating system name."},
			"os_version":             {Type: schema.TypeString, Computed: true, Description: "Operating system version."},
			"image_version":          {Type: schema.TypeString, Computed: true, Description: "Descriptive version of the image build."},
		},
	}
}

func dataSourceCustomDiskImageRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, diags := resolveClient(d, meta.(*ProviderConfig))
	if diags.HasError() {
		return diags
	}
	name := d.Get("name").(string)
	image, err := client.Compute().CustomDiskImages().Get(ctx, name)
	if err != nil {
		return diag.Errorf("error reading custom disk image %s: %s", name, err)
	}
	fqid := client.Compute().CustomDiskImageRef(image.Metadata.Id).String()
	d.SetId(fqid)
	diags = setDiag(d, "name", image.Metadata.Id, diags)
	diags = setDiag(d, "project", client.DefaultProject(), diags)
	diags = setDiag(d, "region", client.DefaultRegion(), diags)
	diags = setDiag(d, "fqid", fqid, diags)
	diags = setDiag(d, "architecture", string(image.Spec.Architecture), diags)
	diags = setDiag(d, "bucket", image.Spec.Source.BucketRef, diags)
	diags = setDiag(d, "bucket_service_account", image.Spec.Source.ServiceAccountRef, diags)
	diags = setDiag(d, "object_path", image.Spec.Source.Path, diags)
	diags = setDiag(d, "object_version", derefString(image.Spec.Source.Version), diags)
	diags = setDiag(d, "default_disk_size", customDiskImageSizeGB(image.Spec.DefaultDiskSize), diags)
	diags = setDiag(d, "ready", compute.IsCustomDiskImageReady(image), diags)
	diags = setDiag(d, "description", derefString(image.Spec.Description), diags)
	diags = setDiag(d, "os_name", derefString(image.Spec.OsName), diags)
	diags = setDiag(d, "os_version", derefString(image.Spec.OsVersion), diags)
	diags = setDiag(d, "image_version", derefString(image.Spec.ImageVersion), diags)
	return diags
}
