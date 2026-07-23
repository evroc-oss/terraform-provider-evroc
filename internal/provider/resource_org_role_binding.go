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

func resourceOrgRoleBinding() *schema.Resource {
	return &schema.Resource{
		Description: "Manages an organization-scoped IAM role binding. A role binding grants one or more roles to a principal (user or service account) across the entire organization. " +
			"Only one role binding may exist per principal in a given organization.",

		CreateContext: resourceOrgRoleBindingCreate,
		ReadContext:   resourceOrgRoleBindingRead,
		UpdateContext: resourceOrgRoleBindingUpdate,
		DeleteContext: resourceOrgRoleBindingDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(5 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				Description: "Unique identifier for the role binding. Any name is accepted, but by convention names follow " +
					"u-{user uuid} for users and sa-{project}-{service account name} for service accounts.",
			},
			"principal": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Principal FQID (e.g., /iam/users/<uuid> or /iam/projects/<project>/serviceAccounts/<name>).",
			},
			"roles": {
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    1,
				Description: "Roles granted to the principal at organization scope.",
				Elem:        roleEntrySchema(),
			},
			"display_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Optional human-friendly display name for the binding.",
			},
			"organization": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Organization to create the role binding in. Defaults to the provider organization.",
			},
			"user_labels": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "User-defined labels (key/value pairs) for organizing and selecting resources.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			// Computed
			"uid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "System-assigned unique identifier (UUID) of the role binding.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the role binding was created (RFC3339 format).",
			},
		},
	}
}

func resolveOrganization(d *schema.ResourceData, config *ProviderConfig) string {
	if v, ok := d.GetOk("organization"); ok {
		if org := v.(string); org != "" {
			return org
		}
	}
	if config.baseConfig != nil {
		return config.baseConfig.Context.Organization
	}
	return ""
}

func resourceOrgRoleBindingCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	name := d.Get("name").(string)
	principal := d.Get("principal").(string)
	roles := expandRoleEntries(d.Get("roles").([]interface{}))
	org := resolveOrganization(d, config)

	req := &iamtypes.OrgRequest{
		ApiVersion: roleBindingAPIVersion,
		Kind:       roleBindingKind,
		Metadata: iamtypes.GlobalOrgMetadataRequest{
			Id:           name,
			Organization: &org,
		},
		Spec: iamtypes.RolebindingSpec{
			Principal: principal,
			Roles:     roles,
		},
	}

	if v, ok := d.GetOk("display_name"); ok {
		displayName := v.(string)
		req.Spec.Name = &displayName
	}

	if labels, ok := d.GetOk("user_labels"); ok {
		userLabels := iamtypes.UserLabels{}
		for k, v := range labels.(map[string]interface{}) {
			userLabels[k] = v.(string)
		}
		req.Metadata.UserLabels = &userLabels
	}

	rb, err := config.Client.IAM().RoleBindings().CreateOrgRoleBinding(ctx, req)
	if err != nil {
		return diag.Errorf("error creating org role binding %s: %s", name, err)
	}

	d.SetId(rb.Metadata.Id)

	return resourceOrgRoleBindingRead(ctx, d, meta)
}

func resourceOrgRoleBindingRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	rb, err := config.Client.IAM().RoleBindings().GetOrgRoleBinding(ctx, d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading org role binding: %s", err)
	}

	diags = setDiag(d, "name", rb.Metadata.Id, diags)
	diags = setDiag(d, "uid", rb.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", rb.Metadata.CreationTimestamp.Format(time.RFC3339), diags)
	diags = setDiag(d, "principal", rb.Spec.Principal, diags)
	diags = setDiag(d, "roles", flattenRoleEntries(rb.Spec.Roles), diags)
	diags = setDiag(d, "organization", resolveOrganization(d, config), diags)

	if rb.Spec.Name != nil {
		diags = setDiag(d, "display_name", *rb.Spec.Name, diags)
	}

	if rb.Metadata.UserLabels != nil && len(*rb.Metadata.UserLabels) > 0 {
		diags = setDiag(d, "user_labels", flattenLabels(rb.Metadata.UserLabels), diags)
	}

	return diags
}

func resourceOrgRoleBindingUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	if d.HasChanges("roles", "display_name", "user_labels") {
		rb, err := config.Client.IAM().RoleBindings().GetOrgRoleBinding(ctx, d.Id())
		if err != nil {
			return diag.Errorf("error reading org role binding %s for update: %s", d.Id(), err)
		}

		if d.HasChange("roles") {
			rb.Spec.Roles = expandRoleEntries(d.Get("roles").([]interface{}))
		}

		if d.HasChange("display_name") {
			if v, ok := d.GetOk("display_name"); ok {
				displayName := v.(string)
				rb.Spec.Name = &displayName
			} else {
				rb.Spec.Name = nil
			}
		}

		if d.HasChange("user_labels") {
			if labels, ok := d.GetOk("user_labels"); ok {
				userLabels := iamtypes.UserLabels{}
				for k, v := range labels.(map[string]interface{}) {
					userLabels[k] = v.(string)
				}
				rb.Metadata.UserLabels = &userLabels
			} else {
				rb.Metadata.UserLabels = nil
			}
		}

		patch := &iamtypes.OrgPatchRequest{
			ApiVersion: &rb.ApiVersion,
			Kind:       &rb.Kind,
			Spec:       &rb.Spec,
		}

		_, err = config.Client.IAM().RoleBindings().PatchOrgRoleBinding(ctx, d.Id(), patch)
		if err != nil {
			return diag.Errorf("error updating org role binding %s: %s", d.Id(), err)
		}
	}

	return resourceOrgRoleBindingRead(ctx, d, meta)
}

func resourceOrgRoleBindingDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	err := config.Client.IAM().RoleBindings().DeleteOrgRoleBinding(ctx, d.Id())
	if err != nil {
		if !isNotFoundError(err) {
			return diag.Errorf("error deleting org role binding %s: %s", d.Id(), err)
		}
	}

	d.SetId("")
	return nil
}
