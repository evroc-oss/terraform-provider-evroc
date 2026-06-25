// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVPC() *schema.Resource {
	return &schema.Resource{
		Description: "Get information about an existing evroc VPC.",
		ReadContext: dataSourceVPCRead,
		Schema: map[string]*schema.Schema{
			"name":             {Type: schema.TypeString, Required: true, Description: "Name of the VPC to look up."},
			"project":          {Type: schema.TypeString, Optional: true, Computed: true, Description: "Project this resource belongs to."},
			"vpc_id":           {Type: schema.TypeString, Computed: true, Description: "Unique identifier (UUID)."},
			"region":           {Type: schema.TypeString, Computed: true, Description: "Region of the VPC."},
			"ipv4_cidr_blocks": {Type: schema.TypeList, Computed: true, Elem: &schema.Schema{Type: schema.TypeString}, Description: "IPv4 CIDR blocks."},
			"stack_type":       {Type: schema.TypeString, Computed: true, Description: "Stack type (dual-stack or ipv6-only)."},
			"user_labels":      {Type: schema.TypeMap, Computed: true, Elem: &schema.Schema{Type: schema.TypeString}, Description: "User-defined labels."},
			"system_labels":    {Type: schema.TypeMap, Computed: true, Elem: &schema.Schema{Type: schema.TypeString}, Description: "System-managed labels."},
			"created_at":       {Type: schema.TypeString, Computed: true, Description: "Creation timestamp (RFC3339)."},
			"fqid":             {Type: schema.TypeString, Computed: true, Description: "Fully qualified resource ID."},
		},
	}
}

func dataSourceVPCRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}
	var diags diag.Diagnostics
	name := d.Get("name").(string)

	vpc, err := client.Networking().VirtualPrivateClouds().Get(ctx, name)
	if err != nil {
		return diag.Errorf("error reading VPC %s: %s", name, err)
	}

	d.SetId(vpc.Metadata.Id)
	diags = setDiag(d, "name", vpc.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "region", derefString(vpc.Metadata.Region), diags)
	diags = setDiag(d, "vpc_id", vpc.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", vpc.Metadata.CreationTimestamp.Format(time.RFC3339), diags)
	if vpc.Spec.Ipv4CidrBlocks != nil {
		diags = setDiag(d, "ipv4_cidr_blocks", *vpc.Spec.Ipv4CidrBlocks, diags)
	}
	if vpc.Spec.StackType != nil {
		diags = setDiag(d, "stack_type", string(*vpc.Spec.StackType), diags)
	}
	diags = setDiag(d, "user_labels", flattenLabels(vpc.Metadata.UserLabels), diags)
	diags = setDiag(d, "system_labels", flattenLabels(vpc.Metadata.SystemLabels), diags)
	diags = setDiag(d, "fqid", client.Networking().VPCRef(vpc.Metadata.Id), diags)

	return diags
}
