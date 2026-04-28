// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"time"

	iamtypes "github.com/evroc-oss/evroc-go-sdk/types/iam"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourcePermissionSet() *schema.Resource {
	return &schema.Resource{
		Description: "Manages an evroc permission set for granting user access to a project.",

		CreateContext: resourcePermissionSetCreate,
		ReadContext:   resourcePermissionSetRead,
		UpdateContext: resourcePermissionSetUpdate,
		DeleteContext: resourcePermissionSetDelete,

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
				Description: "Unique identifier for the permission set. Immutable after creation.",
			},
			"project": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Project ID this permission set belongs to.",
			},
			"email": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Email address of the user to grant permissions to.",
			},
			"admin": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether this permission set grants admin privileges.",
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
			"permission_set_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "System-assigned unique identifier (UUID) of the permission set.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the permission set was created (RFC3339 format).",
			},
		},
	}
}

func resourcePermissionSetCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	name := d.Get("name").(string)
	project := d.Get("project").(string)
	email := d.Get("email").(string)
	admin := d.Get("admin").(bool)

	// Get user labels
	var userLabels map[string]string
	if labels, ok := d.GetOk("user_labels"); ok {
		userLabels = make(map[string]string)
		for k, v := range labels.(map[string]interface{}) {
			userLabels[k] = v.(string)
		}
	}

	req := BuildPermissionSetCreateRequest(name, project, email, admin, userLabels)

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	ps, err := client.IAM().PermissionSets().Create(ctx, req)
	if err != nil {
		return diag.Errorf("error creating permission set %s: %s", name, err)
	}

	d.SetId(ps.Metadata.Id)

	return resourcePermissionSetRead(ctx, d, meta)
}

func resourcePermissionSetRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	ps, err := client.IAM().PermissionSets().Get(ctx, d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading permission set: %s", err)
	}

	diags = setDiag(d, "name", ps.Metadata.Id, diags)
	diags = setDiag(d, "permission_set_id", ps.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", ps.Metadata.CreationTimestamp.Format(time.RFC3339), diags)
	diags = setDiag(d, "email", ps.Spec.Subject.User.Email, diags)
	diags = setDiag(d, "admin", ps.Spec.Admin, diags)
	if ps.Metadata.UserLabels != nil && len(*ps.Metadata.UserLabels) > 0 {
		diags = setDiag(d, "user_labels", flattenLabels(ps.Metadata.UserLabels), diags)
	}

	return diags
}

func resourcePermissionSetUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	if d.HasChanges("admin", "user_labels") {
		ps, err := client.IAM().PermissionSets().Get(ctx, d.Id())
		if err != nil {
			return diag.Errorf("error reading permission set %s: %s", d.Id(), err)
		}

		if d.HasChange("admin") {
			ps.Spec.Admin = d.Get("admin").(bool)
		}

		if d.HasChange("user_labels") {
			if labels, ok := d.GetOk("user_labels"); ok {
				userLabels := make(iamtypes.UserLabels)
				for k, v := range labels.(map[string]interface{}) {
					userLabels[k] = v.(string)
				}
				ps.Metadata.UserLabels = &userLabels
			} else {
				ps.Metadata.UserLabels = nil
			}
		}

		_, err = client.IAM().PermissionSets().Patch(ctx, d.Id(), ps)
		if err != nil {
			return diag.Errorf("error updating permission set %s: %s", d.Id(), err)
		}
	}

	return resourcePermissionSetRead(ctx, d, meta)
}

func resourcePermissionSetDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	err := client.IAM().PermissionSets().Delete(ctx, d.Id())
	if err != nil {
		if !isNotFoundError(err) {
			return diag.Errorf("error deleting permission set %s: %s", d.Id(), err)
		}
	}

	d.SetId("")
	return nil
}
