// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"time"

	"github.com/evroc-oss/evroc-go-sdk/compute"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceCustomDiskImage() *schema.Resource {
	return &schema.Resource{
		Description: "Registers an image object stored in an evroc bucket as a custom disk image. " +
			"Use `fqid` as the `image` of an `evroc_disk`. The object itself is uploaded outside Terraform. " +
			"Source, architecture and default disk size are immutable; changing them replaces the registration.",

		CreateContext: resourceCustomDiskImageCreate,
		ReadContext:   resourceCustomDiskImageRead,
		UpdateContext: resourceCustomDiskImageUpdate,
		DeleteContext: resourceCustomDiskImageDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the custom disk image. Must be unique within the project and region.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Project this resource belongs to. Defaults to the provider project.",
			},
			"region": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Region of the image, selected by the provider configuration.",
			},
			"bucket": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				DiffSuppressFunc: suppressFQIDDiff,
				Description:      "Bucket holding the image object. Accepts the bucket name or its FQID (e.g., evroc_bucket.images.fqid). Must be in the same project and region.",
			},
			"bucket_service_account": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				DiffSuppressFunc: suppressFQIDDiff,
				Description:      "Bucket service account authorized to read the object. Accepts the name or FQID (e.g., evroc_bucket_service_account.reader.fqid).",
			},
			"object_path": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotEmpty),
				Description:      "Path of the image object within the bucket (e.g., images/node-v1.qcow2).",
			},
			"object_version": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Version ID of the object to register, from a versioned bucket. Without it, new disks use whatever object is at object_path at the time they are created.",
			},
			"default_disk_size": {
				Type:             schema.TypeInt,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(1)),
				Description:      "Default disk size in GB for disks created from this image. Must be at least the image's virtual disk size.",
			},
			"architecture": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "amd64",
				ForceNew:    true,
				Description: "CPU architecture of the image.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Free-form description of the image.",
			},
			"os_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Operating system name (e.g., ubuntu).",
			},
			"os_version": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Operating system version (e.g., 24.04).",
			},
			"image_version": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Descriptive version of the image build. Does not pin the object; use object_version for that.",
			},
			"user_labels": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "User-defined labels (key/value pairs) for organizing and selecting resources.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			// Computed fields
			"image_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier (UUID) of the custom disk image.",
			},
			"ready": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the registration is ready for use. Readiness does not verify that the object is a bootable image.",
			},
			"system_labels": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "System-managed labels automatically set by evroc (read-only).",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the image was registered (RFC3339 format).",
			},
			"fqid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified resource ID (FQID). Pass this to evroc_disk.image.",
			},
		},
	}
}

func resourceCustomDiskImageCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	var userLabels map[string]string
	if labels, ok := d.GetOk("user_labels"); ok {
		userLabels = make(map[string]string)
		for k, v := range labels.(map[string]interface{}) {
			userLabels[k] = v.(string)
		}
	}

	name := d.Get("name").(string)
	req := BuildCustomDiskImageCreateRequest(client, name,
		d.Get("bucket").(string),
		d.Get("bucket_service_account").(string),
		d.Get("object_path").(string),
		d.Get("object_version").(string),
		d.Get("architecture").(string),
		d.Get("default_disk_size").(int),
		d.Get("description").(string),
		d.Get("os_name").(string),
		d.Get("os_version").(string),
		d.Get("image_version").(string),
		userLabels)

	image, err := client.Compute().CustomDiskImages().Create(ctx, req)
	if err != nil {
		return diag.Errorf("error creating custom disk image %s: %s", name, err)
	}

	d.SetId(image.Metadata.Id)
	d.Set("project", resolveProject(d, config))

	timeout := d.Timeout(schema.TimeoutCreate)
	if _, err := client.Compute().CustomDiskImages().WaitForReady(ctx, name, timeout); err != nil {
		return diag.Errorf("error waiting for custom disk image %s to be ready: %s", name, err)
	}

	return resourceCustomDiskImageRead(ctx, d, meta)
}

func resourceCustomDiskImageRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	image, err := client.Compute().CustomDiskImages().Get(ctx, d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading custom disk image: %s", err)
	}

	diags = setDiag(d, "name", image.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "region", derefString(image.Metadata.Region), diags)
	diags = setDiag(d, "image_id", image.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", image.Metadata.CreationTimestamp.Format(time.RFC3339), diags)

	diags = setDiag(d, "bucket", image.Spec.Source.BucketRef, diags)
	diags = setDiag(d, "bucket_service_account", image.Spec.Source.ServiceAccountRef, diags)
	diags = setDiag(d, "object_path", image.Spec.Source.Path, diags)
	diags = setDiag(d, "object_version", derefString(image.Spec.Source.Version), diags)
	diags = setDiag(d, "default_disk_size", customDiskImageSizeGB(image.Spec.DefaultDiskSize), diags)
	diags = setDiag(d, "architecture", string(image.Spec.Architecture), diags)
	diags = setDiag(d, "description", derefString(image.Spec.Description), diags)
	diags = setDiag(d, "os_name", derefString(image.Spec.OsName), diags)
	diags = setDiag(d, "os_version", derefString(image.Spec.OsVersion), diags)
	diags = setDiag(d, "image_version", derefString(image.Spec.ImageVersion), diags)
	diags = setDiag(d, "ready", compute.IsCustomDiskImageReady(image), diags)

	if image.Metadata.UserLabels != nil && len(*image.Metadata.UserLabels) > 0 {
		diags = setDiag(d, "user_labels", flattenLabels(image.Metadata.UserLabels), diags)
	}
	diags = setDiag(d, "system_labels", flattenLabels(image.Metadata.SystemLabels), diags)
	diags = setDiag(d, "fqid", client.Compute().CustomDiskImageRef(image.Metadata.Id).String(), diags)

	return diags
}

// resourceCustomDiskImageUpdate patches only the mutable descriptive fields and
// labels. The source, architecture and default size are immutable and never
// resent.
func resourceCustomDiskImageUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	spec := map[string]interface{}{}
	for key, field := range map[string]string{
		"description":   "description",
		"os_name":       "osName",
		"os_version":    "osVersion",
		"image_version": "imageVersion",
	} {
		if d.HasChange(key) {
			if v := d.Get(key).(string); v != "" {
				spec[field] = v
			} else {
				spec[field] = nil
			}
		}
	}
	patch := map[string]interface{}{}
	if len(spec) > 0 {
		patch["spec"] = spec
	}
	if d.HasChange("user_labels") {
		var userLabels interface{}
		if labels, ok := d.GetOk("user_labels"); ok {
			userLabels = labels
		}
		patch["metadata"] = map[string]interface{}{"userLabels": userLabels}
	}

	if len(patch) > 0 {
		if _, err := client.Compute().CustomDiskImages().Patch(ctx, d.Id(), patch); err != nil {
			return diag.Errorf("error updating custom disk image %s: %s", d.Id(), err)
		}
	}

	return resourceCustomDiskImageRead(ctx, d, meta)
}

func resourceCustomDiskImageDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	if err := client.Compute().CustomDiskImages().Delete(ctx, d.Id()); err != nil {
		if !isNotFoundError(err) {
			return diag.Errorf("error deleting custom disk image: %s", err)
		}
		d.SetId("")
		return nil
	}

	timeout := d.Timeout(schema.TimeoutDelete)
	if err := client.Compute().CustomDiskImages().WaitForDeleted(ctx, d.Id(), timeout); err != nil {
		return diag.Errorf("error waiting for custom disk image %s to be deleted: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}
