// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	lbtypes "github.com/evroc-oss/evroc-go-sdk/types/loadbalancer"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func ipProtocolPtr(v lbtypes.BackendserviceSpecIpProtocolSelection) *lbtypes.BackendserviceSpecIpProtocolSelection {
	return &v
}

func mockLoadBalancer(name string) *lbtypes.Loadbalancer {
	conds := []lbtypes.LoadbalancerStatusConditionsItem{
		{Type: "Ready", Status: "True"},
	}
	region := "se-sto"
	uid := openapi_types.UUID{}
	ip := "203.0.113.10"
	port := int32(80)
	protocol := lbtypes.LoadbalancerSpecListenersItemProtocol("TCP")
	listeners := []lbtypes.LoadbalancerSpecListenersItem{
		{Port: port, Protocol: protocol, RouteRefs: &[]string{"/loadbalancer/projects/test-project/regions/se-sto/l4Routes/test-route"}},
	}
	return &lbtypes.Loadbalancer{
		ApiVersion: "v1alpha1",
		Kind:       "Loadbalancer",
		Metadata: lbtypes.RegionalMetadataResponse{
			Id:                name,
			CreationTimestamp: time.Now(),
			Generation:        1,
			Region:            &region,
			Uid:               uid,
		},
		Spec: lbtypes.LoadbalancerSpec{
			PublicIPRef: "/networking/projects/test-project/regions/se-sto/publicIPs/test-pip",
			Listeners:   &listeners,
		},
		Status: lbtypes.LoadbalancerStatus{
			Conditions: &conds,
			Networking: &lbtypes.LoadbalancerStatusNetworking{
				PublicIPv4Address: &ip,
			},
		},
	}
}

func setupLoadBalancerHandlers(ms *mockServer, name string) {
	lb := mockLoadBalancer(name)
	resourcePath := fmt.Sprintf("/loadbalancer/v1alpha1/projects/test-project/regions/se-sto/loadBalancers/%s", name)
	ms.mux.HandleFunc("/loadbalancer/v1alpha1/projects/test-project/regions/se-sto/loadBalancers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			respondJSON(w, http.StatusCreated, lb)
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
			respondJSON(w, http.StatusOK, lb)
		case http.MethodDelete:
			ms.markDeleted(resourcePath)
			w.WriteHeader(http.StatusNoContent)
		case http.MethodPatch:
			respondJSON(w, http.StatusOK, lb)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

func mockBackendPool(name string) *lbtypes.Backendpool {
	region := "se-sto"
	uid := openapi_types.UUID{}
	refs := []string{"/compute/projects/test-project/regions/se-sto/virtualMachines/test-vm"}
	return &lbtypes.Backendpool{
		ApiVersion: "v1alpha1",
		Kind:       "Backendpool",
		Metadata: lbtypes.RegionalMetadataResponse{
			Id:                name,
			CreationTimestamp: time.Now(),
			Generation:        1,
			Region:            &region,
			Uid:               uid,
		},
		Spec: lbtypes.BackendpoolSpec{
			BackendRefs: &refs,
		},
	}
}

func setupBackendPoolHandlers(ms *mockServer, name string) {
	pool := mockBackendPool(name)
	resourcePath := fmt.Sprintf("/loadbalancer/v1alpha1/projects/test-project/regions/se-sto/backendPools/%s", name)
	ms.mux.HandleFunc("/loadbalancer/v1alpha1/projects/test-project/regions/se-sto/backendPools", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			respondJSON(w, http.StatusCreated, pool)
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
			respondJSON(w, http.StatusOK, pool)
		case http.MethodDelete:
			ms.markDeleted(resourcePath)
			w.WriteHeader(http.StatusNoContent)
		case http.MethodPatch:
			var patch lbtypes.Backendpool
			if err := json.NewDecoder(r.Body).Decode(&patch); err == nil {
				if patch.Spec.BackendRefs != nil {
					pool.Spec.BackendRefs = patch.Spec.BackendRefs
				}
			}
			respondJSON(w, http.StatusOK, pool)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

func mockBackendService(name string) *lbtypes.Backendservice {
	conds := []lbtypes.BackendserviceStatusConditionsItem{
		{Type: "Ready", Status: "True"},
	}
	region := "se-sto"
	uid := openapi_types.UUID{}
	poolRef := "/loadbalancer/projects/test-project/regions/se-sto/backendPools/test-pool"
	return &lbtypes.Backendservice{
		ApiVersion: "v1alpha1",
		Kind:       "Backendservice",
		Metadata: lbtypes.RegionalMetadataResponse{
			Id:                name,
			CreationTimestamp: time.Now(),
			Generation:        1,
			Region:            &region,
			Uid:               uid,
		},
		Spec: lbtypes.BackendserviceSpec{
			Port:                80,
			BackendPoolRef:      &poolRef,
			IpProtocolSelection: ipProtocolPtr(lbtypes.IPv4),
		},
		Status: lbtypes.BackendserviceStatus{
			Conditions: &conds,
			Backends: &[]lbtypes.BackendserviceStatusBackendsItem{
				{Name: "test-backend", Zone: "se-sto-1a", Address: "10.0.0.1"},
			},
		},
	}
}

func setupBackendServiceHandlers(ms *mockServer, name string) {
	svc := mockBackendService(name)
	resourcePath := fmt.Sprintf("/loadbalancer/v1alpha1/projects/test-project/regions/se-sto/backendServices/%s", name)
	ms.mux.HandleFunc("/loadbalancer/v1alpha1/projects/test-project/regions/se-sto/backendServices", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			respondJSON(w, http.StatusCreated, svc)
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
			respondJSON(w, http.StatusOK, svc)
		case http.MethodDelete:
			ms.markDeleted(resourcePath)
			w.WriteHeader(http.StatusNoContent)
		case http.MethodPatch:
			respondJSON(w, http.StatusOK, svc)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

func mockL4Route(name string) *lbtypes.L4route {
	conds := []lbtypes.L4routeStatusConditionsItem{
		{Type: "Ready", Status: "True"},
	}
	region := "se-sto"
	uid := openapi_types.UUID{}
	svcRef := "/loadbalancer/projects/test-project/regions/se-sto/backendServices/test-svc"
	return &lbtypes.L4route{
		ApiVersion: "v1alpha1",
		Kind:       "L4route",
		Metadata: lbtypes.RegionalMetadataResponse{
			Id:                name,
			CreationTimestamp: time.Now(),
			Generation:        1,
			Region:            &region,
			Uid:               uid,
		},
		Spec: lbtypes.L4routeSpec{
			DefaultBackendServiceRef: svcRef,
		},
		Status: lbtypes.L4routeStatus{
			Conditions: &conds,
		},
	}
}

func setupL4RouteHandlers(ms *mockServer, name string) {
	route := mockL4Route(name)
	resourcePath := fmt.Sprintf("/loadbalancer/v1alpha1/projects/test-project/regions/se-sto/l4Routes/%s", name)
	ms.mux.HandleFunc("/loadbalancer/v1alpha1/projects/test-project/regions/se-sto/l4Routes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			respondJSON(w, http.StatusCreated, route)
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
			respondJSON(w, http.StatusOK, route)
		case http.MethodDelete:
			ms.markDeleted(resourcePath)
			w.WriteHeader(http.StatusNoContent)
		case http.MethodPatch:
			respondJSON(w, http.StatusOK, route)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}
