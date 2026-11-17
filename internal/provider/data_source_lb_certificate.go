// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceLBCertificate() *schema.Resource {
	return &schema.Resource{
		Description: "Looks up load balancer certificate metadata and readiness. Certificate content and private keys are never returned.",
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Certificate resource name, unique within the project and region.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Project containing the certificate. Defaults to the provider project.",
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Region containing the certificate. Defaults to the provider region.",
			},
			"user_labels": {
				Type:        schema.TypeMap,
				Description: "Labels set when creating the certificate. The API does not support updating certificate labels.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed: true,
			},
			"system_labels": {
				Type:        schema.TypeMap,
				Description: "System-managed resource labels.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed: true,
			},
			"certificate_id": {
				Type:        schema.TypeString,
				Description: "System-assigned certificate resource UUID.",
				Computed:    true,
			},
			"created_at": {
				Type:        schema.TypeString,
				Description: "Creation timestamp in RFC3339 format.",
				Computed:    true,
			},
			"fqid": {
				Type:        schema.TypeString,
				Description: "Fully qualified certificate reference for HTTPS listeners.",
				Computed:    true,
			},
			"ready": {
				Type:        schema.TypeBool,
				Description: "Whether the API reports the certificate ready for use.",
				Computed:    true,
			},
			"certificate_sha256_fingerprint": {
				Type:        schema.TypeString,
				Description: "Observed SHA256 fingerprint of the leaf certificate.",
				Computed:    true,
			},
			"public_key_sha256_fingerprint": {
				Type:        schema.TypeString,
				Description: "Observed SHA256 fingerprint of the key's public key in DER SubjectPublicKeyInfo format.",
				Computed:    true,
			},
			"subject": {
				Type:        schema.TypeString,
				Description: "Observed leaf certificate subject.",
				Computed:    true,
			},
			"issuer": {
				Type:        schema.TypeString,
				Description: "Observed leaf certificate issuer.",
				Computed:    true,
			},
			"not_before": {
				Type:        schema.TypeString,
				Description: "Start of the certificate validity window in RFC3339 format.",
				Computed:    true,
			},
			"not_after": {
				Type:        schema.TypeString,
				Description: "End of the certificate validity window in RFC3339 format.",
				Computed:    true,
			},
			"subject_alternative_names": {
				Type:        schema.TypeSet,
				Description: "DNS names and IP addresses in the leaf certificate SAN extension.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed: true,
			},
		},
		ReadContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
			client, diags := resolveClient(d, meta.(*ProviderConfig))
			if diags.HasError() {
				return diags
			}
			certificate, err := client.LoadBalancer().LoadBalancerCertificates().Get(ctx, d.Get("name").(string))
			if err != nil {
				return diag.Errorf("error reading load balancer certificate: %s", err)
			}
			d.SetId(certificate.Metadata.Id)
			return flattenLBCertificate(d, client, certificate)
		},
	}
}
