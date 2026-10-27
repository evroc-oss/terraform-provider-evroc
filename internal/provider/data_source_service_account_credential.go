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

func dataSourceServiceAccountCredential() *schema.Resource {
	return &schema.Resource{
		Description: "Retrieves metadata for an existing RS256 JWT or HMAC-SigV4 service account credential. Private keys and HMAC secrets are only returned during creation and cannot be retrieved by this data source.",

		ReadContext: dataSourceServiceAccountCredentialRead,

		Schema: map[string]*schema.Schema{
			"type":          {Type: schema.TypeString, Computed: true, Description: "Credential type: rs256-jwt or hmac-sigv4."},
			"access_key_id": {Type: schema.TypeString, Computed: true, Description: "Access key identifier for HMAC credentials."},
			"revoked_at":    {Type: schema.TypeString, Computed: true, Description: "Revocation timestamp in RFC3339 format, or empty if not revoked."},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the credential to look up.",
			},
			"service_account_ref": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Fully qualified ID of the parent service account.",
			},
			"project": {
				Optional:    true,
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Project containing the credential. Defaults to the provider project; set it to match the project in service_account_ref.",
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Human-readable description of the credential's purpose.",
			},
			"expires_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Expiration timestamp for the credential (RFC3339 format).",
			},
			"access_token_lifetime": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Access token lifetime in seconds for RS256 JWT credentials.",
			},
			"credential_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "System-assigned unique identifier (UUID) of the credential.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the credential was created (RFC3339 format).",
			},
		},
	}
}

func dataSourceServiceAccountCredentialRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	name := d.Get("name").(string)
	saRef := d.Get("service_account_ref").(string)
	saID := serviceAccountIDFromRef(saRef)

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}
	cred, err := client.IAM().ServiceAccountCredentials(saID).Get(ctx, name)
	if err != nil {
		return diag.Errorf("error reading service account credential %s: %s", name, err)
	}

	d.SetId(cred.Metadata.Id)
	diags = setDiag(d, "project", client.DefaultProject(), diags)

	diags = setDiag(d, "name", cred.Metadata.Id, diags)
	diags = setDiag(d, "credential_id", cred.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", cred.Metadata.CreationTimestamp.Format(time.RFC3339), diags)
	diags = setDiag(d, "service_account_ref", cred.Spec.AccountRef.Fqid, diags)
	diags = setDiag(d, "expires_at", cred.Spec.ExpiresAt.Format(time.RFC3339), diags)

	if cred.Spec.Description != nil {
		diags = setDiag(d, "description", *cred.Spec.Description, diags)
	}

	if cred.Spec.Rs256Jwt != nil && cred.Spec.Rs256Jwt.AccessTokenLifetime != nil {
		diags = setDiag(d, "access_token_lifetime", *cred.Spec.Rs256Jwt.AccessTokenLifetime, diags)
	}

	return flattenCredentialType(d, cred, diags)
}

func flattenCredentialType(d *schema.ResourceData, cred *iamtypes.Serviceaccountcredential, diags diag.Diagnostics) diag.Diagnostics {
	diags = setDiag(d, "type", string(cred.Spec.Type), diags)
	accessKey := ""
	if cred.Spec.HmacSigv4 != nil {
		accessKey = cred.Spec.HmacSigv4.AccessKeyId
	}
	diags = setDiag(d, "access_key_id", accessKey, diags)
	revoked := ""
	if cred.Status.RevokedAt != nil {
		revoked = cred.Status.RevokedAt.Format(time.RFC3339)
	}
	return setDiag(d, "revoked_at", revoked, diags)
}
