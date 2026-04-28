// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	evroc "github.com/evroc-oss/evroc-go-sdk"
	"github.com/evroc-oss/evroc-go-sdk/config"
	computetypes "github.com/evroc-oss/evroc-go-sdk/types/compute"
	iamtypes "github.com/evroc-oss/evroc-go-sdk/types/iam"
	networkingtypes "github.com/evroc-oss/evroc-go-sdk/types/networking"
	storagetypes "github.com/evroc-oss/evroc-go-sdk/types/storage"
	thinktypes "github.com/evroc-oss/evroc-go-sdk/types/think"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// mockServer provides a test HTTP server that mimics the evroc API.
type mockServer struct {
	server  *httptest.Server
	mux     *http.ServeMux
	deleted map[string]bool // tracks deleted resources by path
}

func newMockServer() *mockServer {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	return &mockServer{server: server, mux: mux, deleted: make(map[string]bool)}
}

func (m *mockServer) markDeleted(path string) {
	m.deleted[path] = true
}

func (m *mockServer) isDeleted(path string) bool {
	return m.deleted[path]
}

func (m *mockServer) close() {
	m.server.Close()
}

// newTestClient creates an evroc client pointing at the mock server.
func newTestClient(t *testing.T, serverURL string) *evroc.Client {
	t.Helper()
	cfg := &config.Config{
		Auth: config.AuthConfig{
			Token: "fake-test-token",
		},
		API: config.APIConfig{
			BaseURL: serverURL,
		},
		Context: config.ContextConfig{
			Organization: "test-org",
			Project:      "test-project",
			Region:       "se-sto",
		},
	}
	client, err := evroc.New(context.Background(), *cfg, evroc.WithHTTPClient(http.DefaultClient))
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}
	return client
}

// newTestProviderConfig creates a ProviderConfig for testing.
func newTestProviderConfig(t *testing.T, serverURL string) *ProviderConfig {
	t.Helper()
	return &ProviderConfig{
		Client:  newTestClient(t, serverURL),
		Project: "test-project",
		Region:  "se-sto",
		clients: make(map[string]*evroc.Client),
	}
}

// newTestResourceData creates ResourceData for a given resource.
func newTestResourceData(t *testing.T, res *schema.Resource) *schema.ResourceData {
	t.Helper()
	return res.TestResourceData()
}

// respondJSON writes a JSON response.
func respondJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// ---------- Disk mock responses ----------

func mockDisk(name string) *computetypes.Disk {
	conds := []computetypes.DiskStatusConditionsItem{
		{Type: "Ready", Status: "True"},
	}
	region := "se-sto"
	imgRef := "/compute/global/diskImages/evroc/ubuntu-minimal.24-04.1"
	zone := "se-sto-1a"
	uid := openapi_types.UUID{}
	return &computetypes.Disk{
		ApiVersion: "v1beta1",
		Kind:       "Disk",
		Metadata: computetypes.RegionalMetadataResponse{
			Id:                name,
			CreationTimestamp: time.Now(),
			Generation:        1,
			Region:            &region,
			Uid:               uid,
		},
		Spec: computetypes.DiskSpec{
			DiskImageRef: &imgRef,
			DiskSize: &computetypes.DiskSpecDiskSize{
				Amount: 100,
				Unit:   computetypes.DiskSpecDiskSizeUnitGB,
			},
			Placement: computetypes.DiskSpecPlacement{
				Zone: &zone,
			},
		},
		Status: computetypes.DiskStatus{
			Conditions: &conds,
		},
	}
}

func setupDiskHandlers(ms *mockServer, name string) {
	disk := mockDisk(name)
	resourcePath := fmt.Sprintf("/compute/v1beta1/projects/test-project/regions/se-sto/disks/%s", name)
	ms.mux.HandleFunc("/compute/v1beta1/projects/test-project/regions/se-sto/disks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			respondJSON(w, http.StatusCreated, disk)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})
	ms.mux.HandleFunc(resourcePath, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if ms.isDeleted(resourcePath) {
				respondJSON(w, http.StatusNotFound, map[string]string{"reason": "not found"})
				return
			}
			respondJSON(w, http.StatusOK, disk)
		case http.MethodDelete:
			ms.markDeleted(resourcePath)
			w.WriteHeader(http.StatusNoContent)
		case http.MethodPatch:
			respondJSON(w, http.StatusOK, disk)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

// ---------- Public IP mock responses ----------

func mockPublicIP(name string) *networkingtypes.PublicIP {
	conds := []networkingtypes.PublicIPStatusConditionsItem{
		{Type: "Ready", Status: "True"},
	}
	region := "se-sto"
	ip := "203.0.113.1"
	uid := openapi_types.UUID{}
	return &networkingtypes.PublicIP{
		ApiVersion: "v1beta1",
		Kind:       "PublicIP",
		Metadata: networkingtypes.RegionalMetadataResponse{
			Id:                name,
			CreationTimestamp: time.Now(),
			Generation:        1,
			Region:            &region,
			Uid:               uid,
		},
		Status: networkingtypes.PublicIPStatus{
			Conditions:        &conds,
			PublicIPv4Address: &ip,
		},
	}
}

func setupPublicIPHandlers(ms *mockServer, name string) {
	pip := mockPublicIP(name)
	resourcePath := fmt.Sprintf("/networking/v1beta1/projects/test-project/regions/se-sto/publicIPs/%s", name)
	ms.mux.HandleFunc("/networking/v1beta1/projects/test-project/regions/se-sto/publicIPs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			respondJSON(w, http.StatusCreated, pip)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})
	ms.mux.HandleFunc(resourcePath, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if ms.isDeleted(resourcePath) {
				respondJSON(w, http.StatusNotFound, map[string]string{"reason": "not found"})
				return
			}
			respondJSON(w, http.StatusOK, pip)
		case http.MethodDelete:
			ms.markDeleted(resourcePath)
			w.WriteHeader(http.StatusNoContent)
		case http.MethodPatch:
			respondJSON(w, http.StatusOK, pip)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

// ---------- Security Group mock responses ----------

func mockSecurityGroup(name string) *networkingtypes.SecurityGroup {
	conds := []networkingtypes.SecurityGroupStatusConditionsItem{
		{Type: "Ready", Status: "True"},
	}
	region := "se-sto"
	uid := openapi_types.UUID{}
	return &networkingtypes.SecurityGroup{
		ApiVersion: "v1beta1",
		Kind:       "SecurityGroup",
		Metadata: networkingtypes.RegionalMetadataResponse{
			Id:                name,
			CreationTimestamp: time.Now(),
			Generation:        1,
			Region:            &region,
			Uid:               uid,
		},
		Status: networkingtypes.SecurityGroupStatus{
			Conditions: &conds,
		},
	}
}

func setupSecurityGroupHandlers(ms *mockServer, name string) {
	sg := mockSecurityGroup(name)
	resourcePath := fmt.Sprintf("/networking/v1beta1/projects/test-project/regions/se-sto/securityGroups/%s", name)
	ms.mux.HandleFunc("/networking/v1beta1/projects/test-project/regions/se-sto/securityGroups", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			respondJSON(w, http.StatusCreated, sg)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})
	ms.mux.HandleFunc(resourcePath, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if ms.isDeleted(resourcePath) {
				respondJSON(w, http.StatusNotFound, map[string]string{"reason": "not found"})
				return
			}
			respondJSON(w, http.StatusOK, sg)
		case http.MethodDelete:
			ms.markDeleted(resourcePath)
			w.WriteHeader(http.StatusNoContent)
		case http.MethodPatch:
			respondJSON(w, http.StatusOK, sg)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

// ---------- Placement Group mock responses ----------

func mockPlacementGroup(name string) *computetypes.PlacementGroup {
	conds := []computetypes.PlacementGroupStatusConditionsItem{
		{Type: "Ready", Status: "True"},
	}
	region := "se-sto"
	zone := "se-sto-1a"
	uid := openapi_types.UUID{}
	return &computetypes.PlacementGroup{
		ApiVersion: "v1beta1",
		Kind:       "PlacementGroup",
		Metadata: computetypes.RegionalMetadataResponse{
			Id:                name,
			CreationTimestamp: time.Now(),
			Generation:        1,
			Region:            &region,
			Uid:               uid,
		},
		Spec: computetypes.PlacementGroupSpec{
			Strategy: computetypes.PlacementGroupSpecStrategy{
				Type: "spread",
			},
			Placement: computetypes.PlacementGroupSpecPlacement{
				Zone: &zone,
			},
		},
		Status: computetypes.PlacementGroupStatus{
			Conditions: &conds,
		},
	}
}

func setupPlacementGroupHandlers(ms *mockServer, name string) {
	pg := mockPlacementGroup(name)
	resourcePath := fmt.Sprintf("/compute/v1beta1/projects/test-project/regions/se-sto/placementGroups/%s", name)
	ms.mux.HandleFunc("/compute/v1beta1/projects/test-project/regions/se-sto/placementGroups", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			respondJSON(w, http.StatusCreated, pg)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})
	ms.mux.HandleFunc(resourcePath, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if ms.isDeleted(resourcePath) {
				respondJSON(w, http.StatusNotFound, map[string]string{"reason": "not found"})
				return
			}
			respondJSON(w, http.StatusOK, pg)
		case http.MethodDelete:
			ms.markDeleted(resourcePath)
			w.WriteHeader(http.StatusNoContent)
		case http.MethodPatch:
			respondJSON(w, http.StatusOK, pg)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

// ---------- Bucket mock responses ----------

func mockBucket(name string) *storagetypes.Bucket {
	conds := []storagetypes.BucketStatusConditionsItem{
		{Type: "Ready", Status: "True"},
	}
	region := "se-sto"
	retMode := storagetypes.BucketSpecObjectRetentionMode("Disabled")
	uid := openapi_types.UUID{}
	return &storagetypes.Bucket{
		ApiVersion: "v1beta1",
		Kind:       "Bucket",
		Metadata: storagetypes.RegionalMetadataResponse{
			Id:                name,
			CreationTimestamp: time.Now(),
			Generation:        1,
			Region:            &region,
			Uid:               uid,
		},
		Spec: storagetypes.BucketSpec{
			ObjectRetentionMode: &retMode,
		},
		Status: storagetypes.BucketStatus{
			Conditions: &conds,
		},
	}
}

func setupBucketHandlers(ms *mockServer, name string) {
	bucket := mockBucket(name)
	resourcePath := fmt.Sprintf("/storage/v1/projects/test-project/regions/se-sto/buckets/%s", name)
	ms.mux.HandleFunc("/storage/v1/projects/test-project/regions/se-sto/buckets", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			respondJSON(w, http.StatusCreated, bucket)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})
	ms.mux.HandleFunc(resourcePath, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if ms.isDeleted(resourcePath) {
				respondJSON(w, http.StatusNotFound, map[string]string{"reason": "not found"})
				return
			}
			respondJSON(w, http.StatusOK, bucket)
		case http.MethodDelete:
			ms.markDeleted(resourcePath)
			w.WriteHeader(http.StatusNoContent)
		case http.MethodPatch:
			respondJSON(w, http.StatusOK, bucket)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

// ---------- Bucket Service Account mock responses ----------

func mockBucketServiceAccount(name string) *storagetypes.BucketServiceAccount {
	conds := []storagetypes.BucketServiceAccountStatusConditionsItem{
		{Type: "Ready", Status: "True"},
	}
	region := "se-sto"
	secret := "s3-credentials-secret"
	buckets := []string{"/storage/projects/test-project/regions/se-sto/buckets/test-bucket"}
	uid := openapi_types.UUID{}
	return &storagetypes.BucketServiceAccount{
		ApiVersion: "v1beta1",
		Kind:       "BucketServiceAccount",
		Metadata: storagetypes.RegionalMetadataResponse{
			Id:                name,
			CreationTimestamp: time.Now(),
			Generation:        1,
			Region:            &region,
			Uid:               uid,
		},
		Spec: storagetypes.BucketServiceAccountSpec{
			Buckets: &buckets,
		},
		Status: storagetypes.BucketServiceAccountStatus{
			Conditions:              &conds,
			S3CredentialsSecretName: &secret,
		},
	}
}

func setupBucketServiceAccountHandlers(ms *mockServer, name string) {
	sa := mockBucketServiceAccount(name)
	resourcePath := fmt.Sprintf("/storage/v1/projects/test-project/regions/se-sto/bucketServiceAccounts/%s", name)
	ms.mux.HandleFunc("/storage/v1/projects/test-project/regions/se-sto/bucketServiceAccounts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			respondJSON(w, http.StatusCreated, sa)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})
	ms.mux.HandleFunc(resourcePath, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if ms.isDeleted(resourcePath) {
				respondJSON(w, http.StatusNotFound, map[string]string{"reason": "not found"})
				return
			}
			respondJSON(w, http.StatusOK, sa)
		case http.MethodDelete:
			ms.markDeleted(resourcePath)
			w.WriteHeader(http.StatusNoContent)
		case http.MethodPatch:
			respondJSON(w, http.StatusOK, sa)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

// ---------- Project mock responses ----------

func mockProject(name string) *iamtypes.Project {
	conds := []iamtypes.ProjectStatusConditionsItem{
		{Type: "Ready", Status: "True"},
	}
	displayName := "Test Project"
	uid := openapi_types.UUID{}
	return &iamtypes.Project{
		ApiVersion: "v1beta1",
		Kind:       "Project",
		Metadata: iamtypes.GlobalMetadataResponse{
			Id:                name,
			CreationTimestamp: time.Now(),
			Generation:        1,
			Uid:               uid,
		},
		Spec: iamtypes.ProjectSpec{
			Name:         &displayName,
			Organization: "test-org",
		},
		Status: iamtypes.ProjectStatus{
			Conditions: &conds,
		},
	}
}

func setupProjectHandlers(ms *mockServer, name string) {
	project := mockProject(name)
	resourcePath := fmt.Sprintf("/iam/v1beta1/projects/%s", name)
	ms.mux.HandleFunc("/iam/v1beta1/projects", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			respondJSON(w, http.StatusCreated, project)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})
	ms.mux.HandleFunc(resourcePath, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if ms.isDeleted(resourcePath) {
				respondJSON(w, http.StatusNotFound, map[string]string{"reason": "not found"})
				return
			}
			respondJSON(w, http.StatusOK, project)
		case http.MethodDelete:
			ms.markDeleted(resourcePath)
			w.WriteHeader(http.StatusNoContent)
		case http.MethodPatch:
			respondJSON(w, http.StatusOK, project)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

// ---------- Virtual Machine mock responses ----------

func mockVirtualMachine(name string) *computetypes.VirtualMachine {
	conds := []computetypes.VirtualMachineStatusConditionsItem{
		{Type: "Ready", Status: "True"},
	}
	region := "se-sto"
	zone := "se-sto-1a"
	uid := openapi_types.UUID{}
	running := true
	bootFrom := true
	publicIPRef := "/networking/projects/test-project/regions/se-sto/publicIPs/test-pip"
	sgRefs := []string{"/networking/projects/test-project/regions/se-sto/securityGroups/test-sg"}
	pubIP := "203.0.113.1"
	privIP := "10.0.0.1"
	sshKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA test"
	userData := "#!/bin/bash\necho hello"
	pgRef := "/compute/projects/test-project/regions/se-sto/placementGroups/test-pg"
	return &computetypes.VirtualMachine{
		ApiVersion: "v1beta1",
		Kind:       "VirtualMachine",
		Metadata: computetypes.RegionalMetadataResponse{
			Id:                name,
			CreationTimestamp: time.Now(),
			Generation:        1,
			Region:            &region,
			Uid:               uid,
		},
		Spec: computetypes.VirtualMachineSpec{
			ComputeProfileRef: "/compute/global/computeProfiles/a1a.s",
			Disks: &[]computetypes.VirtualMachineSpecDisksItem{
				{DiskRef: "/compute/projects/test-project/regions/se-sto/disks/test-disk", BootFrom: &bootFrom},
			},
			Running: &running,
			OsSettings: &computetypes.VirtualMachineSpecOsSettings{
				CloudInitUserData: &userData,
				Ssh: &struct {
					AuthorizedKeys *[]computetypes.VirtualMachineSpecOsSettingsAuthorizedKeysItem `json:"authorizedKeys,omitempty"`
				}{
					AuthorizedKeys: &[]computetypes.VirtualMachineSpecOsSettingsAuthorizedKeysItem{
						{Value: sshKey},
					},
				},
			},
			Networking: &computetypes.VirtualMachineSpecNetworking{
				PublicIPv4Address: &struct {
					Static *computetypes.VirtualMachineSpecNetworkingStatic `json:"static,omitempty"`
				}{
					Static: &computetypes.VirtualMachineSpecNetworkingStatic{
						PublicIPRef: &publicIPRef,
					},
				},
				SecurityGroupSettings: &struct {
					SecurityGroupMemberRefs *[]string `json:"securityGroupMemberRefs,omitempty"`
				}{
					SecurityGroupMemberRefs: &sgRefs,
				},
			},
			Placement: computetypes.VirtualMachineSpecPlacement{
				Zone:              &zone,
				PlacementGroupRef: &pgRef,
			},
		},
		Status: computetypes.VirtualMachineStatus{
			Conditions:           &conds,
			VirtualMachineStatus: stringPtr("Running"),
			Networking: &computetypes.VirtualMachineStatusNetworking{
				PublicIPv4Address:  &pubIP,
				PrivateIPv4Address: &privIP,
			},
		},
	}
}

func setupVirtualMachineHandlers(ms *mockServer, name string) {
	vm := mockVirtualMachine(name)
	resourcePath := fmt.Sprintf("/compute/v1beta1/projects/test-project/regions/se-sto/virtualMachines/%s", name)
	ms.mux.HandleFunc("/compute/v1beta1/projects/test-project/regions/se-sto/virtualMachines", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			respondJSON(w, http.StatusCreated, vm)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})
	ms.mux.HandleFunc(resourcePath, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if ms.isDeleted(resourcePath) {
				respondJSON(w, http.StatusNotFound, map[string]string{"reason": "not found"})
				return
			}
			respondJSON(w, http.StatusOK, vm)
		case http.MethodDelete:
			ms.markDeleted(resourcePath)
			w.WriteHeader(http.StatusNoContent)
		case http.MethodPatch:
			// Decode patch to track running state changes
			var patch computetypes.VirtualMachine
			if err := json.NewDecoder(r.Body).Decode(&patch); err == nil {
				if patch.Spec.Running != nil {
					vm.Spec.Running = patch.Spec.Running
					if *patch.Spec.Running {
						vm.Status.VirtualMachineStatus = stringPtr("Running")
					} else {
						vm.Status.VirtualMachineStatus = stringPtr("Stopped")
					}
				}
			}
			respondJSON(w, http.StatusOK, vm)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

// ---------- Disk Attachment mock responses ----------

func mockDiskAttachment(name string) *computetypes.HotswapDiskAttachment {
	conds := []computetypes.HotswapDiskAttachmentStatusConditionsItem{
		{Type: "Ready", Status: "True"},
	}
	region := "se-sto"
	uid := openapi_types.UUID{}
	serial := "disk-serial-123"
	return &computetypes.HotswapDiskAttachment{
		ApiVersion: "v1beta1",
		Kind:       "HotswapDiskAttachment",
		Metadata: computetypes.RegionalMetadataResponse{
			Id:                name,
			CreationTimestamp: time.Now(),
			Generation:        1,
			Region:            &region,
			Uid:               uid,
		},
		Spec: computetypes.HotswapDiskAttachmentSpec{
			VirtualMachineRef: "/compute/projects/test-project/regions/se-sto/virtualMachines/test-vm",
			DiskRef:           "/compute/projects/test-project/regions/se-sto/disks/test-disk",
		},
		Status: computetypes.HotswapDiskAttachmentStatus{
			Conditions: &conds,
			Serial:     &serial,
		},
	}
}

func setupDiskAttachmentHandlers(ms *mockServer, name string) {
	attachment := mockDiskAttachment(name)
	resourcePath := fmt.Sprintf("/compute/v1beta1/projects/test-project/regions/se-sto/hotswapDiskAttachments/%s", name)
	ms.mux.HandleFunc("/compute/v1beta1/projects/test-project/regions/se-sto/hotswapDiskAttachments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			respondJSON(w, http.StatusCreated, attachment)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})
	ms.mux.HandleFunc(resourcePath, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if ms.isDeleted(resourcePath) {
				respondJSON(w, http.StatusNotFound, map[string]string{"reason": "not found"})
				return
			}
			respondJSON(w, http.StatusOK, attachment)
		case http.MethodDelete:
			ms.markDeleted(resourcePath)
			w.WriteHeader(http.StatusNoContent)
		case http.MethodPatch:
			respondJSON(w, http.StatusOK, attachment)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

// ---------- Think Instance mock responses ----------

func mockThinkInstance(name string) *thinktypes.Instance {
	region := "se-sto"
	uid := openapi_types.UUID{}
	phase := thinktypes.Running
	endpoint := "https://models.think.se-sto.evroc.com/projects/test-project/instances/" + name
	size := "a100.2x"
	return &thinktypes.Instance{
		ApiVersion: "v1beta2",
		Kind:       "Instance",
		Metadata: thinktypes.RegionalMetadataResponse{
			Id:                name,
			CreationTimestamp: time.Now(),
			Generation:        1,
			Region:            &region,
			Uid:               uid,
		},
		Spec: thinktypes.InstanceSpec{
			Model: "meta-llama/Llama-3.3-70B-Instruct",
			Size:  &size,
		},
		Status: thinktypes.InstanceStatus{
			Phase:    &phase,
			Endpoint: &endpoint,
		},
	}
}

func setupThinkInstanceHandlers(ms *mockServer, name string) {
	instance := mockThinkInstance(name)
	collectionPath := "/think/v1beta2/projects/test-project/regions/se-sto/instances"
	resourcePath := fmt.Sprintf("%s/%s", collectionPath, name)
	startPath := fmt.Sprintf("%s/start", resourcePath)
	stopPath := fmt.Sprintf("%s/stop", resourcePath)

	ms.mux.HandleFunc(collectionPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			respondJSON(w, http.StatusCreated, instance)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})
	ms.mux.HandleFunc(startPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			phase := thinktypes.Running
			instance.Status.Phase = &phase
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})
	ms.mux.HandleFunc(stopPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			phase := thinktypes.Stopped
			instance.Status.Phase = &phase
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})
	ms.mux.HandleFunc(resourcePath, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if ms.isDeleted(resourcePath) {
				respondJSON(w, http.StatusNotFound, map[string]string{"reason": "not found"})
				return
			}
			respondJSON(w, http.StatusOK, instance)
		case http.MethodDelete:
			ms.markDeleted(resourcePath)
			w.WriteHeader(http.StatusNoContent)
		case http.MethodPatch:
			respondJSON(w, http.StatusOK, instance)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

// ---------- Think API Key mock responses ----------

func mockThinkAPIKey(name string) *thinktypes.Apikey {
	uid := openapi_types.UUID{}
	token := "ev-test-token-secret-1234567890"
	tokenPrefix := "ev-test"
	return &thinktypes.Apikey{
		ApiVersion: "v1beta2",
		Kind:       "Apikey",
		Metadata: thinktypes.GlobalProjectMetadataResponse{
			Id:                name,
			CreationTimestamp: time.Now(),
			Generation:        1,
			Uid:               uid,
		},
		Spec: thinktypes.ApikeySpec{},
		Status: thinktypes.ApikeyStatus{
			Token:       &token,
			TokenPrefix: &tokenPrefix,
		},
	}
}

func setupThinkAPIKeyHandlers(ms *mockServer, name string) {
	apiKey := mockThinkAPIKey(name)
	collectionPath := "/think/v1beta2/projects/test-project/apiKeys"
	resourcePath := fmt.Sprintf("%s/%s", collectionPath, name)

	ms.mux.HandleFunc(collectionPath, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			respondJSON(w, http.StatusCreated, apiKey)
		case http.MethodGet:
			// List endpoint
			items := []thinktypes.Apikey{}
			if !ms.isDeleted(resourcePath) {
				items = append(items, *apiKey)
			}
			respondJSON(w, http.StatusOK, map[string]interface{}{
				"items": items,
			})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	ms.mux.HandleFunc(resourcePath, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodDelete:
			ms.markDeleted(resourcePath)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

// ---------- Think Models mock responses ----------

func setupThinkModelsHandlers(ms *mockServer) {
	desc := "A large language model for chat and code generation."
	handle := "meta-llama/Llama-3.3-70B-Instruct"
	license := "llama3.3"
	defaultSize := "a100.2x"
	uid := openapi_types.UUID{}
	models := []thinktypes.Model{
		{
			ApiVersion: "v1beta2",
			Kind:       "Model",
			Metadata: thinktypes.GlobalMetadataResponse{
				Id:                "meta-llama/Llama-3.3-70B-Instruct",
				CreationTimestamp: time.Now(),
				Generation:        1,
				Uid:               uid,
			},
			Spec: thinktypes.ModelSpec{
				Description: &desc,
				Handle:      &handle,
				License:     &license,
				DefaultSize: defaultSize,
			},
			Status: thinktypes.ModelStatus{},
		},
	}

	ms.mux.HandleFunc("/think/v1beta2/projects/test-project/models", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]interface{}{
				"items": models,
			})
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})
}

// ---------- Think Sizes mock responses ----------

func setupThinkSizesHandlers(ms *mockServer) {
	desc := "2x NVIDIA A100 80GB GPUs"
	uid := openapi_types.UUID{}
	sizes := []thinktypes.Size{
		{
			ApiVersion: "v1beta2",
			Kind:       "Size",
			Metadata: thinktypes.GlobalMetadataResponse{
				Id:                "a100.2x",
				CreationTimestamp: time.Now(),
				Generation:        1,
				Uid:               uid,
			},
			Spec: thinktypes.SizeSpec{
				Description: &desc,
			},
			Status: thinktypes.SizeStatus{},
		},
	}

	ms.mux.HandleFunc("/think/v1beta2/projects/test-project/sizes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]interface{}{
				"items": sizes,
			})
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})
}

// ---------- Catch-all handler ----------

// setupCatchAll adds a catch-all handler for debugging unexpected requests.
func setupCatchAll(ms *mockServer) {
	ms.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/compute/") ||
			strings.HasPrefix(r.URL.Path, "/networking/") ||
			strings.HasPrefix(r.URL.Path, "/storage/") ||
			strings.HasPrefix(r.URL.Path, "/iam/") ||
			strings.HasPrefix(r.URL.Path, "/think/") {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"not found"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}
