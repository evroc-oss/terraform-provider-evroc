// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"time"

	thinktypes "github.com/evroc-oss/evroc-go-sdk/types/think"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceThinkInstance() *schema.Resource {
	return &schema.Resource{
		Description: "Manages an evroc Think instance for dedicated model inference. " +
			"Large models may take 10+ minutes to become ready.",

		CreateContext: resourceThinkInstanceCreate,
		ReadContext:   resourceThinkInstanceRead,
		UpdateContext: resourceThinkInstanceUpdate,
		DeleteContext: resourceThinkInstanceDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Update: schema.DefaultTimeout(15 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the Think instance. Must be unique within the project.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Project this resource belongs to. Defaults to the provider project.",
			},
			"model": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Model to serve (e.g., meta-llama/Llama-3.3-70B-Instruct). Must be the ID of an available Model.",
			},
			"size": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Instance size for GPU allocation. Must be the ID of an available Size.",
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Region where the instance is created. Defaults to provider region.",
			},
			"running": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether the instance should be running. Set to false to stop the instance and release GPU resources.",
			},
			"user_labels": {
				Type:        schema.TypeMap,
				Optional:    true,
				Deprecated:  "Think does not currently support labels. Setting this field has no effect. This will be enabled in a future release.",
				Description: "User-defined labels (key/value pairs) for organizing and selecting resources. Not currently supported by Think — any values set will be ignored.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			// Computed fields
			"instance_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier (UUID) of the instance.",
			},
			"system_labels": {
				Type:        schema.TypeMap,
				Computed:    true,
				Deprecated:  "Think does not currently support labels. This field will always be empty. This will be enabled in a future release.",
				Description: "System-managed labels automatically set by evroc (read-only). Not currently supported by Think.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the instance was created (RFC3339 format).",
			},
			"endpoint": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "OpenAI-compatible API endpoint URL for the running instance.",
			},
			"phase": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Current lifecycle phase of the instance (Creating, Running, Stopped, Failed, etc.).",
			},
		},
	}
}

func resourceThinkInstanceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)
	running := d.Get("running").(bool)

	if _, ok := d.GetOk("user_labels"); ok {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Think does not currently support labels",
			Detail:   "The user_labels field is accepted but has no effect. Labels will be supported in a future release.",
		})
	}

	req := BuildThinkInstanceCreateRequest(name, d.Get("model").(string), d.Get("size").(string), !running)

	instance, err := client.Think().Instances().Create(ctx, req)
	if err != nil {
		return diag.Errorf("error creating Think instance %s: %s", name, err)
	}

	d.SetId(instance.Metadata.Id)
	d.Set("project", resolveProject(d, config))

	if running {
		timeout := d.Timeout(schema.TimeoutCreate)
		readyInstance, err := client.Think().Instances().WaitForReady(ctx, name, timeout)
		if err != nil {
			return diag.Errorf("error waiting for Think instance %s to be ready: %s", name, err)
		}
		d.SetId(readyInstance.Metadata.Id)
	}

	readDiags := resourceThinkInstanceRead(ctx, d, meta)
	return append(diags, readDiags...)
}

func resourceThinkInstanceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	instance, err := client.Think().Instances().Get(ctx, d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading Think instance: %s", err)
	}

	diags = setDiag(d, "name", instance.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "region", derefString(instance.Metadata.Region), diags)
	diags = setDiag(d, "instance_id", instance.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", instance.Metadata.CreationTimestamp.Format(time.RFC3339), diags)
	diags = setDiag(d, "model", instance.Spec.Model, diags)

	if instance.Spec.Size != nil {
		diags = setDiag(d, "size", *instance.Spec.Size, diags)
	}

	// Derive running from phase
	if instance.Status.Phase != nil {
		phase := string(*instance.Status.Phase)
		diags = setDiag(d, "phase", phase, diags)
		diags = setDiag(d, "running", *instance.Status.Phase == thinktypes.Running, diags)
	}

	if instance.Status.Endpoint != nil {
		diags = setDiag(d, "endpoint", *instance.Status.Endpoint, diags)
	} else {
		diags = setDiag(d, "endpoint", "", diags)
	}

	if _, ok := d.GetOk("user_labels"); ok {
		diags = setDiag(d, "user_labels", flattenLabels(instance.Metadata.UserLabels), diags)
	}
	diags = setDiag(d, "system_labels", flattenLabels(instance.Metadata.SystemLabels), diags)

	return diags
}

func resourceThinkInstanceUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	instanceService := client.Think().Instances()

	if d.HasChange("user_labels") {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Think does not currently support labels",
			Detail:   "The user_labels field is accepted but has no effect. Labels will be supported in a future release.",
		})
	}

	if d.HasChange("running") {
		running := d.Get("running").(bool)
		timeout := d.Timeout(schema.TimeoutUpdate)

		if running {
			if err := instanceService.Start(ctx, d.Id()); err != nil {
				return diag.Errorf("error starting Think instance %s: %s", d.Id(), err)
			}
			if _, err := instanceService.WaitForReady(ctx, d.Id(), timeout); err != nil {
				return diag.Errorf("error waiting for Think instance %s to be ready: %s", d.Id(), err)
			}
		} else {
			if err := instanceService.Stop(ctx, d.Id()); err != nil {
				return diag.Errorf("error stopping Think instance %s: %s", d.Id(), err)
			}
			if _, err := instanceService.WaitForStopped(ctx, d.Id(), timeout); err != nil {
				return diag.Errorf("error waiting for Think instance %s to stop: %s", d.Id(), err)
			}
		}
	}

	readDiags := resourceThinkInstanceRead(ctx, d, meta)
	return append(diags, readDiags...)
}

func resourceThinkInstanceDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	err := client.Think().Instances().Delete(ctx, d.Id())
	if err != nil {
		if !isNotFoundError(err) {
			return diag.Errorf("error deleting Think instance %s: %s", d.Id(), err)
		}
		d.SetId("")
		return nil
	}

	timeout := d.Timeout(schema.TimeoutDelete)
	if err := client.Think().Instances().WaitForDeleted(ctx, d.Id(), timeout); err != nil {
		return diag.Errorf("error waiting for Think instance %s to be deleted: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}
