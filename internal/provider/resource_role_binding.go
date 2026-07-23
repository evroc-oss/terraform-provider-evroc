// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"strings"
	"time"

	iamtypes "github.com/evroc-oss/evroc-go-sdk/types/iam"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	roleFQIDPrefix        = "/iam/roles/"
	roleBindingKind       = "RoleBinding"
	roleBindingAPIVersion = "iam/v1beta1"
)

func normalizeRoleFQID(role string) string {
	if strings.HasPrefix(role, roleFQIDPrefix) {
		return role
	}
	return roleFQIDPrefix + role
}

func suppressRoleFQIDDiff(_, old, new string, _ *schema.ResourceData) bool {
	return normalizeRoleFQID(old) == normalizeRoleFQID(new)
}

func roleEntrySchema() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"role": {
				Type:             schema.TypeString,
				Required:         true,
				DiffSuppressFunc: suppressRoleFQIDDiff,
				Description:      "Role to grant. Can be a short name (e.g., \"computeOperator\") or full FQID (\"/iam/roles/computeOperator\").",
			},
			"resources": {
				Type:     schema.TypeList,
				Optional: true,
				Description: "Optional list of resource FQIDs the role is limited to. " +
					"Supports wildcards (*) for path segments. Omit to grant the role on all applicable resources.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceRoleBinding() *schema.Resource {
	return &schema.Resource{
		Description: "Manages a project-scoped IAM role binding. A role binding grants one or more roles to a principal (user or service account) within a project. " +
			"Only one role binding may exist per principal in a given project.",

		CreateContext: resourceRoleBindingCreate,
		ReadContext:   resourceRoleBindingRead,
		UpdateContext: resourceRoleBindingUpdate,
		DeleteContext: resourceRoleBindingDelete,

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
				Description: "Principal FQID (e.g., /iam/projects/<project>/serviceAccounts/<name> or /iam/users/<uuid>).",
			},
			"roles": {
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    1,
				Description: "Roles granted to the principal. Each entry specifies a role and optionally limits it to specific resources.",
				Elem:        roleEntrySchema(),
			},
			"display_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Optional human-friendly display name for the binding.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Project to create the role binding in. Defaults to the provider project.",
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

func expandRoleEntries(raw []interface{}) []iamtypes.RoleEntry {
	entries := make([]iamtypes.RoleEntry, len(raw))
	for i, v := range raw {
		m := v.(map[string]interface{})
		entry := iamtypes.RoleEntry{
			Name: normalizeRoleFQID(m["role"].(string)),
		}
		if resList, ok := m["resources"].([]interface{}); ok && len(resList) > 0 {
			resources := make([]string, len(resList))
			for j, r := range resList {
				resources[j] = r.(string)
			}
			entry.Resources = &resources
		}
		entries[i] = entry
	}
	return entries
}

func flattenRoleEntries(entries []iamtypes.RoleEntry) []map[string]interface{} {
	result := make([]map[string]interface{}, len(entries))
	for i, e := range entries {
		m := map[string]interface{}{
			"role": e.Name,
		}
		if e.Resources != nil {
			m["resources"] = *e.Resources
		} else {
			m["resources"] = []string{}
		}
		result[i] = m
	}
	return result
}

func resourceRoleBindingCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	name := d.Get("name").(string)
	principal := d.Get("principal").(string)
	roles := expandRoleEntries(d.Get("roles").([]interface{}))
	project := resolveProject(d, config)

	req := &iamtypes.RolebindingRequest{
		ApiVersion: roleBindingAPIVersion,
		Kind:       roleBindingKind,
		Metadata: iamtypes.GlobalProjectMetadataRequest{
			Id:      name,
			Project: &project,
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

	rb, err := client.IAM().RoleBindings().CreateProjectRoleBinding(ctx, req)
	if err != nil {
		return diag.Errorf("error creating role binding %s: %s", name, err)
	}

	d.SetId(rb.Metadata.Id)

	return resourceRoleBindingRead(ctx, d, meta)
}

func resourceRoleBindingRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	rb, err := client.IAM().RoleBindings().GetProjectRoleBinding(ctx, d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading role binding: %s", err)
	}

	diags = setDiag(d, "name", rb.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "uid", rb.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", rb.Metadata.CreationTimestamp.Format(time.RFC3339), diags)
	diags = setDiag(d, "principal", rb.Spec.Principal, diags)
	diags = setDiag(d, "roles", flattenRoleEntries(rb.Spec.Roles), diags)

	if rb.Spec.Name != nil {
		diags = setDiag(d, "display_name", *rb.Spec.Name, diags)
	}

	if rb.Metadata.UserLabels != nil && len(*rb.Metadata.UserLabels) > 0 {
		diags = setDiag(d, "user_labels", flattenLabels(rb.Metadata.UserLabels), diags)
	}

	return diags
}

func resourceRoleBindingUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	if d.HasChanges("roles", "display_name", "user_labels") {
		rb, err := client.IAM().RoleBindings().GetProjectRoleBinding(ctx, d.Id())
		if err != nil {
			return diag.Errorf("error reading role binding %s for update: %s", d.Id(), err)
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

		patch := &iamtypes.RolebindingPatchRequest{
			ApiVersion: &rb.ApiVersion,
			Kind:       &rb.Kind,
			Spec:       &rb.Spec,
		}

		_, err = client.IAM().RoleBindings().PatchProjectRoleBinding(ctx, d.Id(), patch)
		if err != nil {
			return diag.Errorf("error updating role binding %s: %s", d.Id(), err)
		}
	}

	return resourceRoleBindingRead(ctx, d, meta)
}

func resourceRoleBindingDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	err := client.IAM().RoleBindings().DeleteProjectRoleBinding(ctx, d.Id())
	if err != nil {
		if !isNotFoundError(err) {
			return diag.Errorf("error deleting role binding %s: %s", d.Id(), err)
		}
	}

	d.SetId("")
	return nil
}
