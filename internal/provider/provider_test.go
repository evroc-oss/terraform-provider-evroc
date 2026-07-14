// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/evroc-oss/evroc-go-sdk/config"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var testAccProvider *schema.Provider
var testAccProviderFactories map[string]func() (*schema.Provider, error)

func init() {
	testAccProvider = New("dev")()
	testAccProviderFactories = map[string]func() (*schema.Provider, error){
		"evroc": func() (*schema.Provider, error) {
			return testAccProvider, nil
		},
	}
}

func TestProvider(t *testing.T) {
	if err := New("dev")().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

// providerTestResourceData creates ResourceData from the provider schema for testing.
func providerTestResourceData() *schema.ResourceData {
	p := New("test")()
	res := &schema.Resource{Schema: p.Schema}
	return res.TestResourceData()
}

func TestConfigureNoAuth(t *testing.T) {
	// Isolate from any existing CLI config by pointing HOME to a temp dir
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", t.TempDir())
	defer os.Setenv("HOME", origHome)

	p := New("test")()
	d := providerTestResourceData()

	configureFn := configure("test", p)
	_, diags := configureFn(context.Background(), d)
	if !diags.HasError() {
		t.Fatal("expected error when no auth is provided")
	}
	if !strings.Contains(diags[0].Summary, "no credentials found") {
		t.Errorf("expected 'no credentials found' error, got: %s", diags[0].Summary)
	}
}

func TestConfigureBothAuth(t *testing.T) {
	p := New("test")()
	d := providerTestResourceData()

	d.Set("token", "some-token")
	d.Set("username", "user")
	d.Set("password", "pass")

	configureFn := configure("test", p)
	_, diags := configureFn(context.Background(), d)
	if !diags.HasError() {
		t.Fatal("expected error when both token and username/password are provided")
	}
	if !strings.Contains(diags[0].Summary, "only one authentication method") {
		t.Errorf("expected 'only one authentication method' error, got: %s", diags[0].Summary)
	}
}

func testAccPreCheck(t *testing.T) {
	// Check for config file, (token OR refresh_token), OR (username AND password)
	// This matches the provider's credential chain
	hasConfigFile := os.Getenv("EVROC_CONFIG_FILE") != ""
	hasToken := os.Getenv("EVROC_TOKEN") != "" || os.Getenv("EVROC_REFRESH_TOKEN") != ""
	hasUsernamePassword := os.Getenv("EVROC_USERNAME") != "" && os.Getenv("EVROC_PASSWORD") != ""

	if !hasConfigFile && !hasToken && !hasUsernamePassword {
		// Check if ~/.evroc/config.yaml exists as fallback
		if path, err := config.DefaultCLIConfigPath(); err != nil || !fileExists(path) {
			t.Fatal("Authentication required: set EVROC_CONFIG_FILE, EVROC_TOKEN, EVROC_USERNAME+EVROC_PASSWORD, or configure ~/.evroc/config.yaml")
		}
	}

	// When using config file or CLI config, project comes from the file
	if !hasConfigFile {
		if v := os.Getenv("EVROC_PROJECT"); v == "" {
			if path, err := config.DefaultCLIConfigPath(); err != nil || !fileExists(path) {
				t.Fatal("EVROC_PROJECT must be set for acceptance tests (or use config file)")
			}
		}
	}
}

func TestConfigureWithConfigFile(t *testing.T) {
	// Create a temp config file with test values
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yaml"
	os.WriteFile(configPath, []byte(`
auth:
  token: "test-token"
  refresh_token: "test-refresh"
context:
  project: "test-project"
  region: "se-sto"
  organization: "test-org"
`), 0600)

	p := New("test")()
	d := providerTestResourceData()
	d.Set("config_file", configPath)

	configureFn := configure("test", p)
	meta, diags := configureFn(context.Background(), d)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	cfg := meta.(*ProviderConfig)
	if cfg.Project != "test-project" {
		t.Errorf("expected project test-project, got %s", cfg.Project)
	}
	if cfg.Region != "se-sto" {
		t.Errorf("expected region se-sto, got %s", cfg.Region)
	}
}

func TestConfigureWithConfigFileNotFound(t *testing.T) {
	p := New("test")()
	d := providerTestResourceData()
	d.Set("config_file", "/nonexistent/path/config.yaml")

	configureFn := configure("test", p)
	_, diags := configureFn(context.Background(), d)
	if !diags.HasError() {
		t.Fatal("expected error for nonexistent config file")
	}
	if !strings.Contains(diags[0].Summary, "failed to load config from file") {
		t.Errorf("expected 'failed to load config' error, got: %s", diags[0].Summary)
	}
}

func TestConfigureWithToken(t *testing.T) {
	p := New("test")()
	d := providerTestResourceData()

	d.Set("token", "test-token")
	d.Set("refresh_token", "test-refresh")
	d.Set("project", "my-project")
	d.Set("region", "se-sto")

	configureFn := configure("test", p)
	meta, diags := configureFn(context.Background(), d)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	cfg := meta.(*ProviderConfig)
	if cfg.Project != "my-project" {
		t.Errorf("expected project my-project, got %s", cfg.Project)
	}
	if cfg.Region != "se-sto" {
		t.Errorf("expected region se-sto, got %s", cfg.Region)
	}
}

func TestConfigureWithUsernamePassword(t *testing.T) {
	// Username/password auth hits the real auth endpoint, so we expect an auth error.
	// This test verifies the code path reaches client creation (not an input validation error).
	p := New("test")()
	d := providerTestResourceData()

	d.Set("username", "testuser")
	d.Set("password", "testpass")
	d.Set("project", "my-project")
	d.Set("region", "se-sto")

	configureFn := configure("test", p)
	_, diags := configureFn(context.Background(), d)
	if !diags.HasError() {
		t.Fatal("expected error from auth endpoint with fake credentials")
	}
	// Verify we reached client creation (not input validation)
	if strings.Contains(diags[0].Summary, "only one authentication method") || strings.Contains(diags[0].Summary, "no credentials found") {
		t.Errorf("expected auth error, got input validation error: %s", diags[0].Summary)
	}
}

func TestConfigureSDKCredentialChainEnvVars(t *testing.T) {
	// Test Strategy 3: SDK credential chain via environment variables.
	// config.Load() checks env vars first before falling back to CLI config.
	t.Setenv("EVROC_TOKEN", "env-token")
	t.Setenv("EVROC_REFRESH_TOKEN", "env-refresh")
	t.Setenv("EVROC_PROJECT", "env-project")
	t.Setenv("EVROC_REGION", "se-sto")
	t.Setenv("EVROC_ORGANIZATION", "env-org")

	p := New("test")()
	d := providerTestResourceData()
	// No explicit provider attributes — should fall through to SDK chain

	configureFn := configure("test", p)
	meta, diags := configureFn(context.Background(), d)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	cfg := meta.(*ProviderConfig)
	if cfg.Project != "env-project" {
		t.Errorf("expected project env-project, got %s", cfg.Project)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
