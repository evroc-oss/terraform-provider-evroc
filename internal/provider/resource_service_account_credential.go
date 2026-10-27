// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"path"
	"time"

	evrociam "github.com/evroc-oss/evroc-go-sdk/iam"
	iamtypes "github.com/evroc-oss/evroc-go-sdk/types/iam"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func serviceAccountIDFromRef(fqid string) string {
	return path.Base(fqid)
}

func resourceServiceAccountCredential() *schema.Resource {
	return &schema.Resource{
		Description: "Manages an evroc IAM service account credential. Supports RS256 JWT and HMAC-SigV4 credentials. Secrets are available only at creation and stored in Terraform state.",

		CreateContext: resourceServiceAccountCredentialCreate,
		ReadContext:   resourceServiceAccountCredentialRead,
		DeleteContext: resourceServiceAccountCredentialDelete,
		CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
			if d.Get("type") == string(iamtypes.HmacSigv4) {
				if _, ok := d.GetOk("access_token_lifetime"); ok {
					return fmt.Errorf("access_token_lifetime is only valid for rs256-jwt credentials")
				}
			}
			return nil
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"type":              {Type: schema.TypeString, Optional: true, Default: string(iamtypes.Rs256Jwt), ForceNew: true, Description: "Credential type: rs256-jwt (default) for OAuth or hmac-sigv4 for AWS Signature V4.", ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{string(iamtypes.Rs256Jwt), string(iamtypes.HmacSigv4)}, false))},
			"access_key_id":     {Type: schema.TypeString, Computed: true, Description: "Access key identifier for HMAC credentials."},
			"secret_access_key": {Type: schema.TypeString, Computed: true, Sensitive: true, Description: "HMAC secret returned only at creation. Stored in Terraform state; protect state access."},
			"revoked_at":        {Type: schema.TypeString, Computed: true, Description: "Revocation timestamp in RFC3339 format, or empty if not revoked."},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Unique identifier for the credential. Immutable after creation.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Project this credential belongs to. Defaults to the provider project.",
			},
			"service_account_ref": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Fully qualified ID of the parent service account (e.g., /iam/projects/<project>/serviceAccounts/<name>).",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Human-readable description of the credential's purpose.",
			},
			"expires_at": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Expiration timestamp for the credential (RFC3339 format).",
			},
			"access_token_lifetime": {
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Description: "Access token lifetime in seconds for RS256 JWT credentials. Cannot be set for hmac-sigv4.",
			},
			// Computed fields
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
			"private_key_jwk": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "Base64-encoded private key in JWK format. Only available at creation time.",
			},
		},
	}
}

func resourceServiceAccountCredentialCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	name := d.Get("name").(string)
	saRef := d.Get("service_account_ref").(string)
	project := resolveProject(d, config)

	expiresAtStr := d.Get("expires_at").(string)
	expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
	if err != nil {
		return diag.Errorf("invalid expires_at format, expected RFC3339: %s", err)
	}

	builder := evrociam.NewServiceAccountCredentialBuilder(name, project, saRef, expiresAt)
	if d.Get("type") == string(iamtypes.HmacSigv4) {
		builder = builder.WithHMACSigV4()
	}

	if v, ok := d.GetOk("description"); ok {
		builder = builder.WithDescription(v.(string))
	}
	if v, ok := d.GetOk("access_token_lifetime"); ok {
		builder = builder.WithAccessTokenLifetime(v.(int))
	}

	saID := serviceAccountIDFromRef(saRef)
	cred, err := builder.Create(ctx, client.IAM().ServiceAccountCredentials(saID))
	if err != nil {
		return diag.Errorf("error creating service account credential %s: %s", name, err)
	}

	d.SetId(cred.Metadata.Id)
	d.Set("project", resolveProject(d, config))

	if cred.Status.SecretAccessKey != nil {
		if err := d.Set("secret_access_key", *cred.Status.SecretAccessKey); err != nil {
			return diag.FromErr(err)
		}
	}
	if cred.Status.PrivateKeyJwk != nil {
		d.Set("private_key_jwk", *cred.Status.PrivateKeyJwk)
	}

	return resourceServiceAccountCredentialRead(ctx, d, meta)
}

func resourceServiceAccountCredentialRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	saRef := d.Get("service_account_ref").(string)
	saID := serviceAccountIDFromRef(saRef)
	cred, err := client.IAM().ServiceAccountCredentials(saID).Get(ctx, d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading service account credential: %s", err)
	}

	diags = setDiag(d, "name", cred.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
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

func resourceServiceAccountCredentialDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	saRef := d.Get("service_account_ref").(string)
	saID := serviceAccountIDFromRef(saRef)
	err := client.IAM().ServiceAccountCredentials(saID).Delete(ctx, d.Id())
	if err != nil {
		if !isNotFoundError(err) {
			return diag.Errorf("error deleting service account credential %s: %s", d.Id(), err)
		}
	}

	d.SetId("")
	return nil
}
