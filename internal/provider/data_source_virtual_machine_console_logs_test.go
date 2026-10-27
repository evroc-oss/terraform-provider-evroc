// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccEvrocVirtualMachineConsoleLogs(t *testing.T) {
	suffix := time.Now().UnixNano()
	config := testAccEvrocVirtualMachineConfig_basic(fmt.Sprintf("tf-console-%d", suffix), fmt.Sprintf("tf-console-disk-%d", suffix)) + `
data "evroc_virtual_machine_console_logs" "test" { name = evroc_virtual_machine.test.name }
`
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckEvrocVirtualMachineDestroy,
		Steps: []resource.TestStep{{
			Config: config,
			Check:  resource.TestCheckResourceAttrSet("data.evroc_virtual_machine_console_logs.test", "logs.#"),
		}},
	})
}

func TestVirtualMachineConsoleLogsDataSource(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	ms.mux.HandleFunc("/compute/v1beta2/projects/test-project/regions/se-sto/virtualMachines/vm/logs/console", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, 200, map[string]interface{}{"logs": []string{"boot complete"}})
	})
	res := dataSourceVirtualMachineConsoleLogs()
	d := schema.TestResourceDataRaw(t, res.Schema, map[string]interface{}{"name": "vm"})
	if diags := res.ReadContext(context.Background(), d, newTestProviderConfig(t, ms.server.URL)); diags.HasError() {
		t.Fatal(diags)
	}
	assertField(t, d, "logs.0", "boot complete")
}
