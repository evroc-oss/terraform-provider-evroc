// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVirtualMachineConsoleLogs() *schema.Resource {
	return &schema.Resource{
		Description: "Reads the last 1,000 VM console log lines for explicit troubleshooting. Logs are sensitive and are stored in Terraform state; protect state access.",
		Schema: map[string]*schema.Schema{
			"name":    {Type: schema.TypeString, Required: true, Description: "Virtual machine name."},
			"project": {Type: schema.TypeString, Optional: true, Computed: true, Description: "Project containing the VM. Defaults to the provider project."},
			"region":  {Type: schema.TypeString, Computed: true, Description: "Region selected by the provider configuration."},
			"logs":    {Type: schema.TypeList, Computed: true, Sensitive: true, Description: "Console log lines, in API order.", Elem: &schema.Schema{Type: schema.TypeString}},
		},
		ReadContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
			client, diags := resolveClient(d, meta.(*ProviderConfig))
			if diags.HasError() {
				return diags
			}
			name := d.Get("name").(string)
			result, err := client.Compute().VirtualMachines().ConsoleLogs(ctx, name)
			if err != nil {
				return diag.FromErr(err)
			}
			d.SetId(client.Compute().VMRef(name).String())
			diags = setDiag(d, "project", client.DefaultProject(), diags)
			diags = setDiag(d, "region", client.DefaultRegion(), diags)
			return setDiag(d, "logs", result.Logs, diags)
		},
	}
}
