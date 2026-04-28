// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"time"

	computetypes "github.com/evroc-oss/evroc-go-sdk/types/compute"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceHotswapDiskAttachment() *schema.Resource {
	return &schema.Resource{
		Description: "Manages a hot-swap disk attachment to an evroc virtual machine. Allows attaching and detaching disks without VM restart.",

		CreateContext: resourceHotswapDiskAttachmentCreate,
		ReadContext:   resourceHotswapDiskAttachmentRead,
		UpdateContext: resourceHotswapDiskAttachmentUpdate,
		DeleteContext: resourceHotswapDiskAttachmentDelete,

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
				Description: "Name of the disk attachment.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Project this resource belongs to. Defaults to the provider project.",
			},
			"virtual_machine": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				DiffSuppressFunc: suppressFQIDDiff,
				Description:      "Name of the virtual machine to attach the disk to.",
			},
			"disk": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				DiffSuppressFunc: suppressFQIDDiff,
				Description:      "Name of the disk to attach.",
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Region where the disk attachment will be created.",
			},
			"user_labels": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "User-defined labels (key/value pairs) for organizing and selecting resources.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"attachment_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier of the disk attachment.",
			},
			"system_labels": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "System-managed labels automatically set by evroc (read-only).",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"serial": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Serial identifier of the attached disk.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the disk attachment was created.",
			},
		},
	}
}

func resourceHotswapDiskAttachmentCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)
	vmName := d.Get("virtual_machine").(string)
	diskName := d.Get("disk").(string)

	// Get user labels
	var userLabels map[string]string
	if labels, ok := d.GetOk("user_labels"); ok {
		userLabels = make(map[string]string)
		for k, v := range labels.(map[string]interface{}) {
			userLabels[k] = v.(string)
		}
	}

	req := BuildDiskAttachmentCreateRequest(client, name, vmName, diskName, userLabels)

	attachment, err := client.Compute().HotswapDiskAttachments().Create(ctx, req)
	if err != nil {
		return diag.Errorf("error creating disk attachment: %s", err)
	}

	d.SetId(attachment.Metadata.Id)
	d.Set("project", resolveProject(d, config))

	// Wait for attachment to be ready and capture the ready resource
	timeout := d.Timeout(schema.TimeoutCreate)
	readyAttachment, err := client.Compute().HotswapDiskAttachments().WaitForReady(ctx, name, timeout)
	if err != nil {
		return diag.Errorf("error waiting for disk attachment %s to be ready: %s", name, err)
	}

	// Use the ready resource's identity
	d.SetId(readyAttachment.Metadata.Id)

	return resourceHotswapDiskAttachmentRead(ctx, d, meta)
}

func resourceHotswapDiskAttachmentRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	attachment, err := client.Compute().HotswapDiskAttachments().Get(ctx, d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading disk attachment: %s", err)
	}

	diags = setDiag(d, "name", attachment.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "region", derefString(attachment.Metadata.Region), diags)
	diags = setDiag(d, "attachment_id", attachment.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", attachment.Metadata.CreationTimestamp.Format(time.RFC3339), diags)

	diags = setDiag(d, "virtual_machine", attachment.Spec.VirtualMachineRef, diags)
	diags = setDiag(d, "disk", attachment.Spec.DiskRef, diags)

	if attachment.Status.Serial != nil {
		diags = setDiag(d, "serial", *attachment.Status.Serial, diags)
	}

	if attachment.Metadata.UserLabels != nil && len(*attachment.Metadata.UserLabels) > 0 {
		diags = setDiag(d, "user_labels", flattenLabels(attachment.Metadata.UserLabels), diags)
	}

	diags = setDiag(d, "system_labels", flattenLabels(attachment.Metadata.SystemLabels), diags)

	return diags
}

func resourceHotswapDiskAttachmentUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	if d.HasChange("user_labels") {
		attachment, err := client.Compute().HotswapDiskAttachments().Get(ctx, d.Id())
		if err != nil {
			return diag.Errorf("error reading disk attachment %s: %s", d.Id(), err)
		}

		if labels, ok := d.GetOk("user_labels"); ok {
			userLabels := make(computetypes.UserLabels)
			for k, v := range labels.(map[string]interface{}) {
				userLabels[k] = v.(string)
			}
			attachment.Metadata.UserLabels = &userLabels
		} else {
			attachment.Metadata.UserLabels = nil
		}

		_, err = client.Compute().HotswapDiskAttachments().Patch(ctx, d.Id(), attachment)
		if err != nil {
			return diag.Errorf("error updating disk attachment %s: %s", d.Id(), err)
		}
	}

	return resourceHotswapDiskAttachmentRead(ctx, d, meta)
}

func resourceHotswapDiskAttachmentDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	err := client.Compute().HotswapDiskAttachments().Delete(ctx, d.Id())
	if err != nil {
		// If already deleted, that's okay
		if !isNotFoundError(err) {
			return diag.Errorf("error deleting disk attachment: %s", err)
		}
		d.SetId("")
		return nil
	}

	// Wait for deletion to complete
	timeout := d.Timeout(schema.TimeoutDelete)
	if err := client.Compute().HotswapDiskAttachments().WaitForDeleted(ctx, d.Id(), timeout); err != nil {
		return diag.Errorf("error waiting for disk attachment %s to be deleted: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}
