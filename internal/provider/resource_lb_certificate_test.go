// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"

	lbtypes "github.com/evroc-oss/evroc-go-sdk/types/loadbalancer"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

type certificateTestPair struct {
	certificate, key, fingerprint, publicKeyFingerprint string
}

func newCertificateTestPair(t *testing.T) certificateTestPair {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	requireLBNoError(t, err)
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "api.example.com"},
		DNSNames: []string{"api.example.com"}, NotBefore: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		NotAfter: time.Date(2036, 1, 1, 0, 0, 0, 0, time.UTC),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
	requireLBNoError(t, err)
	keyDER, err := x509.MarshalECPrivateKey(key)
	requireLBNoError(t, err)
	publicDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	requireLBNoError(t, err)
	// Fingerprints are defined by the API as SHA256 of leaf DER and public-key SPKI.
	return certificateTestPair{
		certificate: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
		key:         string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})),
		fingerprint: fmt.Sprintf("%x", sha256.Sum256(der)), publicKeyFingerprint: fmt.Sprintf("%x", sha256.Sum256(publicDER)),
	}
}

func TestLBCertificateUploadAndRead(t *testing.T) {
	ms, config := newLBTest(t)
	pair := newCertificateTestPair(t)
	d := schema.TestResourceDataRaw(t, resourceLBCertificate().Schema, map[string]interface{}{
		"name": "web", "certificate_pem": pair.certificate, "private_key_pem": pair.key,
	})
	expires := time.Date(2036, 1, 1, 0, 0, 0, 0, time.UTC)
	certificate := &lbtypes.Loadbalancercertificate{
		Metadata: mockLoadBalancer("web").Metadata,
		Status: lbtypes.LoadbalancercertificateStatus{
			Conditions:                   &[]lbtypes.LoadbalancercertificateStatusConditionsItem{{Type: "Ready", Status: lbtypes.True}},
			CertificateSha256Fingerprint: &pair.fingerprint, PublicKeySha256Fingerprint: &pair.publicKeyFingerprint, NotAfter: &expires,
		},
	}
	path := "/loadbalancer/v1alpha2/projects/test-project/regions/se-sto/loadBalancerCertificates"
	ms.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusCreated, certificate)
	})
	uploads := 0
	ms.mux.HandleFunc(path+"/web/content", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if !decodeLBRequest(t, w, r, &body) {
			return
		}
		if r.Method != http.MethodPost || body["certificate"] != pair.certificate || body["privateKey"] != pair.key {
			t.Error("upload did not contain the configured certificate/key pair")
		}
		uploads++
		respondJSON(w, http.StatusOK, map[string]interface{}{"data": map[string]string{"result": "success"}})
	})
	missing := false
	ms.mux.HandleFunc(path+"/web", func(w http.ResponseWriter, r *http.Request) {
		if missing {
			respondJSON(w, http.StatusNotFound, map[string]string{"reason": "not found"})
			return
		}
		respondJSON(w, http.StatusOK, certificate)
	})

	ctx := context.Background()
	requireLBSuccess(t, resourceLBCertificateCreate(ctx, d, config))
	if uploads != 1 {
		t.Fatalf("uploads = %d, want 1", uploads)
	}
	assertAttrs(t, d, map[string]interface{}{
		"certificate_pem": pair.certificate, "private_key_pem": pair.key,
		"certificate_sha256_fingerprint": pair.fingerprint, "public_key_sha256_fingerprint": pair.publicKeyFingerprint,
		"ready": true, "not_after": "2036-01-01T00:00:00Z",
	})
	imported := resourceLBCertificate().TestResourceData()
	imported.SetId("web")
	requireLBSuccess(t, resourceLBCertificateRead(ctx, imported, config))
	assertAttrs(t, imported, map[string]interface{}{"certificate_pem": "", "private_key_pem": "", "certificate_sha256_fingerprint": pair.fingerprint})
	dsRes := dataSourceLBCertificate()
	for _, secret := range []string{"private_key_pem", "certificate_pem"} {
		if _, ok := dsRes.Schema[secret]; ok {
			t.Errorf("data source exposes %s", secret)
		}
	}
	ds := schema.TestResourceDataRaw(t, dsRes.Schema, map[string]interface{}{"name": "web"})
	requireLBSuccess(t, dsRes.ReadContext(ctx, ds, config))
	assertAttrs(t, ds, map[string]interface{}{"certificate_sha256_fingerprint": pair.fingerprint, "ready": true})

	pair = newCertificateTestPair(t)
	requireLBNoError(t, d.Set("certificate_pem", pair.certificate))
	requireLBNoError(t, d.Set("private_key_pem", pair.key))
	requireLBSuccess(t, resourceLBCertificateUpdate(ctx, d, config))
	if uploads != 2 || d.Id() != "web" {
		t.Fatalf("rotation: uploads=%d ID=%q", uploads, d.Id())
	}

	certificate.Status = lbtypes.LoadbalancercertificateStatus{}
	requireLBSuccess(t, resourceLBCertificateRead(ctx, d, config))
	assertAttrs(t, d, map[string]interface{}{"certificate_sha256_fingerprint": "", "public_key_sha256_fingerprint": "", "ready": false, "not_after": ""})
	missing = true
	if diags := resourceLBCertificateRead(ctx, d, config); diags.HasError() || d.Id() != "" {
		t.Fatalf("missing resource read = %v, ID=%q", diags, d.Id())
	}
	if diags := dsRes.ReadContext(ctx, ds, config); !diags.HasError() {
		t.Fatal("missing data source must report an error")
	}
}

func TestLBCertificateReadinessRequiresCurrentContent(t *testing.T) {
	for _, tc := range []struct {
		name, leaf, key string
		ready           lbtypes.LoadbalancercertificateStatusConditionsItemStatus
	}{
		{"stale leaf", "old-leaf", "current-key", lbtypes.True},
		{"stale key", "current-leaf", "old-key", lbtypes.True},
		{"not ready", "current-leaf", "current-key", lbtypes.False},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ms, config := newLBTest(t)
			observations := 0
			ms.mux.HandleFunc("/loadbalancer/v1alpha2/projects/test-project/regions/se-sto/loadBalancerCertificates/web", func(w http.ResponseWriter, r *http.Request) {
				leaf, key, ready := tc.leaf, tc.key, tc.ready
				if observations > 0 {
					leaf, key, ready = "current-leaf", "current-key", lbtypes.True
				}
				observations++
				respondJSON(w, http.StatusOK, &lbtypes.Loadbalancercertificate{Metadata: mockLoadBalancer("web").Metadata, Status: lbtypes.LoadbalancercertificateStatus{
					CertificateSha256Fingerprint: &leaf, PublicKeySha256Fingerprint: &key,
					Conditions: &[]lbtypes.LoadbalancercertificateStatusConditionsItem{{Type: "Ready", Status: ready}},
				}})
			})
			requireLBNoError(t, waitLBCertificate(context.Background(), config.Client.LoadBalancer().LoadBalancerCertificates(), "web", "current-leaf", "current-key", 5*time.Second))
			if observations < 2 {
				t.Fatal("accepted stale or unready content")
			}
		})
	}
}

func TestLBCertificateUploadFailureKeepsID(t *testing.T) {
	ms, config := newLBTest(t)
	pair := newCertificateTestPair(t)
	d := schema.TestResourceDataRaw(t, resourceLBCertificate().Schema, map[string]interface{}{"name": "web", "certificate_pem": pair.certificate, "private_key_pem": pair.key})
	path := "/loadbalancer/v1alpha2/projects/test-project/regions/se-sto/loadBalancerCertificates"
	ms.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusCreated, &lbtypes.Loadbalancercertificate{Metadata: mockLoadBalancer("web").Metadata, Spec: lbtypes.LoadbalancercertificateSpec{}})
	})
	ms.mux.HandleFunc(path+"/web/content", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusBadRequest, map[string]string{"reason": "invalid certificate chain"})
	})
	if diags := resourceLBCertificateCreate(context.Background(), d, config); !diags.HasError() || d.Id() != "web" {
		t.Fatalf("upload failure: diagnostics=%v ID=%q; created resource must remain tracked", diags, d.Id())
	}
}

func TestLBCertificateValidation(t *testing.T) {
	pair := newCertificateTestPair(t)
	other := newCertificateTestPair(t)
	res := resourceLBCertificate()
	if !res.Schema["certificate_pem"].Sensitive || !res.Schema["private_key_pem"].Sensitive {
		t.Fatal("PEM inputs must be sensitive")
	}
	for _, tc := range []struct {
		name, certificate, key string
		wantError              bool
	}{
		{"matching pair", pair.certificate, pair.key, false},
		{"malformed certificate", "invalid-certificate-input", pair.key, true},
		{"malformed key", pair.certificate, "invalid-private-key-input", true},
		{"mismatched pair", pair.certificate, other.key, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := map[string]interface{}{"name": "web", "certificate_pem": tc.certificate, "private_key_pem": tc.key}
			message := checkLBValidation(t, res, cfg, tc.wantError)
			if strings.Contains(message, tc.certificate) || strings.Contains(message, tc.key) {
				t.Fatal("validation diagnostic exposed PEM input")
			}
		})
	}
}

func TestLBCertificateRepairsExternalContentChange(t *testing.T) {
	ms, config := newLBTest(t)
	res := resourceLBCertificate()
	pair := newCertificateTestPair(t)
	external := newCertificateTestPair(t)
	cfg := map[string]interface{}{"name": "web", "certificate_pem": pair.certificate, "private_key_pem": pair.key}
	d := schema.TestResourceDataRaw(t, res.Schema, cfg)
	d.SetId("web")
	certificate := &lbtypes.Loadbalancercertificate{
		Metadata: mockLoadBalancer("web").Metadata, Spec: lbtypes.LoadbalancercertificateSpec{},
		Status: lbtypes.LoadbalancercertificateStatus{
			Conditions:                   &[]lbtypes.LoadbalancercertificateStatusConditionsItem{{Type: "Ready", Status: lbtypes.True}},
			CertificateSha256Fingerprint: &external.fingerprint, PublicKeySha256Fingerprint: &external.publicKeyFingerprint,
		},
	}
	path := "/loadbalancer/v1alpha2/projects/test-project/regions/se-sto/loadBalancerCertificates/web"
	uploads := 0
	ms.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusOK, certificate)
	})
	ms.mux.HandleFunc(path+"/content", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if !decodeLBRequest(t, w, r, &body) {
			return
		}
		if body["certificate"] != pair.certificate || body["privateKey"] != pair.key {
			t.Error("repair did not upload the configured PEM pair")
		}
		uploads++
		certificate.Status.CertificateSha256Fingerprint = &pair.fingerprint
		certificate.Status.PublicKeySha256Fingerprint = &pair.publicKeyFingerprint
		respondJSON(w, http.StatusOK, map[string]interface{}{"data": map[string]string{"result": "success"}})
	})

	ctx := context.Background()
	requireLBSuccess(t, res.ReadContext(ctx, d, config))
	if d.Get("certificate_pem") != pair.certificate || d.Get("certificate_sha256_fingerprint") != external.fingerprint {
		t.Fatal("refresh must retain configured content and observe externally changed content")
	}
	state := d.State()
	diff, err := res.Diff(ctx, state, terraform.NewResourceConfigRaw(cfg), config)
	requireLBNoError(t, err)
	if diff == nil || diff.RequiresNew() {
		t.Fatalf("external content change must plan an in-place repair, got %#v", diff)
	}
	repaired, diags := res.Apply(ctx, state, diff, config)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if uploads != 1 || certificate.Status.CertificateSha256Fingerprint == nil || *certificate.Status.CertificateSha256Fingerprint != pair.fingerprint || certificate.Status.PublicKeySha256Fingerprint == nil || *certificate.Status.PublicKeySha256Fingerprint != pair.publicKeyFingerprint {
		t.Fatal("apply did not restore the configured content through the API")
	}
	if repaired.ID != "web" || repaired.Attributes["certificate_pem"] != pair.certificate || repaired.Attributes["private_key_pem"] != pair.key {
		t.Fatal("repair must retain the resource identity and configured PEM inputs")
	}
	if diff, err = res.Diff(ctx, repaired, terraform.NewResourceConfigRaw(cfg), config); err != nil {
		t.Fatal(err)
	} else if !diff.Empty() {
		t.Fatalf("repair did not converge: %#v", diff)
	}
}

func TestLBCertificateUnknownContentPlansUnknownFingerprints(t *testing.T) {
	pair := newCertificateTestPair(t)
	for _, unknown := range []string{"certificate_pem", "private_key_pem"} {
		t.Run(unknown, func(t *testing.T) {
			res := resourceLBCertificate()
			d := schema.TestResourceDataRaw(t, res.Schema, map[string]interface{}{
				"name": "web", "certificate_pem": pair.certificate, "private_key_pem": pair.key,
				"certificate_sha256_fingerprint": pair.fingerprint, "public_key_sha256_fingerprint": pair.publicKeyFingerprint,
			})
			d.SetId("web")
			values := map[string]cty.Value{
				"name": cty.StringVal("web"), "certificate_pem": cty.StringVal(pair.certificate), "private_key_pem": cty.StringVal(pair.key),
			}
			values[unknown] = cty.UnknownVal(cty.String)
			cfg := terraform.NewResourceConfigShimmed(cty.ObjectVal(values), res.CoreConfigSchema())
			diff, err := res.Diff(context.Background(), d.State(), cfg, nil)
			requireLBNoError(t, err)
			if diff == nil {
				t.Fatal("unknown certificate content must produce a plan")
			}
			for _, field := range []string{"certificate_sha256_fingerprint", "public_key_sha256_fingerprint"} {
				if attr := diff.Attributes[field]; attr == nil || !attr.NewComputed {
					t.Errorf("%s retained a known fingerprint for unknown content: %#v", field, attr)
				}
			}
		})
	}
}
