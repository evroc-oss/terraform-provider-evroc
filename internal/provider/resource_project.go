// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"time"

	iamtypes "github.com/evroc-oss/evroc-go-sdk/types/iam"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceProject() *schema.Resource {
	return &schema.Resource{
		Description: "Manages an evroc project for organizing cloud resources within an organization.",

		CreateContext: resourceProjectCreate,
		ReadContext:   resourceProjectRead,
		UpdateContext: resourceProjectUpdate,
		DeleteContext: resourceProjectDelete,

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
				Description: "Unique identifier for the project. Immutable after creation. Shown as 'id' in the console and CLI.",
			},
			"display_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Human-friendly display name for the project. Mutable. Shown as 'name' in the console and CLI.",
			},
			"organization": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Organization ID this project belongs to. Defaults to the provider organization. Immutable after creation.",
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
			"project_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "System-assigned unique identifier (UUID) of the project.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the project was created (RFC3339 format).",
			},
		},
	}
}

func resourceProjectCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	name := d.Get("name").(string)
	organization := d.Get("organization").(string)
	if organization == "" {
		organization = config.baseConfig.Context.Organization
	}
	displayName := d.Get("display_name").(string)

	// Get user labels
	var userLabels map[string]string
	if labels, ok := d.GetOk("user_labels"); ok {
		userLabels = make(map[string]string)
		for k, v := range labels.(map[string]interface{}) {
			userLabels[k] = v.(string)
		}
	}

	req, err := BuildProjectCreateRequest(name, organization, displayName, userLabels)
	if err != nil {
		return diag.Errorf("error building project create request: %s", err)
	}

	project, err := config.Client.IAM().Projects().Create(ctx, req)
	if err != nil {
		return diag.Errorf("error creating project %s: %s", name, err)
	}

	d.SetId(project.Metadata.Id)

	// Wait for the project to be accessible via project-scoped API calls.
	// A newly created project may not be immediately usable — the API
	// returns 403 on project-scoped endpoints until the project is fully
	// propagated in the auth system.
	if err := config.WaitForProject(ctx, name, d.Timeout(schema.TimeoutCreate)); err != nil {
		return diag.Diagnostics{{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("project %s created but not yet accessible: %s", name, err),
		}}
	}

	return resourceProjectRead(ctx, d, meta)
}

func resourceProjectRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	project, err := config.Client.IAM().Projects().Get(ctx, d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading project: %s", err)
	}

	diags = setDiag(d, "name", project.Metadata.Id, diags)
	diags = setDiag(d, "project_id", project.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", project.Metadata.CreationTimestamp.Format(time.RFC3339), diags)

	// Set organization from spec
	diags = setDiag(d, "organization", project.Spec.Organization, diags)

	// Set display name if present
	if project.Spec.Name != nil {
		diags = setDiag(d, "display_name", *project.Spec.Name, diags)
	}

	if project.Metadata.UserLabels != nil && len(*project.Metadata.UserLabels) > 0 {
		diags = setDiag(d, "user_labels", flattenLabels(project.Metadata.UserLabels), diags)
	}

	return diags
}

func resourceProjectUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	if d.HasChanges("display_name", "user_labels") {
		project, err := config.Client.IAM().Projects().Get(ctx, d.Id())
		if err != nil {
			return diag.Errorf("error reading project %s: %s", d.Id(), err)
		}

		// Update display name
		if d.HasChange("display_name") {
			displayName := d.Get("display_name").(string)
			if displayName != "" {
				project.Spec.Name = &displayName
			} else {
				project.Spec.Name = nil
			}
		}

		// Update user labels
		if d.HasChange("user_labels") {
			if labels, ok := d.GetOk("user_labels"); ok {
				userLabels := make(iamtypes.UserLabels)
				for k, v := range labels.(map[string]interface{}) {
					userLabels[k] = v.(string)
				}
				project.Metadata.UserLabels = &userLabels
			} else {
				project.Metadata.UserLabels = nil
			}
		}

		_, err = config.Client.IAM().Projects().Patch(ctx, d.Id(), project)
		if err != nil {
			return diag.Errorf("error updating project %s: %s", d.Id(), err)
		}
	}

	return resourceProjectRead(ctx, d, meta)
}

func resourceProjectDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	name := d.Id()

	err := config.Client.IAM().Projects().Delete(ctx, name)
	if err != nil {
		if !isNotFoundError(err) {
			return diag.Errorf("error deleting project %s: %s", name, err)
		}
	}

	if err := config.WaitForProjectDeletion(ctx, name, d.Timeout(schema.TimeoutDelete)); err != nil {
		return diag.Errorf("error waiting for project %s to be deleted: %s", name, err)
	}

	d.SetId("")
	return nil
}
