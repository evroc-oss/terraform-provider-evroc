// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccEvrocThinkSharedModels(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{{
			Config: `data "evroc_think_shared_models" "test" {}`,
			Check:  resource.TestCheckResourceAttrSet("data.evroc_think_shared_models.test", "models.#"),
		}},
	})
}

func TestThinkSharedModelsDataSource(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	ms.mux.HandleFunc("/think/v1beta2/projects/test-project/sharedModels", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, 200, map[string]interface{}{"items": []interface{}{map[string]interface{}{"metadata": map[string]string{"id": "model"}, "spec": map[string]interface{}{"sunset": "2027-01-01T00:00:00Z", "aliases": []string{"chat"}}}}})
	})
	res := dataSourceThinkSharedModels()
	d := schema.TestResourceDataRaw(t, res.Schema, nil)
	if diags := res.ReadContext(context.Background(), d, newTestProviderConfig(t, ms.server.URL)); diags.HasError() {
		t.Fatal(diags)
	}
	assertField(t, d, "models.0.sunset", "2027-01-01T00:00:00Z")
}
