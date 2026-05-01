// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceSecurityGroup() *schema.Resource {
	return &schema.Resource{
		Description: "Get information about an existing evroc security group.",

		ReadContext: dataSourceSecurityGroupRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the security group to look up.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Project this resource belongs to. Defaults to the provider project.",
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Region where the security group is located.",
			},
			// Computed fields
			"sg_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier (UUID) of the security group.",
			},
			"rule": {
				Type:        schema.TypeSet,
				Computed:    true,
				Description: "List of security group rules.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Name of the rule.",
						},
						"direction": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Direction of traffic.",
						},
						"protocol": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Protocol.",
						},
						"port": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Port number.",
						},
						"end_port": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "End port for port ranges.",
						},
						"remote_ip": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Remote IP address or CIDR block.",
						},
					},
				},
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the security group was created (RFC3339 format).",
			},
			"fqid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified resource ID (FQID).",
			},
		},
	}
}

func dataSourceSecurityGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	var diags diag.Diagnostics

	name := d.Get("name").(string)

	sg, err := client.Networking().SecurityGroups().Get(ctx, name)
	if err != nil {
		return diag.Errorf("error reading security group %s: %s", name, err)
	}

	d.SetId(sg.Metadata.Id)
	diags = setDiag(d, "name", sg.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "region", derefString(sg.Metadata.Region), diags)
	diags = setDiag(d, "sg_id", sg.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", sg.Metadata.CreationTimestamp.Format(time.RFC3339), diags)

	// Set rules
	if sg.Spec.Rules != nil {
		diags = setDiag(d, "rule", flattenSecurityGroupRules(*sg.Spec.Rules), diags)
	}

	diags = setDiag(d, "fqid", string(client.Networking().SecurityGroupRef(sg.Metadata.Id)), diags)

	return diags
}
