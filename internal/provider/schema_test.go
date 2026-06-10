// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// TestResourceSchemas verifies that all resource schemas are valid and have the expected fields.
func TestResourceSchemas(t *testing.T) {
	tests := []struct {
		name           string
		resource       *schema.Resource
		requiredFields []string
		optionalFields []string
		computedFields []string
	}{
		{
			name:           "evroc_disk",
			resource:       resourceDisk(),
			requiredFields: []string{"name", "zone"},
			optionalFields: []string{"size", "image", "region", "user_labels"},
			computedFields: []string{"size", "disk_id", "system_labels", "created_at", "fqid"},
		},
		{
			name:           "evroc_virtual_machine",
			resource:       resourceVirtualMachine(),
			requiredFields: []string{"name", "flavor", "boot_disk", "zone"},
			optionalFields: []string{"ssh_keys", "cloud_config_user_data", "public_ip", "security_groups", "region", "placement_group", "running", "user_labels", "data_disks"},
			computedFields: []string{"vm_id", "system_labels", "created_at", "status", "public_ipv4_address", "private_ipv4_address", "fqid"},
		},
		{
			name:           "evroc_public_ip",
			resource:       resourcePublicIP(),
			requiredFields: []string{"name"},
			optionalFields: []string{"region", "user_labels"},
			computedFields: []string{"ip_id", "system_labels", "ip_address", "created_at", "fqid"},
		},
		{
			name:           "evroc_security_group",
			resource:       resourceSecurityGroup(),
			requiredFields: []string{"name"},
			optionalFields: []string{"region", "rule", "user_labels"},
			computedFields: []string{"sg_id", "system_labels", "created_at", "fqid"},
		},
		{
			name:           "evroc_placement_group",
			resource:       resourcePlacementGroup(),
			requiredFields: []string{"name", "strategy"},
			optionalFields: []string{"zone", "region", "user_labels"},
			computedFields: []string{"pg_id", "system_labels", "created_at", "fqid"},
		},
		{
			name:           "evroc_hotswap_disk_attachment",
			resource:       resourceHotswapDiskAttachment(),
			requiredFields: []string{"name", "virtual_machine", "disk"},
			optionalFields: []string{"region", "user_labels"},
			computedFields: []string{"attachment_id", "system_labels", "serial", "created_at"},
		},
		{
			name:           "evroc_bucket",
			resource:       resourceBucket(),
			requiredFields: []string{"name"},
			optionalFields: []string{"object_retention_mode", "object_locking", "region", "user_labels"},
			computedFields: []string{"bucket_id", "system_labels", "created_at"},
		},
		{
			name:           "evroc_bucket_service_account",
			resource:       resourceBucketServiceAccount(),
			requiredFields: []string{"name", "buckets"},
			optionalFields: []string{"region", "user_labels"},
			computedFields: []string{"service_account_id", "system_labels", "credentials_secret", "created_at"},
		},
		{
			name:           "evroc_project",
			resource:       resourceProject(),
			requiredFields: []string{"name"},
			optionalFields: []string{"organization", "display_name", "user_labels"},
			computedFields: []string{"project_id", "created_at"},
		},
		{
			name:           "evroc_think_instance",
			resource:       resourceThinkInstance(),
			requiredFields: []string{"name", "model"},
			optionalFields: []string{"size", "region", "running", "user_labels", "project"},
			computedFields: []string{"instance_id", "system_labels", "created_at", "endpoint", "phase"},
		},
		{
			name:           "evroc_loadbalancer",
			resource:       resourceLoadBalancer(),
			requiredFields: []string{"name", "public_ip_ref", "listener"},
			optionalFields: []string{"region", "user_labels"},
			computedFields: []string{"lb_id", "system_labels", "created_at", "fqid"},
		},
		{
			name:           "evroc_lb_backend_pool",
			resource:       resourceLBBackendPool(),
			requiredFields: []string{"name"},
			optionalFields: []string{"region", "backend_refs", "user_labels"},
			computedFields: []string{"pool_id", "system_labels", "created_at", "fqid"},
		},
		{
			name:           "evroc_lb_backend_service",
			resource:       resourceLBBackendService(),
			requiredFields: []string{"name", "port", "backend_pool_ref"},
			optionalFields: []string{"region", "proxy_protocol", "user_labels"},
			computedFields: []string{"service_id", "system_labels", "created_at", "fqid"},
		},
		{
			name:           "evroc_lb_l4_route",
			resource:       resourceLBL4Route(),
			requiredFields: []string{"name", "default_backend_service_ref"},
			optionalFields: []string{"region", "user_labels"},
			computedFields: []string{"route_id", "system_labels", "created_at", "fqid"},
		},
		{
			name:           "evroc_think_api_key",
			resource:       resourceThinkAPIKey(),
			requiredFields: []string{"name"},
			optionalFields: []string{"expiry", "project"},
			computedFields: []string{"token", "token_prefix", "created_at"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.resource == nil {
				t.Fatal("resource is nil")
			}

			// Verify CRUD functions are set
			if tt.resource.CreateContext == nil {
				t.Error("CreateContext is not set")
			}
			if tt.resource.ReadContext == nil {
				t.Error("ReadContext is not set")
			}

			// Think API key is immutable (no update)
			if tt.name != "evroc_think_api_key" {
				if tt.resource.UpdateContext == nil {
					t.Error("UpdateContext is not set")
				}
			}

			if tt.resource.DeleteContext == nil {
				t.Error("DeleteContext is not set")
			}

			// Think API key doesn't support import
			if tt.name != "evroc_think_api_key" {
				if tt.resource.Importer == nil {
					t.Error("Importer is not set (resource does not support import)")
				}
			}

			// Verify timeouts
			if tt.resource.Timeouts == nil {
				t.Error("Timeouts is not set")
			} else {
				if tt.resource.Timeouts.Create == nil {
					t.Error("Create timeout is not set")
				}
				// Think API key is immutable (no update)
				if tt.name != "evroc_think_api_key" {
					if tt.resource.Timeouts.Update == nil {
						t.Error("Update timeout is not set")
					}
				}
				if tt.resource.Timeouts.Delete == nil {
					t.Error("Delete timeout is not set")
				}
			}

			// Verify description is set
			if tt.resource.Description == "" {
				t.Error("resource Description is empty")
			}

			// Verify required fields exist and are Required
			for _, field := range tt.requiredFields {
				s, ok := tt.resource.Schema[field]
				if !ok {
					t.Errorf("required field %q not found in schema", field)
					continue
				}
				if !s.Required {
					t.Errorf("field %q should be Required", field)
				}
				if s.Description == "" {
					t.Errorf("field %q is missing Description", field)
				}
			}

			// Verify optional fields exist
			for _, field := range tt.optionalFields {
				s, ok := tt.resource.Schema[field]
				if !ok {
					t.Errorf("optional field %q not found in schema", field)
					continue
				}
				if !s.Optional {
					t.Errorf("field %q should be Optional", field)
				}
				if s.Description == "" {
					t.Errorf("field %q is missing Description", field)
				}
			}

			// Verify computed fields exist and are Computed
			for _, field := range tt.computedFields {
				s, ok := tt.resource.Schema[field]
				if !ok {
					t.Errorf("computed field %q not found in schema", field)
					continue
				}
				if !s.Computed {
					t.Errorf("field %q should be Computed", field)
				}
				if s.Description == "" {
					t.Errorf("field %q is missing Description", field)
				}
			}

			// Validate schema structure
			if err := tt.resource.InternalValidate(nil, true); err != nil {
				t.Errorf("schema validation failed: %s", err)
			}
		})
	}
}

// TestDataSourceSchemas verifies that all data source schemas are valid.
func TestDataSourceSchemas(t *testing.T) {
	tests := []struct {
		name           string
		resource       *schema.Resource
		requiredFields []string
		computedFields []string
	}{
		{
			name:           "evroc_disk (data)",
			resource:       dataSourceDisk(),
			requiredFields: []string{"name"},
			computedFields: []string{"disk_id", "size", "image", "created_at", "fqid"},
		},
		{
			name:           "evroc_virtual_machine (data)",
			resource:       dataSourceVirtualMachine(),
			requiredFields: []string{"name"},
			computedFields: []string{"vm_id", "created_at", "fqid", "data_disks"},
		},
		{
			name:           "evroc_public_ip (data)",
			resource:       dataSourcePublicIP(),
			requiredFields: []string{"name"},
			computedFields: []string{"ip_id", "ip_address", "created_at", "fqid"},
		},
		{
			name:           "evroc_security_group (data)",
			resource:       dataSourceSecurityGroup(),
			requiredFields: []string{"name"},
			computedFields: []string{"sg_id", "created_at", "fqid"},
		},
		{
			name:           "evroc_placement_group (data)",
			resource:       dataSourcePlacementGroup(),
			requiredFields: []string{"name"},
			computedFields: []string{"pg_id", "strategy", "created_at", "fqid"},
		},
		{
			name:           "evroc_hotswap_disk_attachment (data)",
			resource:       dataSourceHotswapDiskAttachment(),
			requiredFields: []string{"name"},
			computedFields: []string{"attachment_id", "virtual_machine", "disk", "created_at"},
		},
		{
			name:           "evroc_bucket (data)",
			resource:       dataSourceBucket(),
			requiredFields: []string{"name"},
			computedFields: []string{"bucket_id", "created_at"},
		},
		{
			name:           "evroc_bucket_service_account (data)",
			resource:       dataSourceBucketServiceAccount(),
			requiredFields: []string{"name"},
			computedFields: []string{"service_account_id", "created_at"},
		},
		{
			name:           "evroc_disk_images (data)",
			resource:       dataSourceDiskImages(),
			requiredFields: []string{},
			computedFields: []string{"images"},
		},
		{
			name:           "evroc_compute_profiles (data)",
			resource:       dataSourceComputeProfiles(),
			requiredFields: []string{},
			computedFields: []string{"profiles", "series"},
		},
		{
			name:           "evroc_project (data)",
			resource:       dataSourceProject(),
			requiredFields: []string{"name"},
			computedFields: []string{"project_id", "created_at"},
		},
		{
			name:           "evroc_think_instance (data)",
			resource:       dataSourceThinkInstance(),
			requiredFields: []string{"name"},
			computedFields: []string{"instance_id", "model", "phase", "endpoint"},
		},
		{
			name:           "evroc_think_models (data)",
			resource:       dataSourceThinkModels(),
			requiredFields: []string{},
			computedFields: []string{"models"},
		},
		{
			name:           "evroc_think_sizes (data)",
			resource:       dataSourceThinkSizes(),
			requiredFields: []string{},
			computedFields: []string{"sizes"},
		},
		{
			name:           "evroc_loadbalancer (data)",
			resource:       dataSourceLoadBalancer(),
			requiredFields: []string{"name"},
			computedFields: []string{"lb_id", "public_ip_ref", "created_at", "fqid"},
		},
		{
			name:           "evroc_lb_backend_pool (data)",
			resource:       dataSourceLBBackendPool(),
			requiredFields: []string{"name"},
			computedFields: []string{"pool_id", "created_at", "fqid"},
		},
		{
			name:           "evroc_lb_backend_service (data)",
			resource:       dataSourceLBBackendService(),
			requiredFields: []string{"name"},
			computedFields: []string{"service_id", "port", "created_at", "fqid"},
		},
		{
			name:           "evroc_lb_l4_route (data)",
			resource:       dataSourceLBL4Route(),
			requiredFields: []string{"name"},
			computedFields: []string{"route_id", "default_backend_service_ref", "created_at", "fqid"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.resource == nil {
				t.Fatal("data source is nil")
			}

			// Data sources should have ReadContext
			if tt.resource.ReadContext == nil {
				t.Error("ReadContext is not set")
			}

			// Data sources should NOT have Create/Update/Delete
			if tt.resource.CreateContext != nil {
				t.Error("data source should not have CreateContext")
			}
			if tt.resource.UpdateContext != nil {
				t.Error("data source should not have UpdateContext")
			}
			if tt.resource.DeleteContext != nil {
				t.Error("data source should not have DeleteContext")
			}

			// Verify description is set
			if tt.resource.Description == "" {
				t.Error("data source Description is empty")
			}

			// Verify required fields
			for _, field := range tt.requiredFields {
				s, ok := tt.resource.Schema[field]
				if !ok {
					t.Errorf("required field %q not found", field)
					continue
				}
				if !s.Required {
					t.Errorf("field %q should be Required", field)
				}
				if s.Description == "" {
					t.Errorf("field %q is missing Description", field)
				}
			}

			// Verify computed fields
			for _, field := range tt.computedFields {
				s, ok := tt.resource.Schema[field]
				if !ok {
					t.Errorf("computed field %q not found", field)
					continue
				}
				if !s.Computed {
					t.Errorf("field %q should be Computed", field)
				}
			}

			// Validate schema structure
			if err := tt.resource.InternalValidate(nil, false); err != nil {
				t.Errorf("schema validation failed: %s", err)
			}
		})
	}
}

// TestProviderSchema verifies provider configuration schema.
func TestProviderSchema(t *testing.T) {
	p := New("test")()

	// Verify sensitive fields
	sensitiveFields := []string{"token", "refresh_token", "password"}
	for _, field := range sensitiveFields {
		s, ok := p.Schema[field]
		if !ok {
			t.Errorf("provider field %q not found", field)
			continue
		}
		if !s.Sensitive {
			t.Errorf("field %q should be Sensitive", field)
		}
	}

	// Verify non-sensitive fields
	nonSensitiveFields := []string{"username", "organization", "project", "region", "api_endpoint"}
	for _, field := range nonSensitiveFields {
		s, ok := p.Schema[field]
		if !ok {
			t.Errorf("provider field %q not found", field)
			continue
		}
		if s.Sensitive {
			t.Errorf("field %q should NOT be Sensitive", field)
		}
	}

	// Verify all provider fields are optional (auth is validated at configure time)
	for name, s := range p.Schema {
		if !s.Optional {
			t.Errorf("provider field %q should be Optional", name)
		}
		if s.Description == "" {
			t.Errorf("provider field %q is missing Description", name)
		}
	}

	// Verify resource count
	expectedResources := 17
	if len(p.ResourcesMap) != expectedResources {
		t.Errorf("expected %d resources, got %d", expectedResources, len(p.ResourcesMap))
	}

	// Verify data source count
	expectedDataSources := 20
	if len(p.DataSourcesMap) != expectedDataSources {
		t.Errorf("expected %d data sources, got %d", expectedDataSources, len(p.DataSourcesMap))
	}

	// Verify all resources are registered
	expectedResourceNames := []string{
		"evroc_disk",
		"evroc_public_ip",
		"evroc_virtual_machine",
		"evroc_security_group",
		"evroc_placement_group",
		"evroc_hotswap_disk_attachment",
		"evroc_bucket",
		"evroc_bucket_service_account",
		"evroc_project",
		"evroc_think_instance",
		"evroc_think_api_key",
		"evroc_permission_set",
		"evroc_loadbalancer",
		"evroc_lb_backend_pool",
		"evroc_lb_backend_service",
		"evroc_lb_l4_route",
	}
	for _, name := range expectedResourceNames {
		if _, ok := p.ResourcesMap[name]; !ok {
			t.Errorf("resource %q not registered", name)
		}
	}

	// Verify all data sources are registered
	expectedDataSourceNames := []string{
		"evroc_disk",
		"evroc_public_ip",
		"evroc_virtual_machine",
		"evroc_security_group",
		"evroc_placement_group",
		"evroc_hotswap_disk_attachment",
		"evroc_bucket",
		"evroc_bucket_service_account",
		"evroc_disk_images",
		"evroc_compute_profiles",
		"evroc_project",
		"evroc_think_instance",
		"evroc_think_models",
		"evroc_think_sizes",
		"evroc_permission_set",
		"evroc_loadbalancer",
		"evroc_lb_backend_pool",
		"evroc_lb_backend_service",
		"evroc_lb_l4_route",
	}
	for _, name := range expectedDataSourceNames {
		if _, ok := p.DataSourcesMap[name]; !ok {
			t.Errorf("data source %q not registered", name)
		}
	}
}

// TestResourceForceNewFields verifies that immutable fields are marked ForceNew.
func TestResourceForceNewFields(t *testing.T) {
	tests := []struct {
		name           string
		resource       *schema.Resource
		forceNewFields []string
		mutableFields  []string
	}{
		{
			name:           "evroc_disk",
			resource:       resourceDisk(),
			forceNewFields: []string{"name", "size", "image", "zone"},
			mutableFields:  []string{"user_labels"},
		},
		{
			name:           "evroc_virtual_machine",
			resource:       resourceVirtualMachine(),
			forceNewFields: []string{"name", "boot_disk", "zone", "data_disks"},
			mutableFields:  []string{"flavor", "running", "public_ip", "security_groups", "placement_group", "user_labels"},
		},
		{
			name:           "evroc_public_ip",
			resource:       resourcePublicIP(),
			forceNewFields: []string{"name"},
			mutableFields:  []string{"user_labels"},
		},
		{
			name:           "evroc_security_group",
			resource:       resourceSecurityGroup(),
			forceNewFields: []string{"name"},
			mutableFields:  []string{"rule", "user_labels"},
		},
		{
			name:           "evroc_placement_group",
			resource:       resourcePlacementGroup(),
			forceNewFields: []string{"name", "strategy"},
			mutableFields:  []string{"user_labels"},
		},
		{
			name:           "evroc_hotswap_disk_attachment",
			resource:       resourceHotswapDiskAttachment(),
			forceNewFields: []string{"name", "virtual_machine", "disk"},
			mutableFields:  []string{"user_labels"},
		},
		{
			name:           "evroc_bucket",
			resource:       resourceBucket(),
			forceNewFields: []string{"name"},
			mutableFields:  []string{"object_retention_mode", "user_labels"},
		},
		{
			name:           "evroc_project",
			resource:       resourceProject(),
			forceNewFields: []string{"name", "organization"},
			mutableFields:  []string{"display_name", "user_labels"},
		},
		{
			name:           "evroc_think_instance",
			resource:       resourceThinkInstance(),
			forceNewFields: []string{"name", "model", "region"},
			mutableFields:  []string{"running", "user_labels"},
		},
		{
			name:           "evroc_think_api_key",
			resource:       resourceThinkAPIKey(),
			forceNewFields: []string{"name", "expiry"},
			mutableFields:  []string{},
		},
		{
			name:           "evroc_permission_set",
			resource:       resourcePermissionSet(),
			forceNewFields: []string{"name", "project", "email"},
			mutableFields:  []string{"admin", "user_labels"},
		},
		{
			name:           "evroc_loadbalancer",
			resource:       resourceLoadBalancer(),
			forceNewFields: []string{"name", "public_ip_ref"},
			mutableFields:  []string{"listener", "user_labels"},
		},
		{
			name:           "evroc_lb_backend_pool",
			resource:       resourceLBBackendPool(),
			forceNewFields: []string{"name"},
			mutableFields:  []string{"backend_refs", "user_labels"},
		},
		{
			name:           "evroc_lb_backend_service",
			resource:       resourceLBBackendService(),
			forceNewFields: []string{"name"},
			mutableFields:  []string{"port", "backend_pool_ref", "proxy_protocol", "user_labels"},
		},
		{
			name:           "evroc_lb_l4_route",
			resource:       resourceLBL4Route(),
			forceNewFields: []string{"name"},
			mutableFields:  []string{"default_backend_service_ref", "user_labels"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, field := range tt.forceNewFields {
				s, ok := tt.resource.Schema[field]
				if !ok {
					t.Errorf("field %q not found", field)
					continue
				}
				if !s.ForceNew {
					t.Errorf("field %q should be ForceNew", field)
				}
			}
			for _, field := range tt.mutableFields {
				s, ok := tt.resource.Schema[field]
				if !ok {
					t.Errorf("field %q not found", field)
					continue
				}
				if s.ForceNew {
					t.Errorf("field %q should NOT be ForceNew (it is mutable)", field)
				}
			}
		})
	}
}
