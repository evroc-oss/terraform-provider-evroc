// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import (
	"context"
	"time"

	"github.com/evroc-oss/evroc-go-sdk/networking"
	networkingtypes "github.com/evroc-oss/evroc-go-sdk/types/networking"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceVPC() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an evroc Virtual Private Cloud (VPC) resource for network isolation.",

		CreateContext: resourceVPCCreate,
		ReadContext:   resourceVPCRead,
		UpdateContext: resourceVPCUpdate,
		DeleteContext: resourceVPCDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the VPC. Must be unique within the project.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Project this resource belongs to. Defaults to the provider project.",
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Computed:    true,
				Description: "Region where the VPC is created. Defaults to provider region.",
			},
			"ipv4_cidr_blocks": {
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    true,
				Description: "IPv4 CIDR blocks for the VPC. Must be within RFC 1918 ranges. Defaults to [\"10.0.0.0/16\"].",
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validateVPCCIDR(),
				},
			},
			"stack_type": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				Computed:         true,
				ValidateDiagFunc: validateVPCStackType(),
				Description:      "Stack type: 'dual-stack' (IPv4 + IPv6) or 'ipv6-only'. Defaults to dual-stack.",
			},
			"user_labels": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "User-defined labels (key/value pairs) for organizing and selecting resources.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			// Computed fields
			"vpc_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier (UUID) of the VPC.",
			},
			"assigned_ipv4_cidr_blocks": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "IPv4 CIDR blocks actually assigned to the VPC.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"assigned_ipv6_gua_block": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Publicly routable IPv6 range assigned to the VPC.",
			},
			"available_ipv4_cidr_blocks": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "IPv4 CIDR blocks still available (not allocated) in the VPC.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"subnets": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Subnets allocated within this VPC.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ipv4_cidr_block": {Type: schema.TypeString, Computed: true},
						"ipv6_gua_block":  {Type: schema.TypeString, Computed: true},
						"subnet_ref":      {Type: schema.TypeString, Computed: true},
						"zone":            {Type: schema.TypeString, Computed: true},
					},
				},
			},
			"system_labels": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "System-managed labels automatically set by evroc (read-only).",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the VPC was created (RFC3339 format).",
			},
			"fqid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified resource ID (FQID). Use this to reference this resource from other resources.",
			},
		},
	}
}

func resourceVPCCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)

	builder := networking.NewVPCBuilder(name)

	if cidrs, ok := d.GetOk("ipv4_cidr_blocks"); ok {
		for _, c := range cidrs.([]interface{}) {
			builder = builder.WithIPv4CIDRBlock(c.(string))
		}
	}

	if st, ok := d.GetOk("stack_type"); ok {
		builder = builder.WithStackType(networkingtypes.VirtualPrivateCloudSpecStackType(st.(string)))
	}

	if labels, ok := d.GetOk("user_labels"); ok {
		userLabels := make(map[string]string)
		for k, v := range labels.(map[string]interface{}) {
			userLabels[k] = v.(string)
		}
		builder = builder.WithLabels(userLabels)
	}

	vpc, err := builder.Create(ctx, client.Networking().VirtualPrivateClouds())
	if err != nil {
		return diag.Errorf("error creating VPC %s: %s", name, err)
	}

	d.SetId(vpc.Metadata.Id)
	d.Set("project", resolveProject(d, config))

	return resourceVPCRead(ctx, d, meta)
}

func resourceVPCRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	vpc, err := client.Networking().VirtualPrivateClouds().Get(ctx, d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading VPC: %s", err)
	}

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

	if vpc.Status.AssignedIPv4CidrBlocks != nil {
		diags = setDiag(d, "assigned_ipv4_cidr_blocks", *vpc.Status.AssignedIPv4CidrBlocks, diags)
	} else {
		diags = setDiag(d, "assigned_ipv4_cidr_blocks", []string{}, diags)
	}
	if vpc.Status.AssignedIPv6GUABlock != nil {
		diags = setDiag(d, "assigned_ipv6_gua_block", *vpc.Status.AssignedIPv6GUABlock, diags)
	}
	if vpc.Status.AvailableIPv4CidrBlocks != nil {
		diags = setDiag(d, "available_ipv4_cidr_blocks", *vpc.Status.AvailableIPv4CidrBlocks, diags)
	} else {
		diags = setDiag(d, "available_ipv4_cidr_blocks", []string{}, diags)
	}
	if vpc.Status.Subnets != nil && vpc.Status.Subnets.AllocatedSubnets != nil {
		subnets := make([]map[string]interface{}, len(*vpc.Status.Subnets.AllocatedSubnets))
		for i, s := range *vpc.Status.Subnets.AllocatedSubnets {
			m := map[string]interface{}{}
			if s.Ipv4CidrBlock != nil {
				m["ipv4_cidr_block"] = *s.Ipv4CidrBlock
			}
			if s.Ipv6GUABlock != nil {
				m["ipv6_gua_block"] = *s.Ipv6GUABlock
			}
			if s.SubnetRef != nil {
				m["subnet_ref"] = *s.SubnetRef
			}
			if s.Placement != nil && s.Placement.Zone != nil {
				m["zone"] = *s.Placement.Zone
			}
			subnets[i] = m
		}
		diags = setDiag(d, "subnets", subnets, diags)
	} else {
		diags = setDiag(d, "subnets", []map[string]interface{}{}, diags)
	}

	if vpc.Metadata.UserLabels != nil && len(*vpc.Metadata.UserLabels) > 0 {
		diags = setDiag(d, "user_labels", flattenLabels(vpc.Metadata.UserLabels), diags)
	}

	diags = setDiag(d, "system_labels", flattenLabels(vpc.Metadata.SystemLabels), diags)
	diags = setDiag(d, "fqid", client.Networking().VPCRef(vpc.Metadata.Id), diags)

	return diags
}

func resourceVPCUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	if d.HasChange("user_labels") {
		vpc, err := client.Networking().VirtualPrivateClouds().Get(ctx, d.Id())
		if err != nil {
			return diag.Errorf("error reading VPC %s: %s", d.Id(), err)
		}

		if labels, ok := d.GetOk("user_labels"); ok {
			userLabels := make(networkingtypes.UserLabels)
			for k, v := range labels.(map[string]interface{}) {
				userLabels[k] = v.(string)
			}
			vpc.Metadata.UserLabels = &userLabels
		} else {
			vpc.Metadata.UserLabels = nil
		}

		if _, err := client.Networking().VirtualPrivateClouds().Patch(ctx, d.Id(), vpc); err != nil {
			return diag.Errorf("error updating VPC %s: %s", d.Id(), err)
		}
	}

	return resourceVPCRead(ctx, d, meta)
}

func resourceVPCDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	err := client.Networking().VirtualPrivateClouds().Delete(ctx, d.Id())
	if err != nil {
		if !isNotFoundError(err) {
			return diag.Errorf("error deleting VPC %s: %s", d.Id(), err)
		}
		d.SetId("")
		return nil
	}

	timeout := d.Timeout(schema.TimeoutDelete)
	if err := client.Networking().VirtualPrivateClouds().WaitForDeleted(ctx, d.Id(), timeout); err != nil {
		return diag.Errorf("error waiting for VPC %s to be deleted: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}
