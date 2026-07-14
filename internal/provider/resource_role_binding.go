// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	iamtypes "github.com/evroc-oss/evroc-go-sdk/types/iam"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const roleFQIDPrefix = "/iam/roles/"

func normalizeRoleFQID(role string) string {
	if strings.HasPrefix(role, roleFQIDPrefix) {
		return role
	}
	return roleFQIDPrefix + role
}

func suppressRoleFQIDDiff(_, old, new string, _ *schema.ResourceData) bool {
	return normalizeRoleFQID(old) == normalizeRoleFQID(new)
}

func resourceRoleBinding() *schema.Resource {
	return &schema.Resource{
		Description: "Assigns an IAM role to a principal (user or service account) at project scope. Removing this resource revokes the role.",

		CreateContext: resourceRoleBindingCreate,
		ReadContext:   resourceRoleBindingRead,
		DeleteContext: resourceRoleBindingDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"principal": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Fully qualified principal FQID (e.g., /iam/projects/<project>/serviceAccounts/<name> or /iam/users/<uuid>).",
			},
			"role": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				DiffSuppressFunc: suppressRoleFQIDDiff,
				Description:      "Role to assign. Can be a short name (e.g., \"computeOperator\") or full FQID (\"/iam/roles/computeOperator\").",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Project to assign the role in. Defaults to the provider project.",
			},
		},
	}
}

func resourceRoleBindingCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	principal := d.Get("principal").(string)
	role := normalizeRoleFQID(d.Get("role").(string))

	req := &iamtypes.AssignRoleRequest{
		Principal: principal,
		Role:      role,
	}

	_, err := client.IAM().RoleBindings().AssignProjectRole(ctx, req)
	if err != nil {
		return diag.Errorf("error assigning role %s to %s: %s", role, principal, err)
	}

	d.SetId(fmt.Sprintf("%s/%s", principal, role))
	d.Set("project", resolveProject(d, config))

	return nil
}

func resourceRoleBindingRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	d.Set("project", resolveProject(d, config))
	d.Set("principal", d.Get("principal").(string))
	d.Set("role", d.Get("role").(string))

	return nil
}

func resourceRoleBindingDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	principal := d.Get("principal").(string)
	role := normalizeRoleFQID(d.Get("role").(string))

	req := &iamtypes.RevokeRoleRequest{
		Principal: principal,
		Role:      role,
	}

	_, err := client.IAM().RoleBindings().RevokeProjectRole(ctx, req)
	if err != nil {
		return diag.Errorf("error revoking role %s from %s: %s", role, principal, err)
	}

	d.SetId("")
	return nil
}
