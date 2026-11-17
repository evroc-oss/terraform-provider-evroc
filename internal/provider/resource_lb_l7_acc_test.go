// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	_ "embed"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"text/template"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	lbL7Hostname           = "l7.example.test"
	lbL7OtherHostname      = "other.example.test"
	lbL7APIResponse        = "tf-l7-api\n"
	lbL7FallbackResponse   = "tf-l7-fallback\n"
	lbL7Address            = "evroc_loadbalancer.test"
	lbL7RouteAddress       = "evroc_lb_http_route.test"
	lbL7CertificateAddress = "evroc_lb_certificate.test"
)

//go:embed testdata/lb_l7/main.tf.tmpl
var lbL7TerraformTemplate string

//go:embed testdata/lb_l7/cloud-init.yaml
var lbL7CloudInit string

func TestAccEvrocLBL7_HTTPAndHTTPS(t *testing.T) {
	if os.Getenv(resource.EnvTfAcc) == "" {
		t.Skip("set TF_ACC to run the live load balancer workflow")
	}

	name := fmt.Sprintf("tf-test-l7-%d", time.Now().UnixNano())
	initial := newLBL7Certificate(t, nil, 1)
	renewed := newLBL7Certificate(t, initial.key, 2)
	rotated := newLBL7Certificate(t, nil, 3)
	external := newLBL7Certificate(t, nil, 4)
	initialConfig := lbL7Configuration(t, name, initial, lbL7Hostname, "v2")
	renewedConfig := lbL7Configuration(t, name, renewed, lbL7Hostname, "v2")
	rotatedConfig := lbL7Configuration(t, name, rotated, lbL7Hostname, "v2")
	updatedConfig := lbL7Configuration(t, name, rotated, lbL7Hostname, "v3")
	unrestrictedConfig := lbL7Configuration(t, name, rotated, "", "v3")
	identities := make(map[string]string)
	check := func(certificate lbL7Certificate, hostname, header string) resource.TestCheckFunc {
		return resource.ComposeTestCheckFunc(
			checkLBL7State(certificate, hostname, header),
			checkLBL7Identities(identities),
			checkLBL7Traffic(t, certificate, hostname, header),
		)
	}

	resource.Test(t, resource.TestCase{
		PreventPostDestroyRefresh: true,
		PreCheck:                  func() { testAccPreCheck(t) }, ProviderFactories: testAccProviderFactories,
		CheckDestroy: checkLBL7Destroy,
		Steps: []resource.TestStep{
			// Observe routing and the served certificate through both public listeners.
			{Config: initialConfig, Check: check(initial, lbL7Hostname, "v2")},
			{ResourceName: lbL7RouteAddress, ImportState: true, ImportStateVerify: true},
			{ResourceName: lbL7Address, ImportState: true, ImportStateVerify: true},
			{ResourceName: lbL7CertificateAddress, ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"certificate_pem", "private_key_pem"}},
			// Reapplying the same inputs must preserve an empty plan and live traffic.
			{Config: initialConfig, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}, Check: check(initial, lbL7Hostname, "v2")},
			// Renewal changes only the leaf certificate; rotation changes the key too.
			{Config: renewedConfig, ConfigPlanChecks: lbL7UpdatePlan(lbL7CertificateAddress), Check: check(renewed, lbL7Hostname, "v2")},
			{Config: rotatedConfig, ConfigPlanChecks: lbL7UpdatePlan(lbL7CertificateAddress), Check: check(rotated, lbL7Hostname, "v2")},
			// Restore the configured pair after an external upload changes the served leaf.
			{PreConfig: replaceLBL7Certificate(t, name, external), Config: rotatedConfig, ConfigPlanChecks: lbL7UpdatePlan(lbL7CertificateAddress), Check: check(rotated, lbL7Hostname, "v2")},
			// A changed header match must change the backend chosen for live requests.
			{Config: updatedConfig, ConfigPlanChecks: lbL7UpdatePlan(lbL7RouteAddress), Check: check(rotated, lbL7Hostname, "v3")},
			{Config: unrestrictedConfig, ConfigPlanChecks: lbL7UpdatePlan(lbL7RouteAddress), Check: check(rotated, "", "v3")},
		},
	})
}

func lbL7UpdatePlan(address string) resource.ConfigPlanChecks {
	return resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)}}
}

type lbL7Certificate struct {
	key                             *ecdsa.PrivateKey
	leaf                            *x509.Certificate
	certificateFile, privateKeyFile string
}

func newLBL7Certificate(t *testing.T, key *ecdsa.PrivateKey, serial int64) lbL7Certificate {
	t.Helper()
	if key == nil {
		var err error
		key, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now()
	certificate := &x509.Certificate{SerialNumber: big.NewInt(serial), Subject: pkix.Name{CommonName: lbL7Hostname}, DNSNames: []string{lbL7Hostname, lbL7OtherHostname}, NotBefore: now.Add(-time.Hour), NotAfter: now.Add(24 * time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(rand.Reader, certificate, certificate, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	result := lbL7Certificate{key: key, leaf: leaf, certificateFile: filepath.Join(dir, "certificate.pem"), privateKeyFile: filepath.Join(dir, "private-key.pem")}
	if err := os.WriteFile(result.certificateFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(result.privateKeyFile, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0600); err != nil {
		t.Fatal(err)
	}
	return result
}

func lbL7Configuration(t *testing.T, name string, certificate lbL7Certificate, hostname, header string) string {
	t.Helper()
	parsed, err := template.New("loadbalancer").Parse(lbL7TerraformTemplate)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	data := struct{ Name, CertificateFile, PrivateKeyFile, Hostname, HeaderValue, CloudInit string }{name, certificate.certificateFile, certificate.privateKeyFile, hostname, header, lbL7CloudInit}
	if err := parsed.Execute(&out, data); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func checkLBL7State(certificate lbL7Certificate, hostname, header string) resource.TestCheckFunc {
	fingerprint := fmt.Sprintf("%x", sha256.Sum256(certificate.leaf.Raw))
	publicFingerprint := fmt.Sprintf("%x", sha256.Sum256(certificate.leaf.RawSubjectPublicKeyInfo))
	hostCount := "1"
	if hostname == "" {
		hostCount = "0"
	}
	return resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttr(lbL7CertificateAddress, "ready", "true"),
		resource.TestCheckResourceAttr(lbL7CertificateAddress, "certificate_sha256_fingerprint", fingerprint),
		resource.TestCheckResourceAttr(lbL7CertificateAddress, "public_key_sha256_fingerprint", publicFingerprint),
		resource.TestCheckResourceAttr("data.evroc_lb_certificate.test", "certificate_sha256_fingerprint", fingerprint),
		resource.TestCheckResourceAttr("data.evroc_lb_certificate.test", "public_key_sha256_fingerprint", publicFingerprint),
		resource.TestCheckResourceAttr("data.evroc_lb_certificate.test", "ready", "true"),
		resource.TestCheckResourceAttrPair(lbL7CertificateAddress, "fqid", "data.evroc_lb_certificate.test", "fqid"),
		resource.TestCheckResourceAttrPair(lbL7RouteAddress, "fqid", "data.evroc_lb_http_route.test", "fqid"),
		resource.TestCheckResourceAttr(lbL7RouteAddress, "hostnames.#", hostCount),
		resource.TestCheckResourceAttr("data.evroc_lb_http_route.test", "hostnames.#", hostCount),
		resource.TestCheckTypeSetElemNestedAttrs("data.evroc_lb_http_route.test", "rule.0.match.0.header.*", map[string]string{"name": "x-version", "value": header}),
		resource.TestCheckResourceAttrPair(lbL7Address, "public_ipv4_address", "data.evroc_loadbalancer.test", "public_ipv4_address"),
	)
}

func checkLBL7Identities(identities map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for address, attribute := range map[string]string{lbL7Address: "lb_id", lbL7RouteAddress: "route_id", lbL7CertificateAddress: "certificate_id"} {
			state, ok := s.RootModule().Resources[address]
			if !ok {
				return fmt.Errorf("resource %s is missing", address)
			}
			value := state.Primary.Attributes[attribute]
			if value == "" {
				return fmt.Errorf("resource %s has no UUID", address)
			}
			if previous, ok := identities[address]; ok && previous != value {
				return fmt.Errorf("resource %s was replaced", address)
			}
			identities[address] = value
		}
		certificateRef := s.RootModule().Resources[lbL7CertificateAddress].Primary.Attributes["fqid"]
		routeRef := s.RootModule().Resources[lbL7RouteAddress].Primary.Attributes["fqid"]
		return resource.ComposeTestCheckFunc(
			resource.TestCheckTypeSetElemAttr(lbL7Address, "listener.*.route_refs.*", routeRef),
			resource.TestCheckTypeSetElemNestedAttrs(lbL7Address, "listener.*", map[string]string{"name": "http", "protocol": "HTTP", "port": "80"}),
			resource.TestCheckTypeSetElemNestedAttrs(lbL7Address, "listener.*", map[string]string{"name": "https", "protocol": "HTTPS", "port": "443", "tls.0.certificate_ref": certificateRef}),
		)(s)
	}
}

type lbL7Probe struct {
	name, scheme, hostname, method, path, header, wantBody string
	reject                                                 bool
}

func lbL7Probes(hostname, header string) []lbL7Probe {
	oldHeader := "v3"
	if header == "v3" {
		oldHeader = "v2"
	}
	requests := []lbL7Probe{
		{name: "all match fields", path: "/api/status?preview=true", header: header, wantBody: lbL7APIResponse},
		{name: "prefix root", path: "/api?preview=true", header: header, wantBody: lbL7APIResponse},
		{name: "alternative OR match", path: "/alternative", wantBody: lbL7APIResponse},
		{name: "omitted method and path", method: http.MethodPost, path: "/default-path", header: "default-path", wantBody: lbL7APIResponse},
		{name: "exact path excludes descendants", path: "/alternative/child", wantBody: lbL7FallbackResponse},
		{name: "missing header", path: "/api/status?preview=true", wantBody: lbL7FallbackResponse},
		{name: "old or wrong header", path: "/api/status?preview=true", header: oldHeader, wantBody: lbL7FallbackResponse},
		{name: "missing query", path: "/api/status", header: header, wantBody: lbL7FallbackResponse},
		{name: "wrong query", path: "/api/status?preview=false", header: header, wantBody: lbL7FallbackResponse},
		{name: "wrong method", method: http.MethodPost, path: "/api/status?preview=true", header: header, wantBody: lbL7FallbackResponse},
		{name: "prefix segment boundary", path: "/apiv2/status?preview=true", header: header, wantBody: lbL7FallbackResponse},
		{name: "fallback", path: "/", wantBody: lbL7FallbackResponse},
	}
	probes := make([]lbL7Probe, 0, 2*len(requests)+2)
	for _, scheme := range []string{"http", "https"} {
		for _, request := range requests {
			request.scheme = scheme
			request.hostname = lbL7Hostname
			if request.method == "" {
				request.method = http.MethodGet
			}
			probes = append(probes, request)
		}
		probes = append(probes, lbL7Probe{name: "other hostname", scheme: scheme, hostname: lbL7OtherHostname, method: http.MethodGet, path: "/alternative", wantBody: lbL7APIResponse, reject: hostname != ""})
	}
	return probes
}

func checkLBL7Traffic(t *testing.T, certificate lbL7Certificate, hostname, header string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ip := s.RootModule().Resources[lbL7Address].Primary.Attributes["public_ipv4_address"]
		if net.ParseIP(ip) == nil {
			return fmt.Errorf("load balancer has no public IP")
		}
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
		defer cancel()
		return waitLBL7Traffic(ctx, ip, certificate.leaf, lbL7Probes(hostname, header))
	}
}

func waitLBL7Traffic(ctx context.Context, ip string, certificate *x509.Certificate, probes []lbL7Probe) error {
	var last error
	consecutive := 0
	for {
		last = probeLBL7Traffic(ctx, ip, certificate, probes)
		if last == nil {
			consecutive++
			if consecutive == 2 {
				return nil
			}
		} else {
			consecutive = 0
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("load balancer traffic did not converge: %w", errors.Join(ctx.Err(), last))
		case <-time.After(2 * time.Second):
		}
	}
}

func probeLBL7Traffic(ctx context.Context, ip string, certificate *x509.Certificate, probes []lbL7Probe) error {
	roots := x509.NewCertPool()
	roots.AddCert(certificate)
	transport := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}, DisableKeepAlives: true, DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		_, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		return (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, network, net.JoinHostPort(ip, port))
	}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	for _, probe := range probes {
		request, err := http.NewRequestWithContext(ctx, probe.method, probe.scheme+"://"+probe.hostname+probe.path, nil)
		if err != nil {
			return err
		}
		if probe.header != "" {
			request.Header.Set("X-Version", probe.header)
		}
		response, err := client.Do(request)
		if err != nil {
			return fmt.Errorf("%s %s: %w", probe.scheme, probe.name, err)
		}
		err = checkLBL7Response(response, certificate, probe)
		response.Body.Close()
		if err != nil {
			return fmt.Errorf("%s %s: %w", probe.scheme, probe.name, err)
		}
	}
	return nil
}

func checkLBL7Response(response *http.Response, certificate *x509.Certificate, probe lbL7Probe) error {
	body, err := io.ReadAll(io.LimitReader(response.Body, 4096))
	if err != nil {
		return err
	}
	if probe.scheme == "https" {
		if response.TLS == nil || len(response.TLS.PeerCertificates) == 0 {
			return fmt.Errorf("HTTPS response has no peer certificate")
		}
		if !bytes.Equal(response.TLS.PeerCertificates[0].Raw, certificate.Raw) {
			return fmt.Errorf("HTTPS is serving a different leaf certificate")
		}
	}
	if probe.reject {
		if response.StatusCode < 400 || string(body) == lbL7APIResponse || string(body) == lbL7FallbackResponse {
			return fmt.Errorf("unmatched hostname reached a backend (status %d)", response.StatusCode)
		}
		return nil
	}
	if response.StatusCode != http.StatusOK || string(body) != probe.wantBody {
		return fmt.Errorf("status %d, body %q; want status 200 and body %q", response.StatusCode, body, probe.wantBody)
	}
	return nil
}

func replaceLBL7Certificate(t *testing.T, name string, certificate lbL7Certificate) func() {
	return func() {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
		defer cancel()
		client := testAccProvider.Meta().(*ProviderConfig).Client.LoadBalancer()
		lb, err := client.LoadBalancers().Get(ctx, name+"-lb")
		if err != nil {
			t.Fatal(err)
		}
		if lb.Status.Networking == nil || lb.Status.Networking.PublicIPv4Address == nil {
			t.Fatal("load balancer has no public IP before external certificate upload")
		}
		certificatePEM, err := os.ReadFile(certificate.certificateFile)
		if err != nil {
			t.Fatal(err)
		}
		keyPEM, err := os.ReadFile(certificate.privateKeyFile)
		if err != nil {
			t.Fatal(err)
		}

		if err := client.LoadBalancerCertificates().UploadContent(ctx, name+"-cert", string(certificatePEM), string(keyPEM)); err != nil {
			t.Fatalf("external certificate upload: %v", err)
		}
		if err := waitLBL7Traffic(ctx, *lb.Status.Networking.PublicIPv4Address, certificate.leaf, lbL7Probes(lbL7Hostname, "v2")); err != nil {
			t.Fatal(err)
		}
	}
}

func checkLBL7Destroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*ProviderConfig).Client
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	for _, state := range s.RootModule().Resources {
		var err error
		switch state.Type {
		case "evroc_loadbalancer":
			_, err = client.LoadBalancer().LoadBalancers().Get(ctx, state.Primary.ID)
		case "evroc_lb_certificate":
			_, err = client.LoadBalancer().LoadBalancerCertificates().Get(ctx, state.Primary.ID)
		case "evroc_lb_http_route":
			_, err = client.LoadBalancer().HTTPRoutes().Get(ctx, state.Primary.ID)
		case "evroc_lb_backend_service":
			_, err = client.LoadBalancer().BackendServices().Get(ctx, state.Primary.ID)
		case "evroc_lb_backend_pool":
			_, err = client.LoadBalancer().BackendPools().Get(ctx, state.Primary.ID)
		case "evroc_public_ip":
			_, err = client.Networking().PublicIPs().Get(ctx, state.Primary.ID)
		case "evroc_virtual_machine":
			_, err = client.Compute().VirtualMachines().Get(ctx, state.Primary.ID)
		case "evroc_disk":
			_, err = client.Compute().Disks().Get(ctx, state.Primary.ID)
		case "evroc_security_group":
			_, err = client.Networking().SecurityGroups().Get(ctx, state.Primary.ID)
		default:
			continue
		}
		if isNotFoundError(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("checking deletion of %s %s: %w", state.Type, state.Primary.ID, err)
		}
		return fmt.Errorf("%s %s still exists", state.Type, state.Primary.ID)
	}
	return nil
}

func TestLBL7ProbeRejectsWrongBackendAndCertificate(t *testing.T) {
	certificate := newLBL7Certificate(t, nil, 1)
	pair, err := tls.LoadX509KeyPair(certificate.certificateFile, certificate.privateKeyFile)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(lbL7APIResponse))
	}))
	server.TLS = &tls.Config{Certificates: []tls.Certificate{pair}, MinVersion: tls.VersionTLS12}
	server.StartTLS()
	defer server.Close()
	host, port, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	renewed := newLBL7Certificate(t, certificate.key, 2)

	for _, scenario := range []struct {
		name, hostname, body string
		leaf                 *x509.Certificate
		wantError            bool
	}{
		{"correct backend and leaf", lbL7Hostname, lbL7APIResponse, certificate.leaf, false},
		{"wrong backend", lbL7Hostname, lbL7FallbackResponse, certificate.leaf, true},
		{"different leaf with same key", lbL7Hostname, lbL7APIResponse, renewed.leaf, true},
		{"untrusted hostname", "uncovered.example.test", lbL7APIResponse, certificate.leaf, true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			probe := lbL7Probe{scheme: "https", hostname: net.JoinHostPort(scenario.hostname, port), method: http.MethodGet, path: "/", wantBody: scenario.body}
			err := probeLBL7Traffic(t.Context(), host, scenario.leaf, []lbL7Probe{probe})
			if (err != nil) != scenario.wantError {
				t.Fatalf("probe error = %v, want error %v", err, scenario.wantError)
			}
		})
	}
}
