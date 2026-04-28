// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"testing"

	networkingtypes "github.com/evroc-oss/evroc-go-sdk/types/networking"
)

func TestBuildDiskCreateRequest(t *testing.T) {
	tests := []struct {
		name       string
		diskName   string
		sizeGB     int
		image      string
		zone       string
		userLabels map[string]string
	}{
		{
			name:     "basic disk",
			diskName: "my-disk",
			sizeGB:   100,
			image:    "ubuntu-minimal.24-04.1",
			zone:     "a",
		},
		{
			name:     "disk without image",
			diskName: "data-disk",
			sizeGB:   50,
			image:    "",
			zone:     "b",
		},
		{
			name:     "disk without zone",
			diskName: "flex-disk",
			sizeGB:   200,
			image:    "rocky-9-6",
			zone:     "",
		},
		{
			name:     "disk with labels",
			diskName: "labeled-disk",
			sizeGB:   100,
			image:    "ubuntu-minimal.24-04.1",
			zone:     "a",
			userLabels: map[string]string{
				"env":  "test",
				"team": "infra",
			},
		},
		{
			name:       "disk with empty labels",
			diskName:   "no-label-disk",
			sizeGB:     100,
			image:      "ubuntu-minimal.24-04.1",
			zone:       "a",
			userLabels: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := BuildDiskCreateRequest(tt.diskName, tt.sizeGB, tt.image, tt.zone, tt.userLabels)
			if req == nil {
				t.Fatal("expected non-nil request")
			}
			if req.Metadata.Id != tt.diskName {
				t.Errorf("expected name %q, got %q", tt.diskName, req.Metadata.Id)
			}
			if tt.sizeGB > 0 && req.Spec.DiskSize == nil {
				t.Error("expected disk size to be set")
			}
			if len(tt.userLabels) > 0 {
				if req.Metadata.UserLabels == nil {
					t.Error("expected user labels to be set")
				} else {
					for k, v := range tt.userLabels {
						got, ok := (*req.Metadata.UserLabels)[k]
						if !ok {
							t.Errorf("expected label %q to exist", k)
						}
						if got != v {
							t.Errorf("expected label %q=%q, got %q", k, v, got)
						}
					}
				}
			}
			if len(tt.userLabels) == 0 && req.Metadata.UserLabels != nil {
				t.Error("expected user labels to be nil when empty")
			}
		})
	}
}

func TestBuildPublicIPCreateRequest(t *testing.T) {
	tests := []struct {
		name       string
		ipName     string
		userLabels map[string]string
	}{
		{
			name:   "basic public IP",
			ipName: "my-ip",
		},
		{
			name:   "public IP with labels",
			ipName: "labeled-ip",
			userLabels: map[string]string{
				"env": "prod",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := BuildPublicIPCreateRequest(tt.ipName, tt.userLabels)
			if req == nil {
				t.Fatal("expected non-nil request")
			}
			if req.Metadata.Id != tt.ipName {
				t.Errorf("expected name %q, got %q", tt.ipName, req.Metadata.Id)
			}
			if len(tt.userLabels) > 0 && req.Metadata.UserLabels == nil {
				t.Error("expected user labels to be set")
			}
		})
	}
}

func TestBuildSecurityGroupCreateRequest(t *testing.T) {
	tests := []struct {
		name       string
		sgName     string
		rules      []networkingtypes.SecurityGroupSpecRulesItem
		userLabels map[string]string
	}{
		{
			name:   "empty rules",
			sgName: "my-sg",
			rules:  nil,
		},
		{
			name:   "with rules",
			sgName: "web-sg",
			rules: func() []networkingtypes.SecurityGroupSpecRulesItem {
				ruleName := "allow-ssh"
				protocol := networkingtypes.SecurityGroupSpecRulesItemProtocol("TCP")
				port := int32(22)
				return []networkingtypes.SecurityGroupSpecRulesItem{
					{
						Name:      &ruleName,
						Direction: networkingtypes.SecurityGroupSpecRulesItemDirection("Ingress"),
						Protocol:  &protocol,
						Port:      &port,
					},
				}
			}(),
		},
		{
			name:   "with labels",
			sgName: "labeled-sg",
			rules:  nil,
			userLabels: map[string]string{
				"env": "staging",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := BuildSecurityGroupCreateRequest(tt.sgName, tt.rules, tt.userLabels)
			if req == nil {
				t.Fatal("expected non-nil request")
			}
			if req.Metadata.Id != tt.sgName {
				t.Errorf("expected name %q, got %q", tt.sgName, req.Metadata.Id)
			}
			if len(tt.rules) > 0 {
				if req.Spec.Rules == nil {
					t.Error("expected rules to be set")
				} else if len(*req.Spec.Rules) != len(tt.rules) {
					t.Errorf("expected %d rules, got %d", len(tt.rules), len(*req.Spec.Rules))
				}
			}
			if len(tt.userLabels) > 0 && req.Metadata.UserLabels == nil {
				t.Error("expected user labels to be set")
			}
		})
	}
}

func TestBuildPlacementGroupCreateRequest(t *testing.T) {
	tests := []struct {
		name       string
		pgName     string
		strategy   string
		zone       string
		userLabels map[string]string
	}{
		{
			name:     "spread strategy",
			pgName:   "my-pg",
			strategy: "spread",
			zone:     "a",
		},
		{
			name:     "cluster strategy",
			pgName:   "cluster-pg",
			strategy: "cluster",
			zone:     "b",
		},
		{
			name:     "no zone",
			pgName:   "flex-pg",
			strategy: "spread",
			zone:     "",
		},
		{
			name:     "with labels",
			pgName:   "labeled-pg",
			strategy: "spread",
			zone:     "a",
			userLabels: map[string]string{
				"tier": "ha",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := BuildPlacementGroupCreateRequest(tt.pgName, tt.strategy, tt.zone, tt.userLabels)
			if req == nil {
				t.Fatal("expected non-nil request")
			}
			if req.Metadata.Id != tt.pgName {
				t.Errorf("expected name %q, got %q", tt.pgName, req.Metadata.Id)
			}
			if len(tt.userLabels) > 0 && req.Metadata.UserLabels == nil {
				t.Error("expected user labels to be set")
			}
		})
	}
}

func TestBuildBucketCreateRequest(t *testing.T) {
	tests := []struct {
		name            string
		bucketName      string
		retentionMode   string
		lockingMode     string
		lockingDuration int32
		userLabels      map[string]string
	}{
		{
			name:          "basic bucket",
			bucketName:    "my-bucket",
			retentionMode: "Disabled",
		},
		{
			name:          "versioned bucket",
			bucketName:    "versioned-bucket",
			retentionMode: "Versioned",
		},
		{
			name:            "locked bucket",
			bucketName:      "locked-bucket",
			retentionMode:   "Locking",
			lockingMode:     "GOVERNANCE",
			lockingDuration: 30,
		},
		{
			name:       "bucket with labels",
			bucketName: "labeled-bucket",
			userLabels: map[string]string{
				"env": "prod",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := BuildBucketCreateRequest(tt.bucketName, tt.retentionMode, tt.lockingMode, tt.lockingDuration, tt.userLabels)
			if req == nil {
				t.Fatal("expected non-nil request")
			}
			if req.Metadata.Id != tt.bucketName {
				t.Errorf("expected name %q, got %q", tt.bucketName, req.Metadata.Id)
			}
			if len(tt.userLabels) > 0 && req.Metadata.UserLabels == nil {
				t.Error("expected user labels to be set")
			}
		})
	}
}

func TestBuildBucketServiceAccountCreateRequest(t *testing.T) {
	tests := []struct {
		name       string
		saName     string
		buckets    []string
		userLabels map[string]string
	}{
		{
			name:    "single bucket",
			saName:  "my-sa",
			buckets: []string{"bucket-1"},
		},
		{
			name:    "multiple buckets",
			saName:  "multi-sa",
			buckets: []string{"bucket-1", "bucket-2", "bucket-3"},
		},
		{
			name:    "with labels",
			saName:  "labeled-sa",
			buckets: []string{"bucket-1"},
			userLabels: map[string]string{
				"role": "backup",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := BuildBucketServiceAccountCreateRequest(tt.saName, tt.buckets, tt.userLabels)
			if req == nil {
				t.Fatal("expected non-nil request")
			}
			if req.Metadata.Id != tt.saName {
				t.Errorf("expected name %q, got %q", tt.saName, req.Metadata.Id)
			}
			if len(tt.userLabels) > 0 && req.Metadata.UserLabels == nil {
				t.Error("expected user labels to be set")
			}
		})
	}
}

func TestBuildProjectCreateRequest(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
		org         string
		displayName string
		userLabels  map[string]string
		wantErr     bool
	}{
		{
			name:        "basic project",
			projectName: "my-project",
			org:         "my-org",
		},
		{
			name:        "project with display name",
			projectName: "prod-project",
			org:         "my-org",
			displayName: "Production Project",
		},
		{
			name:        "project with labels",
			projectName: "labeled-project",
			org:         "my-org",
			userLabels: map[string]string{
				"env": "dev",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := BuildProjectCreateRequest(tt.projectName, tt.org, tt.displayName, tt.userLabels)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if req == nil {
				t.Fatal("expected non-nil request")
			}
		})
	}
}

func TestDerefString(t *testing.T) {
	tests := []struct {
		name     string
		input    *string
		expected string
	}{
		{
			name:     "nil pointer",
			input:    nil,
			expected: "",
		},
		{
			name:     "empty string",
			input:    stringPtr(""),
			expected: "",
		},
		{
			name:     "non-empty string",
			input:    stringPtr("hello"),
			expected: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := derefString(tt.input)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestIsFQID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty string", "", false},
		{"plain name", "my-disk", false},
		{"FQID disk", "/compute/projects/proj/regions/se-sto/disks/my-disk", true},
		{"FQID vm", "/compute/projects/proj/regions/se-sto/virtualMachines/my-vm", true},
		{"FQID public IP", "/networking/projects/proj/regions/se-sto/publicIPs/my-ip", true},
		{"FQID security group", "/networking/projects/proj/regions/se-sto/securityGroups/my-sg", true},
		{"FQID placement group", "/compute/projects/proj/regions/se-sto/placementGroups/my-pg", true},
		{"relative path", "compute/disks/my-disk", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isFQID(tt.input)
			if got != tt.want {
				t.Errorf("isFQID(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSuppressFQIDDiff(t *testing.T) {
	tests := []struct {
		name string
		old  string
		new  string
		want bool
	}{
		{"identical plain names", "my-disk", "my-disk", true},
		{"identical FQIDs", "/compute/projects/p/regions/r/disks/d", "/compute/projects/p/regions/r/disks/d", true},
		{"FQID old, plain new matches", "/compute/projects/p/regions/se-sto/disks/my-disk", "my-disk", true},
		{"FQID old, plain new does not match", "/compute/projects/p/regions/se-sto/disks/other-disk", "my-disk", false},
		{"plain old, FQID new matches", "my-vm", "/compute/projects/p/regions/se-sto/virtualMachines/my-vm", true},
		{"plain old, FQID new does not match", "my-vm", "/compute/projects/p/regions/se-sto/virtualMachines/other-vm", false},
		{"both plain, different", "disk-a", "disk-b", false},
		{"both FQID, different", "/compute/projects/p1/regions/r/disks/d", "/compute/projects/p2/regions/r/disks/d", false},
		{"empty strings", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := suppressFQIDDiff("test_key", tt.old, tt.new, nil)
			if got != tt.want {
				t.Errorf("suppressFQIDDiff(%q, %q) = %v, want %v", tt.old, tt.new, got, tt.want)
			}
		})
	}
}

func TestBuildVirtualMachineCreateRequest(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)

	tests := []struct {
		name           string
		vmName         string
		flavor         string
		bootDisk       string
		sshKeys        []string
		userData       string
		publicIP       string
		zone           string
		securityGroups []string
		placementGroup string
		running        bool
		userLabels     map[string]string
	}{
		{
			name:     "minimal VM",
			vmName:   "basic-vm",
			flavor:   "a1a.s",
			bootDisk: "boot-disk",
			running:  true,
		},
		{
			name:           "full VM with all options",
			vmName:         "full-vm",
			flavor:         "c1a.m",
			bootDisk:       "boot-disk",
			sshKeys:        []string{"ssh-ed25519 AAAA...", "ssh-rsa AAAA..."},
			userData:       "#cloud-config\npackages:\n  - nginx",
			publicIP:       "my-public-ip",
			zone:           "se-sto-1a",
			securityGroups: []string{"sg-web", "sg-ssh"},
			placementGroup: "my-pg",
			running:        true,
			userLabels:     map[string]string{"env": "prod", "team": "infra"},
		},
		{
			name:     "VM stopped with SSH key",
			vmName:   "stopped-vm",
			flavor:   "m1a.l",
			bootDisk: "data-disk",
			sshKeys:  []string{"ssh-ed25519 AAAA..."},
			running:  false,
		},
		{
			name:           "VM with FQID references",
			vmName:         "fqid-vm",
			flavor:         "a1a.s",
			bootDisk:       "/compute/projects/other-proj/regions/se-sto/disks/boot-disk",
			publicIP:       "/networking/projects/other-proj/regions/se-sto/publicIPs/my-ip",
			securityGroups: []string{"/networking/projects/other-proj/regions/se-sto/securityGroups/sg-1"},
			placementGroup: "/compute/projects/other-proj/regions/se-sto/placementGroups/my-pg",
			zone:           "se-sto-1a",
			running:        true,
		},
		{
			name:           "VM with mixed plain and FQID references",
			vmName:         "mixed-vm",
			flavor:         "a1a.s",
			bootDisk:       "boot-disk",
			publicIP:       "/networking/projects/other-proj/regions/se-sto/publicIPs/my-ip",
			securityGroups: []string{"sg-plain", "/networking/projects/other-proj/regions/se-sto/securityGroups/sg-fqid"},
			placementGroup: "my-pg",
			zone:           "se-sto-1a",
			running:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := BuildVirtualMachineCreateRequest(
				config.Client, tt.vmName, tt.flavor, tt.bootDisk,
				tt.sshKeys, tt.userData, tt.publicIP, tt.zone,
				tt.securityGroups, tt.placementGroup, tt.running, tt.userLabels,
			)
			if req == nil {
				t.Fatal("expected non-nil request")
			}
			if req.Metadata.Id != tt.vmName {
				t.Errorf("expected name %q, got %q", tt.vmName, req.Metadata.Id)
			}
			if len(tt.userLabels) > 0 && req.Metadata.UserLabels == nil {
				t.Error("expected user labels to be set")
			}
			if len(tt.userLabels) == 0 && req.Metadata.UserLabels != nil {
				t.Error("expected user labels to be nil when empty")
			}

			// Verify FQID pass-through: if input was FQID, the ref should be preserved
			if req.Spec.Disks != nil && len(*req.Spec.Disks) > 0 {
				diskRef := (*req.Spec.Disks)[0].DiskRef
				if isFQID(tt.bootDisk) && diskRef != tt.bootDisk {
					t.Errorf("FQID boot disk not preserved: got %q, want %q", diskRef, tt.bootDisk)
				}
				if !isFQID(tt.bootDisk) && !isFQID(diskRef) {
					t.Errorf("plain boot disk should be resolved to FQID, got %q", diskRef)
				}
			}
		})
	}
}

func TestBuildDiskAttachmentCreateRequest(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)

	tests := []struct {
		name       string
		attachName string
		vmName     string
		diskName   string
		userLabels map[string]string
	}{
		{
			name:       "basic attachment",
			attachName: "attach-1",
			vmName:     "my-vm",
			diskName:   "my-disk",
		},
		{
			name:       "attachment with labels",
			attachName: "labeled-attach",
			vmName:     "my-vm",
			diskName:   "data-disk",
			userLabels: map[string]string{"env": "test", "purpose": "data"},
		},
		{
			name:       "attachment with FQID references",
			attachName: "fqid-attach",
			vmName:     "/compute/projects/other-proj/regions/se-sto/virtualMachines/my-vm",
			diskName:   "/compute/projects/other-proj/regions/se-sto/disks/my-disk",
		},
		{
			name:       "attachment with mixed references",
			attachName: "mixed-attach",
			vmName:     "my-vm",
			diskName:   "/compute/projects/other-proj/regions/se-sto/disks/my-disk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := BuildDiskAttachmentCreateRequest(config.Client, tt.attachName, tt.vmName, tt.diskName, tt.userLabels)
			if req == nil {
				t.Fatal("expected non-nil request")
			}
			if req.Metadata.Id != tt.attachName {
				t.Errorf("expected name %q, got %q", tt.attachName, req.Metadata.Id)
			}
			if len(tt.userLabels) > 0 && req.Metadata.UserLabels == nil {
				t.Error("expected user labels to be set")
			}
			if len(tt.userLabels) == 0 && req.Metadata.UserLabels != nil {
				t.Error("expected user labels to be nil when empty")
			}

			// Verify FQID pass-through for VM ref
			vmRef := string(req.Spec.VirtualMachineRef)
			if isFQID(tt.vmName) && vmRef != tt.vmName {
				t.Errorf("FQID vm ref not preserved: got %q, want %q", vmRef, tt.vmName)
			}
			if !isFQID(tt.vmName) && !isFQID(vmRef) {
				t.Errorf("plain vm ref should be resolved to FQID, got %q", vmRef)
			}

			// Verify FQID pass-through for disk ref
			diskRef := string(req.Spec.DiskRef)
			if isFQID(tt.diskName) && diskRef != tt.diskName {
				t.Errorf("FQID disk ref not preserved: got %q, want %q", diskRef, tt.diskName)
			}
			if !isFQID(tt.diskName) && !isFQID(diskRef) {
				t.Errorf("plain disk ref should be resolved to FQID, got %q", diskRef)
			}
		})
	}
}

func TestBuildThinkInstanceCreateRequest(t *testing.T) {
	tests := []struct {
		name    string
		instNam string
		model   string
		size    string
		stopped bool
	}{
		{
			name:    "basic instance",
			instNam: "my-instance",
			model:   "meta-llama/Llama-3.3-70B-Instruct",
			size:    "a100.2x",
			stopped: false,
		},
		{
			name:    "instance without size",
			instNam: "no-size",
			model:   "meta-llama/Llama-3.3-70B-Instruct",
			size:    "",
			stopped: false,
		},
		{
			name:    "stopped instance",
			instNam: "stopped-inst",
			model:   "meta-llama/Llama-3.3-70B-Instruct",
			size:    "a100.2x",
			stopped: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := BuildThinkInstanceCreateRequest(tt.instNam, tt.model, tt.size, tt.stopped)
			if req == nil {
				t.Fatal("expected non-nil request")
			}
			if req.Metadata.Id != tt.instNam {
				t.Errorf("expected name %q, got %q", tt.instNam, req.Metadata.Id)
			}
			if req.Spec.Model != tt.model {
				t.Errorf("expected model %q, got %q", tt.model, req.Spec.Model)
			}
			if tt.size != "" && (req.Spec.Size == nil || *req.Spec.Size != tt.size) {
				t.Errorf("expected size %q, got %v", tt.size, req.Spec.Size)
			}
			if tt.size == "" && req.Spec.Size != nil {
				t.Errorf("expected nil size, got %v", *req.Spec.Size)
			}
			if tt.stopped && (req.Spec.Stopped == nil || !*req.Spec.Stopped) {
				t.Error("expected stopped=true")
			}
		})
	}
}

func TestBuildThinkAPIKeyCreateRequest(t *testing.T) {
	tests := []struct {
		name    string
		keyName string
		expiry  string
		wantErr bool
	}{
		{
			name:    "basic key",
			keyName: "my-key",
			expiry:  "",
		},
		{
			name:    "key with expiry",
			keyName: "expiring-key",
			expiry:  "2027-01-01T00:00:00Z",
		},
		{
			name:    "key with invalid expiry",
			keyName: "bad-key",
			expiry:  "not-a-date",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := BuildThinkAPIKeyCreateRequest(tt.keyName, tt.expiry)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if req == nil {
				t.Fatal("expected non-nil request")
			}
			if req.Metadata.Id != tt.keyName {
				t.Errorf("expected name %q, got %q", tt.keyName, req.Metadata.Id)
			}
			if tt.expiry != "" && req.Spec.ExpiryTimestamp == nil {
				t.Error("expected expiry timestamp to be set")
			}
			if tt.expiry == "" && req.Spec.ExpiryTimestamp != nil {
				t.Error("expected expiry timestamp to be nil")
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
