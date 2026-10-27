// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccEvrocRoles(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "evroc_roles" "test" {}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.evroc_roles.test", "roles.#"),
				),
			},
		},
	})
}

func TestDataSourceRolesResourceRestrictions(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	var resources []interface{}
	ms.mux.HandleFunc("/iam/v1beta2/roles", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusOK, map[string]interface{}{"items": []interface{}{
			map[string]interface{}{"id": "/iam/roles/computeOperator", "scope": "project", "resources": resources},
		}})
	})
	config := newTestProviderConfig(t, ms.server.URL)
	d := newTestResourceData(t, dataSourceRoles())
	for _, want := range [][]interface{}{{"compute.virtualMachines", "compute.disks"}, {}, nil} {
		resources = want
		if diags := dataSourceRolesRead(context.Background(), d, config); diags.HasError() {
			t.Fatal(diags)
		}
		roles := d.Get("roles").([]interface{})
		if len(roles) != 1 {
			t.Fatalf("roles = %v", roles)
		}
		got := roles[0].(map[string]interface{})["resources"].([]interface{})
		if len(got) != len(want) || (len(want) > 0 && !reflect.DeepEqual(got, want)) {
			t.Errorf("resources = %v, want %v", got, want)
		}
	}
}
