// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceSubnet() *schema.Resource {
	return &schema.Resource{
		Description: "Get information about an existing evroc subnet.",
		ReadContext: dataSourceSubnetRead,
		Schema: map[string]*schema.Schema{
			"name":            {Type: schema.TypeString, Required: true, Description: "Name of the subnet to look up."},
			"project":         {Type: schema.TypeString, Optional: true, Computed: true, Description: "Project this resource belongs to."},
			"subnet_id":       {Type: schema.TypeString, Computed: true, Description: "Unique identifier (UUID)."},
			"region":          {Type: schema.TypeString, Computed: true, Description: "Region of the subnet."},
			"vpc_ref":         {Type: schema.TypeString, Computed: true, Description: "Reference to the parent VPC."},
			"ipv4_cidr_block": {Type: schema.TypeString, Computed: true, Description: "IPv4 CIDR block."},
			"stack_type":      {Type: schema.TypeString, Computed: true, Description: "Stack type (dual-stack or ipv6-only)."},
			"zone":            {Type: schema.TypeString, Computed: true, Description: "Zone of the subnet."},
			"user_labels":     {Type: schema.TypeMap, Computed: true, Elem: &schema.Schema{Type: schema.TypeString}, Description: "User-defined labels."},
			"system_labels":   {Type: schema.TypeMap, Computed: true, Elem: &schema.Schema{Type: schema.TypeString}, Description: "System-managed labels."},
			"created_at":      {Type: schema.TypeString, Computed: true, Description: "Creation timestamp (RFC3339)."},
			"fqid":            {Type: schema.TypeString, Computed: true, Description: "Fully qualified resource ID."},
		},
	}
}

func dataSourceSubnetRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}
	var diags diag.Diagnostics
	name := d.Get("name").(string)

	subnet, err := client.Networking().Subnets().Get(ctx, name)
	if err != nil {
		return diag.Errorf("error reading subnet %s: %s", name, err)
	}

	d.SetId(subnet.Metadata.Id)
	diags = setDiag(d, "name", subnet.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "region", derefString(subnet.Metadata.Region), diags)
	diags = setDiag(d, "subnet_id", subnet.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", subnet.Metadata.CreationTimestamp.Format(time.RFC3339), diags)
	diags = setDiag(d, "vpc_ref", subnet.Spec.VpcRef, diags)
	diags = setDiag(d, "stack_type", string(subnet.Spec.StackType), diags)
	if subnet.Spec.Ipv4CidrBlock != nil {
		diags = setDiag(d, "ipv4_cidr_block", *subnet.Spec.Ipv4CidrBlock, diags)
	}
	if subnet.Spec.Placement.Zone != nil {
		diags = setDiag(d, "zone", *subnet.Spec.Placement.Zone, diags)
	}
	diags = setDiag(d, "user_labels", flattenLabels(subnet.Metadata.UserLabels), diags)
	diags = setDiag(d, "system_labels", flattenLabels(subnet.Metadata.SystemLabels), diags)
	diags = setDiag(d, "fqid", "/networking/projects/"+resolveProject(d, config)+"/regions/"+derefString(subnet.Metadata.Region)+"/subnets/"+subnet.Metadata.Id, diags)

	return diags
}
