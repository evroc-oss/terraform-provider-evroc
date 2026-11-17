// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"time"

	evroc "github.com/evroc-oss/evroc-go-sdk"
	"github.com/evroc-oss/evroc-go-sdk/loadbalancer"
	lbtypes "github.com/evroc-oss/evroc-go-sdk/types/loadbalancer"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceLBCertificate() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a certificate and private key for load balancer HTTPS termination. Sensitive PEM inputs are stored in Terraform state; protect access to the state.",
		CreateContext: resourceLBCertificateCreate,
		ReadContext:   resourceLBCertificateRead,
		UpdateContext: resourceLBCertificateUpdate,
		DeleteContext: resourceLBCertificateDelete,
		CustomizeDiff: func(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
			if !d.NewValueKnown("certificate_pem") || !d.NewValueKnown("private_key_pem") {
				if err := d.SetNewComputed("certificate_sha256_fingerprint"); err != nil {
					return err
				}
				return d.SetNewComputed("public_key_sha256_fingerprint")
			}
			certHash, keyHash, err := certificateFingerprints(d.Get("certificate_pem").(string), d.Get("private_key_pem").(string))
			if err != nil {
				return err
			}

			// Plan repair when the API reports content different from the configured pair.
			if err := d.SetNew("certificate_sha256_fingerprint", certHash); err != nil {
				return err
			}
			return d.SetNew("public_key_sha256_fingerprint", keyHash)
		},
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
				Description: "Certificate resource name, unique within the project and region.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Project containing the certificate. Defaults to the provider project.",
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Region containing the certificate. Defaults to the provider region.",
			},
			"certificate_pem": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				Description: "PEM certificate chain, leaf first followed by intermediates, without the root CA. Not returned by the API; retained in Terraform state.",
			},
			"private_key_pem": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				Description: "PEM private key matching the leaf certificate. Not returned by the API; retained in Terraform state.",
			},
			"user_labels": {
				Type:        schema.TypeMap,
				Optional:    true,
				ForceNew:    true,
				Description: "Labels set when creating the certificate. The API does not support updating certificate labels.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"system_labels": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "System-managed resource labels.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"certificate_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "System-assigned certificate resource UUID.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Creation timestamp in RFC3339 format.",
			},
			"fqid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified certificate reference for HTTPS listeners.",
			},
			"ready": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the API reports the certificate ready for use.",
			},
			"certificate_sha256_fingerprint": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Observed SHA256 fingerprint of the leaf certificate.",
			},
			"public_key_sha256_fingerprint": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Observed SHA256 fingerprint of the key's public key in DER SubjectPublicKeyInfo format.",
			},
			"subject": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Observed leaf certificate subject.",
			},
			"issuer": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Observed leaf certificate issuer.",
			},
			"not_before": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Start of the certificate validity window in RFC3339 format.",
			},
			"not_after": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "End of the certificate validity window in RFC3339 format.",
			},
			"subject_alternative_names": {
				Type:        schema.TypeSet,
				Computed:    true,
				Description: "DNS names and IP addresses in the leaf certificate SAN extension.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func certificateFingerprints(certificatePEM, privateKeyPEM string) (string, string, error) {
	pair, err := tls.X509KeyPair([]byte(certificatePEM), []byte(privateKeyPEM))
	if err != nil {
		return "", "", fmt.Errorf("certificate_pem and private_key_pem must contain a valid matching PEM certificate and private key")
	}
	leaf, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		return "", "", fmt.Errorf("certificate_pem must contain a valid X.509 leaf certificate")
	}
	return fmt.Sprintf("%x", sha256.Sum256(leaf.Raw)), fmt.Sprintf("%x", sha256.Sum256(leaf.RawSubjectPublicKeyInfo)), nil
}

func resourceLBCertificateCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	certHash, keyHash, err := certificateFingerprints(d.Get("certificate_pem").(string), d.Get("private_key_pem").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	config := meta.(*ProviderConfig)
	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}
	certificates := client.LoadBalancer().LoadBalancerCertificates()
	request := loadbalancer.NewLoadBalancerCertificateBuilder(d.Get("name").(string)).WithLabels(expandUserLabels(d.Get("user_labels").(map[string]interface{}))).Build()
	certificate, err := certificates.Create(ctx, request)
	if err != nil {
		return diag.Errorf("error creating load balancer certificate: %s", err)
	}
	// Keep the record tracked if content upload or asynchronous validation fails.
	d.SetId(certificate.Metadata.Id)
	d.Set("project", resolveProject(d, config))
	if err := certificates.UploadContent(ctx, d.Id(), d.Get("certificate_pem").(string), d.Get("private_key_pem").(string)); err != nil {
		return diag.Errorf("error uploading load balancer certificate: %s", err)
	}
	if err := waitLBCertificate(ctx, certificates, d.Id(), certHash, keyHash, d.Timeout(schema.TimeoutCreate)); err != nil {
		return diag.FromErr(err)
	}
	return resourceLBCertificateRead(ctx, d, meta)
}

func resourceLBCertificateUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, diags := resolveClient(d, meta.(*ProviderConfig))
	if diags.HasError() {
		return diags
	}
	if d.HasChange("certificate_pem") || d.HasChange("private_key_pem") || d.HasChange("certificate_sha256_fingerprint") || d.HasChange("public_key_sha256_fingerprint") {
		certHash, keyHash, err := certificateFingerprints(d.Get("certificate_pem").(string), d.Get("private_key_pem").(string))
		if err != nil {
			return diag.FromErr(err)
		}
		certificates := client.LoadBalancer().LoadBalancerCertificates()
		if err := certificates.UploadContent(ctx, d.Id(), d.Get("certificate_pem").(string), d.Get("private_key_pem").(string)); err != nil {
			return diag.Errorf("error uploading load balancer certificate: %s", err)
		}
		if err := waitLBCertificate(ctx, certificates, d.Id(), certHash, keyHash, d.Timeout(schema.TimeoutUpdate)); err != nil {
			return diag.FromErr(err)
		}
	}
	return resourceLBCertificateRead(ctx, d, meta)
}

func waitLBCertificate(ctx context.Context, certificates *loadbalancer.LoadBalancerCertificatesService, id, certHash, keyHash string, timeout time.Duration) error {
	const pending = "pending"

	// Readiness may still describe the previous upload. Match the desired leaf and
	// public key as well; the API does not expose an intermediate-chain fingerprint.
	wait := &retry.StateChangeConf{Pending: []string{pending}, Target: []string{"ready"}, Timeout: timeout, MinTimeout: time.Second, Refresh: func() (interface{}, string, error) {
		certificate, err := certificates.Get(ctx, id)
		if err != nil {
			return nil, "", err
		}
		if loadbalancer.IsCertificateReady(certificate) && derefString(certificate.Status.CertificateSha256Fingerprint) == certHash && derefString(certificate.Status.PublicKeySha256Fingerprint) == keyHash {
			return certificate, "ready", nil
		}
		return certificate, pending, nil
	}}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("waiting for load balancer certificate %s: %w", id, err)
	}
	return nil
}

func resourceLBCertificateRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, diags := resolveClient(d, meta.(*ProviderConfig))
	if diags.HasError() {
		return diags
	}
	certificate, err := client.LoadBalancer().LoadBalancerCertificates().Get(ctx, d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading load balancer certificate: %s", err)
	}
	return flattenLBCertificate(d, client, certificate)
}

func flattenLBCertificate(d *schema.ResourceData, client *evroc.Client, certificate *lbtypes.Loadbalancercertificate) diag.Diagnostics {
	var diags diag.Diagnostics
	status := certificate.Status
	names := []string{}
	if status.SubjectAlternativeNames != nil {
		names = *status.SubjectAlternativeNames
	}
	notBefore, notAfter := "", ""
	if status.NotBefore != nil {
		notBefore = status.NotBefore.Format(time.RFC3339)
	}
	if status.NotAfter != nil {
		notAfter = status.NotAfter.Format(time.RFC3339)
	}
	// Content is write-only at the API boundary. Preserve configured PEM inputs.
	values := map[string]interface{}{
		"name":                           certificate.Metadata.Id,
		"project":                        client.DefaultProject(),
		"region":                         derefString(certificate.Metadata.Region),
		"certificate_id":                 certificate.Metadata.Uid.String(),
		"created_at":                     certificate.Metadata.CreationTimestamp.Format(time.RFC3339),
		"fqid":                           client.LoadBalancer().LoadBalancerCertificateRef(certificate.Metadata.Id),
		"user_labels":                    flattenLabels(certificate.Metadata.UserLabels),
		"system_labels":                  flattenLabels(certificate.Metadata.SystemLabels),
		"ready":                          loadbalancer.IsCertificateReady(certificate),
		"certificate_sha256_fingerprint": derefString(status.CertificateSha256Fingerprint),
		"public_key_sha256_fingerprint":  derefString(status.PublicKeySha256Fingerprint),
		"subject":                        derefString(status.Subject),
		"issuer":                         derefString(status.Issuer),
		"not_before":                     notBefore,
		"not_after":                      notAfter,
		"subject_alternative_names":      names,
	}
	for field, value := range values {
		diags = setDiag(d, field, value, diags)
	}
	return diags
}

func resourceLBCertificateDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, diags := resolveClient(d, meta.(*ProviderConfig))
	if diags.HasError() {
		return diags
	}
	certificates := client.LoadBalancer().LoadBalancerCertificates()
	if err := certificates.Delete(ctx, d.Id()); err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error deleting load balancer certificate: %s", err)
	}
	if err := certificates.WaitForDeleted(ctx, d.Id(), d.Timeout(schema.TimeoutDelete)); err != nil {
		return diag.Errorf("error waiting for load balancer certificate deletion: %s", err)
	}
	d.SetId("")
	return nil
}
