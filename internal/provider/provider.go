// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	evroc "github.com/evroc-oss/evroc-go-sdk"
	"github.com/evroc-oss/evroc-go-sdk/config"
)

// New returns a new terraform provider
func New(version string) func() *schema.Provider {
	return func() *schema.Provider {
		p := &schema.Provider{
			Schema: map[string]*schema.Schema{
				"config_file": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("EVROC_CONFIG_FILE", nil),
					Description: "Path to evroc SDK config YAML file. Can also be set via EVROC_CONFIG_FILE environment variable. " +
						"When set, authentication and context are loaded from this file. " +
						"If not set, falls back to ~/.evroc/config.yaml or explicit provider attributes.",
				},
				"token": {
					Type:        schema.TypeString,
					Optional:    true,
					Sensitive:   true,
					DefaultFunc: schema.EnvDefaultFunc("EVROC_TOKEN", nil),
					Description: "evroc API token for authentication. Can also be set via EVROC_TOKEN environment variable. Use this OR username/password.",
				},
				"refresh_token": {
					Type:        schema.TypeString,
					Optional:    true,
					Sensitive:   true,
					DefaultFunc: schema.EnvDefaultFunc("EVROC_REFRESH_TOKEN", nil),
					Description: "evroc API refresh token for automatic token renewal. Can also be set via EVROC_REFRESH_TOKEN environment variable.",
				},
				"username": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("EVROC_USERNAME", nil),
					Description: "evroc API username for authentication. Can also be set via EVROC_USERNAME environment variable. Use this with password OR use token.",
				},
				"password": {
					Type:        schema.TypeString,
					Optional:    true,
					Sensitive:   true,
					DefaultFunc: schema.EnvDefaultFunc("EVROC_PASSWORD", nil),
					Description: "evroc API password for authentication. Can also be set via EVROC_PASSWORD environment variable. Use this with username OR use token.",
				},
				"organization": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("EVROC_ORGANIZATION", nil),
					Description: "evroc organization ID. Can also be set via EVROC_ORGANIZATION environment variable.",
				},
				"project": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("EVROC_PROJECT", nil),
					Description: "Default evroc project ID. Can also be set via EVROC_PROJECT environment variable. Can be overridden per resource.",
				},
				"region": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("EVROC_REGION", "se-sto"),
					Description: "Default evroc region (e.g., se-sto). Can also be set via EVROC_REGION environment variable.",
				},
				"api_endpoint": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("EVROC_API_ENDPOINT", "https://api.cloud.evroc.com"),
					Description: "evroc API endpoint. Can also be set via EVROC_API_ENDPOINT environment variable.",
				},
			},

			ResourcesMap: map[string]*schema.Resource{
				"evroc_disk":                    resourceDisk(),
				"evroc_public_ip":               resourcePublicIP(),
				"evroc_virtual_machine":         resourceVirtualMachine(),
				"evroc_security_group":          resourceSecurityGroup(),
				"evroc_placement_group":         resourcePlacementGroup(),
				"evroc_hotswap_disk_attachment": resourceHotswapDiskAttachment(),
				"evroc_bucket":                  resourceBucket(),
				"evroc_bucket_service_account":  resourceBucketServiceAccount(),
				"evroc_project":                 resourceProject(),
				"evroc_think_instance":          resourceThinkInstance(),
				"evroc_think_api_key":           resourceThinkAPIKey(),
				"evroc_permission_set":          resourcePermissionSet(),
			},

			DataSourcesMap: map[string]*schema.Resource{
				"evroc_disk":                    dataSourceDisk(),
				"evroc_public_ip":               dataSourcePublicIP(),
				"evroc_virtual_machine":         dataSourceVirtualMachine(),
				"evroc_security_group":          dataSourceSecurityGroup(),
				"evroc_placement_group":         dataSourcePlacementGroup(),
				"evroc_hotswap_disk_attachment": dataSourceHotswapDiskAttachment(),
				"evroc_bucket":                  dataSourceBucket(),
				"evroc_bucket_service_account":  dataSourceBucketServiceAccount(),
				"evroc_disk_images":             dataSourceDiskImages(),
				"evroc_compute_profiles":        dataSourceComputeProfiles(),
				"evroc_project":                 dataSourceProject(),
				"evroc_think_instance":          dataSourceThinkInstance(),
				"evroc_think_models":            dataSourceThinkModels(),
				"evroc_think_sizes":             dataSourceThinkSizes(),
				"evroc_permission_set":          dataSourcePermissionSet(),
			},
		}

		p.ConfigureContextFunc = configure(version, p)

		return p
	}
}

// ProviderConfig contains the configured evroc client
type ProviderConfig struct {
	Client     *evroc.Client
	Project    string
	Region     string
	baseConfig *config.Config
	mu         sync.Mutex
	clients    map[string]*evroc.Client
}

// ClientForProject returns a client configured for the given project.
// If project matches the provider-level project (or is empty), the default client is returned.
// Otherwise a new client is created and cached for that project.
func (p *ProviderConfig) ClientForProject(project string) (*evroc.Client, error) {
	if project == "" || project == p.Project {
		return p.Client, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if client, ok := p.clients[project]; ok {
		return client, nil
	}

	auth := p.baseConfig.Auth
	auth.Token = ""

	cfg := &config.Config{
		Auth: auth,
		API:  p.baseConfig.API,
		Context: config.ContextConfig{
			Organization: p.baseConfig.Context.Organization,
			Project:      project,
			Region:       p.baseConfig.Context.Region,
		},
	}
	cfg.SetDefaults()

	// Use background context so the client outlives any single request
	client, err := evroc.New(context.Background(), *cfg) //nolint:contextcheck // intentional: per-project client outlives request
	if err != nil {
		return nil, err
	}

	p.clients[project] = client
	return client, nil
}

// WaitForProject polls until the project appears in the project list.
func (p *ProviderConfig) WaitForProject(ctx context.Context, project string, timeout time.Duration) error {
	if p.baseConfig == nil {
		return nil
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		list, err := p.Client.IAM().Projects().List(ctx)
		if err == nil {
			for _, proj := range list.Items {
				if proj.Metadata.Id == project {
					return nil
				}
			}
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("timed out waiting for project %s to become available", project)
}

// WaitForProjectDeletion polls until the project no longer appears in the project list.
func (p *ProviderConfig) WaitForProjectDeletion(ctx context.Context, project string, timeout time.Duration) error {
	if p.baseConfig == nil {
		return nil
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		list, err := p.Client.IAM().Projects().List(ctx)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		found := false
		for _, proj := range list.Items {
			if proj.Metadata.Id == project {
				found = true
				break
			}
		}
		if !found {
			return nil
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("timed out waiting for project %s to be deleted", project)
}

// resolveProject returns the project to use for a resource.
// It checks the resource-level "project" attribute first, falling back to the provider default.
func resolveProject(d *schema.ResourceData, config *ProviderConfig) string {
	if v, ok := d.GetOk("project"); ok {
		if project := v.(string); project != "" {
			return project
		}
	}
	return config.Project
}

// resolveClient returns the SDK client for the resolved project.
func resolveClient(d *schema.ResourceData, config *ProviderConfig) (*evroc.Client, diag.Diagnostics) { //nolint:contextcheck // client lifetime outlives request
	project := resolveProject(d, config)
	if project == "" {
		return nil, diag.Errorf("project is required: set it on the resource, in the provider block, or in ~/.evroc/config.yaml")
	}

	client, err := config.ClientForProject(project)
	if err != nil {
		return nil, diag.Errorf("failed to create client for project %s: %s", project, err)
	}

	return client, nil
}

func configure(version string, p *schema.Provider) func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
	return func(_ context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		configFile := d.Get("config_file").(string)

		// Use a background context for client initialization so the client's
		// lifetime (and token refresh) is not tied to a single request context.
		// Each resource CRUD operation passes its own context to SDK methods.
		clientCtx := context.Background() //nolint:contextcheck // intentional: client outlives the configure request

		// Strategy 1: Explicit config file (like cluster-api's mounted config)
		if configFile != "" {
			cfg, err := config.LoadFromFile(configFile)
			if err != nil {
				return nil, diag.Errorf("failed to load config from file %s: %s", configFile, err)
			}
			cfg.SetDefaults()
			client, err := evroc.New(clientCtx, *cfg) //nolint:contextcheck // intentional: client outlives the configure request
			if err != nil {
				return nil, diag.Errorf("failed to create evroc client from config file %s: %s", configFile, err)
			}
			return &ProviderConfig{
				Client:     client,
				Project:    cfg.Context.Project,
				Region:     cfg.Context.Region,
				baseConfig: cfg,
				clients:    make(map[string]*evroc.Client),
			}, nil
		}

		// Strategy 2: Explicit provider attributes / env vars
		token := d.Get("token").(string)
		refreshToken := d.Get("refresh_token").(string)
		username := d.Get("username").(string)
		password := d.Get("password").(string)
		organization := d.Get("organization").(string)
		project := d.Get("project").(string)
		region := d.Get("region").(string)
		apiEndpoint := d.Get("api_endpoint").(string)

		hasToken := token != "" || refreshToken != ""
		hasUsernamePassword := username != "" && password != ""

		if hasToken && hasUsernamePassword {
			return nil, diag.Errorf("provide either 'token' OR 'username'+'password', not both")
		}

		if hasToken || hasUsernamePassword {
			cfg := &config.Config{
				Auth: config.AuthConfig{
					Token:        token,
					RefreshToken: refreshToken,
					Username:     username,
					Password:     password,
				},
				API: config.APIConfig{
					BaseURL: apiEndpoint,
				},
				Context: config.ContextConfig{
					Organization: organization,
					Project:      project,
					Region:       region,
				},
			}
			cfg.SetDefaults()

			client, err := evroc.New(clientCtx, *cfg) //nolint:contextcheck // intentional: client outlives the configure request
			if err != nil {
				return nil, diag.Errorf("failed to create evroc client: %s", err)
			}
			return &ProviderConfig{
				Client:     client,
				Project:    project,
				Region:     region,
				baseConfig: cfg,
				clients:    make(map[string]*evroc.Client),
			}, nil
		}

		// Strategy 3: SDK credential chain (env vars -> ~/.evroc/config.yaml)
		cfg, err := config.Load()
		if err != nil {
			return nil, diag.Errorf("no credentials found: provide 'config_file', explicit credentials, or configure ~/.evroc/config.yaml: %s", err)
		}
		client, err := evroc.New(clientCtx, *cfg) //nolint:contextcheck // intentional: client outlives the configure request
		if err != nil {
			return nil, diag.Errorf("failed to create evroc client: %s", err)
		}
		return &ProviderConfig{
			Client:     client,
			Project:    cfg.Context.Project,
			Region:     cfg.Context.Region,
			baseConfig: cfg,
			clients:    make(map[string]*evroc.Client),
		}, nil
	}
}
