// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import (
	"fmt"
	"net/http"
	"time"

	computetypes "github.com/evroc-oss/evroc-go-sdk/types/compute"
	networkingtypes "github.com/evroc-oss/evroc-go-sdk/types/networking"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func mockSnapshot(name string) *computetypes.Snapshot {
	conds := []computetypes.SnapshotStatusConditionsItem{
		{Type: "Ready", Status: "True"},
	}
	region := "se-sto"
	uid := openapi_types.UUID{}
	diskRef := "/compute/projects/test-project/regions/se-sto/disks/test-disk"
	restoreSize := computetypes.SnapshotStatusRestoreSize{Amount: 20, Unit: "GB"}
	return &computetypes.Snapshot{
		ApiVersion: "v1beta2",
		Kind:       "Snapshot",
		Metadata: computetypes.RegionalMetadataResponse{
			Id:                name,
			CreationTimestamp: time.Now(),
			Generation:        1,
			Region:            &region,
			Uid:               uid,
		},
		Spec: computetypes.SnapshotSpec{
			DiskRef: &diskRef,
		},
		Status: computetypes.SnapshotStatus{
			Conditions:  &conds,
			RestoreSize: &restoreSize,
		},
	}
}

func setupSnapshotHandlers(ms *mockServer, name string) {
	snap := mockSnapshot(name)
	resourcePath := fmt.Sprintf("/compute/v1beta2/projects/test-project/regions/se-sto/snapshots/%s", name)
	ms.mux.HandleFunc("/compute/v1beta2/projects/test-project/regions/se-sto/snapshots", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			respondJSON(w, http.StatusCreated, snap)
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
			respondJSON(w, http.StatusOK, snap)
		case http.MethodDelete:
			ms.markDeleted(resourcePath)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

func mockVPC(name string) *networkingtypes.VirtualPrivateCloud {
	conds := []networkingtypes.VirtualPrivateCloudStatusConditionsItem{
		{Type: "Ready", Status: "True"},
	}
	region := "se-sto"
	uid := openapi_types.UUID{}
	cidrs := []string{"10.0.0.0/16"}
	stackType := networkingtypes.VirtualPrivateCloudSpecStackType("dual-stack")
	return &networkingtypes.VirtualPrivateCloud{
		ApiVersion: "v1beta2",
		Kind:       "VirtualPrivateCloud",
		Metadata: networkingtypes.RegionalMetadataResponse{
			Id:                name,
			CreationTimestamp: time.Now(),
			Generation:        1,
			Region:            &region,
			Uid:               uid,
		},
		Spec: networkingtypes.VirtualPrivateCloudSpec{
			Ipv4CidrBlocks: &cidrs,
			StackType:      &stackType,
		},
		Status: networkingtypes.VirtualPrivateCloudStatus{
			Conditions: &conds,
		},
	}
}

func setupVPCHandlers(ms *mockServer, name string) {
	vpc := mockVPC(name)
	resourcePath := fmt.Sprintf("/networking/v1beta2/projects/test-project/regions/se-sto/virtualPrivateClouds/%s", name)
	ms.mux.HandleFunc("/networking/v1beta2/projects/test-project/regions/se-sto/virtualPrivateClouds", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			respondJSON(w, http.StatusCreated, vpc)
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
			respondJSON(w, http.StatusOK, vpc)
		case http.MethodDelete:
			ms.markDeleted(resourcePath)
			w.WriteHeader(http.StatusNoContent)
		case http.MethodPatch:
			respondJSON(w, http.StatusOK, vpc)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

func mockSubnet(name string) *networkingtypes.Subnet {
	conds := []networkingtypes.SubnetStatusConditionsItem{
		{Type: "Ready", Status: "True"},
	}
	region := "se-sto"
	uid := openapi_types.UUID{}
	cidr := "10.0.1.0/24"
	zone := "a"
	return &networkingtypes.Subnet{
		ApiVersion: "v1beta2",
		Kind:       "Subnet",
		Metadata: networkingtypes.RegionalMetadataResponse{
			Id:                name,
			CreationTimestamp: time.Now(),
			Generation:        1,
			Region:            &region,
			Uid:               uid,
		},
		Spec: networkingtypes.SubnetSpec{
			VpcRef:        "/networking/projects/test-project/regions/se-sto/virtualPrivateClouds/test-vpc",
			Ipv4CidrBlock: &cidr,
			StackType:     networkingtypes.SubnetSpecStackType("dual-stack"),
			Placement:     networkingtypes.SubnetSpecPlacement{Zone: &zone},
		},
		Status: networkingtypes.SubnetStatus{
			Conditions: &conds,
		},
	}
}

func setupSubnetHandlers(ms *mockServer, name string) {
	subnet := mockSubnet(name)
	resourcePath := fmt.Sprintf("/networking/v1beta2/projects/test-project/regions/se-sto/subnets/%s", name)
	ms.mux.HandleFunc("/networking/v1beta2/projects/test-project/regions/se-sto/subnets", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			respondJSON(w, http.StatusCreated, subnet)
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
			respondJSON(w, http.StatusOK, subnet)
		case http.MethodDelete:
			ms.markDeleted(resourcePath)
			w.WriteHeader(http.StatusNoContent)
		case http.MethodPatch:
			respondJSON(w, http.StatusOK, subnet)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}
