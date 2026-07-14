// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"time"

	evrociam "github.com/evroc-oss/evroc-go-sdk/iam"
	iamtypes "github.com/evroc-oss/evroc-go-sdk/types/iam"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceServiceAccount() *schema.Resource {
	return &schema.Resource{
		Description: "Manages an evroc IAM service account. Service accounts are non-human identities used for programmatic access to evroc APIs.",

		CreateContext: resourceServiceAccountCreate,
		ReadContext:   resourceServiceAccountRead,
		UpdateContext: resourceServiceAccountUpdate,
		DeleteContext: resourceServiceAccountDelete,

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
				Description: "Unique identifier for the service account. Immutable after creation.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Project this service account belongs to. Defaults to the provider project.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Human-readable description of the service account's purpose.",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether the service account is enabled.",
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
			"service_account_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "System-assigned unique identifier (UUID) of the service account.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the service account was created (RFC3339 format).",
			},
			"oauth_client_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "OAuth client ID assigned to the service account.",
			},
			"fqid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified identifier for the service account (e.g., /iam/projects/<project>/serviceAccounts/<name>).",
			},
		},
	}
}

func resourceServiceAccountCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	name := d.Get("name").(string)
	project := resolveProject(d, config)

	builder := evrociam.NewServiceAccountBuilder(name, project).
		WithEnabled(d.Get("enabled").(bool))

	if v, ok := d.GetOk("description"); ok {
		builder = builder.WithDescription(v.(string))
	}

	if labels, ok := d.GetOk("user_labels"); ok {
		userLabels := make(map[string]string)
		for k, v := range labels.(map[string]interface{}) {
			userLabels[k] = v.(string)
		}
		builder = builder.WithLabels(userLabels)
	}

	sa, err := builder.Create(ctx, client.IAM().ServiceAccounts())
	if err != nil {
		return diag.Errorf("error creating service account %s: %s", name, err)
	}

	d.SetId(sa.Metadata.Id)
	d.Set("project", resolveProject(d, config))

	return resourceServiceAccountRead(ctx, d, meta)
}

func resourceServiceAccountRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	sa, err := client.IAM().ServiceAccounts().Get(ctx, d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading service account: %s", err)
	}

	diags = setDiag(d, "name", sa.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "service_account_id", sa.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", sa.Metadata.CreationTimestamp.Format(time.RFC3339), diags)

	if sa.Spec.Description != nil {
		diags = setDiag(d, "description", *sa.Spec.Description, diags)
	}

	enabled := true
	if sa.Spec.Enabled != nil {
		enabled = *sa.Spec.Enabled
	}
	diags = setDiag(d, "enabled", enabled, diags)

	if sa.Status.OauthClientId != nil {
		diags = setDiag(d, "oauth_client_id", *sa.Status.OauthClientId, diags)
	}

	if sa.Metadata.UserLabels != nil && len(*sa.Metadata.UserLabels) > 0 {
		diags = setDiag(d, "user_labels", flattenLabels(sa.Metadata.UserLabels), diags)
	}

	project := resolveProject(d, config)
	fqid := fmt.Sprintf("/iam/projects/%s/serviceAccounts/%s", project, sa.Metadata.Id)
	diags = setDiag(d, "fqid", fqid, diags)

	return diags
}

func resourceServiceAccountUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	if d.HasChanges("description", "enabled", "user_labels") {
		sa, err := client.IAM().ServiceAccounts().Get(ctx, d.Id())
		if err != nil {
			return diag.Errorf("error reading service account %s: %s", d.Id(), err)
		}

		if d.HasChange("description") {
			if v, ok := d.GetOk("description"); ok {
				desc := v.(string)
				sa.Spec.Description = &desc
			} else {
				sa.Spec.Description = nil
			}
		}

		if d.HasChange("enabled") {
			enabled := d.Get("enabled").(bool)
			sa.Spec.Enabled = &enabled
		}

		if d.HasChange("user_labels") {
			if labels, ok := d.GetOk("user_labels"); ok {
				userLabels := make(iamtypes.UserLabels)
				for k, v := range labels.(map[string]interface{}) {
					userLabels[k] = v.(string)
				}
				sa.Metadata.UserLabels = &userLabels
			} else {
				sa.Metadata.UserLabels = nil
			}
		}

		_, err = client.IAM().ServiceAccounts().Patch(ctx, d.Id(), sa)
		if err != nil {
			return diag.Errorf("error updating service account %s: %s", d.Id(), err)
		}
	}

	return resourceServiceAccountRead(ctx, d, meta)
}

func resourceServiceAccountDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	err := client.IAM().ServiceAccounts().Delete(ctx, d.Id())
	if err != nil {
		if !isNotFoundError(err) {
			return diag.Errorf("error deleting service account %s: %s", d.Id(), err)
		}
	}

	d.SetId("")
	return nil
}
