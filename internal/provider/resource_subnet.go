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

func resourceSubnet() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an evroc subnet resource within a VPC.",

		CreateContext: resourceSubnetCreate,
		ReadContext:   resourceSubnetRead,
		UpdateContext: resourceSubnetUpdate,
		DeleteContext: resourceSubnetDelete,

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
				Description: "Name of the subnet. Must be unique within the project.",
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
				Description: "Region where the subnet is created. Defaults to provider region.",
			},
			"vpc_ref": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				DiffSuppressFunc: suppressFQIDDiff,
				Description:      "Reference to the VPC this subnet belongs to. Accepts FQID or plain name.",
			},
			"ipv4_cidr_block": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				ValidateDiagFunc: validateSubnetCIDR(),
				Description:      "IPv4 CIDR block for the subnet. Must be within the VPC's CIDR range, prefix /16 to /29. Required when stack_type is 'dual-stack' (default).",
			},
			"stack_type": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				Computed:         true,
				ValidateDiagFunc: validateVPCStackType(),
				Description:      "Stack type: 'dual-stack' or 'ipv6-only'. Defaults to dual-stack.",
			},
			"zone": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validateZone(),
				Description:      "Zone for the subnet (e.g., a, b, c).",
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
			"subnet_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier (UUID) of the subnet.",
			},
			"assigned_ipv6_gua_block": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Publicly routable IPv6 GUA block assigned to the subnet.",
			},
			"available_ipv4_address_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Count of unassigned IPv4 addresses in the subnet.",
			},
			"available_ipv4_ranges": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"using_ipv4_address_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Count of IPv4 addresses currently in use.",
			},
			"using_ipv4_ranges": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"using_ipv6_address_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Count of IPv6 addresses currently in use.",
			},
			"using_ipv6_ranges": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"attachment_refs": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "References to resources attached to this subnet.",
				Elem:        &schema.Schema{Type: schema.TypeString},
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
				Description: "Timestamp when the subnet was created (RFC3339 format).",
			},
			"fqid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified resource ID (FQID). Use this to reference this resource from other resources.",
			},
		},
	}
}

func resourceSubnetCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)
	vpcRef := d.Get("vpc_ref").(string)

	vpcFQID := client.Networking().VPCRef(vpcRef)
	if isFQID(vpcRef) {
		vpcFQID = vpcRef
	}

	builder := networking.NewSubnetBuilder(name).
		WithVPCRef(vpcFQID).
		WithZone(d.Get("zone").(string))

	stackType := ""
	if st, ok := d.GetOk("stack_type"); ok {
		stackType = st.(string)
	}

	cidr := ""
	if c, ok := d.GetOk("ipv4_cidr_block"); ok {
		cidr = c.(string)
	}

	if (stackType == "" || stackType == "dual-stack") && cidr == "" {
		return diag.Errorf("ipv4_cidr_block is required when stack_type is 'dual-stack' (default)")
	}
	if stackType == "ipv6-only" && cidr != "" {
		return diag.Errorf("ipv4_cidr_block must not be set when stack_type is 'ipv6-only'")
	}

	if cidr != "" {
		builder = builder.WithIPv4CIDRBlock(cidr)
	}

	if st, ok := d.GetOk("stack_type"); ok {
		builder = builder.WithStackType(networkingtypes.SubnetSpecStackType(st.(string)))
	}

	if labels, ok := d.GetOk("user_labels"); ok {
		userLabels := make(map[string]string)
		for k, v := range labels.(map[string]interface{}) {
			userLabels[k] = v.(string)
		}
		builder = builder.WithLabels(userLabels)
	}

	subnet, err := builder.Create(ctx, client.Networking().Subnets())
	if err != nil {
		return diag.Errorf("error creating subnet %s: %s", name, err)
	}

	d.SetId(subnet.Metadata.Id)
	d.Set("project", resolveProject(d, config))

	return resourceSubnetRead(ctx, d, meta)
}

func resourceSubnetRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	subnet, err := client.Networking().Subnets().Get(ctx, d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading subnet: %s", err)
	}

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

	if subnet.Status.AssignedIPv6GUABlock != nil {
		diags = setDiag(d, "assigned_ipv6_gua_block", *subnet.Status.AssignedIPv6GUABlock, diags)
	} else {
		diags = setDiag(d, "assigned_ipv6_gua_block", "", diags)
	}
	if subnet.Status.AvailableIPv4AddressCount != nil {
		diags = setDiag(d, "available_ipv4_address_count", *subnet.Status.AvailableIPv4AddressCount, diags)
	} else {
		diags = setDiag(d, "available_ipv4_address_count", 0, diags)
	}
	if subnet.Status.AvailableIPv4Ranges != nil {
		diags = setDiag(d, "available_ipv4_ranges", *subnet.Status.AvailableIPv4Ranges, diags)
	} else {
		diags = setDiag(d, "available_ipv4_ranges", []string{}, diags)
	}
	if subnet.Status.UsingIPv4AddressCount != nil {
		diags = setDiag(d, "using_ipv4_address_count", *subnet.Status.UsingIPv4AddressCount, diags)
	}
	if subnet.Status.UsingIPv4Ranges != nil {
		diags = setDiag(d, "using_ipv4_ranges", *subnet.Status.UsingIPv4Ranges, diags)
	} else {
		diags = setDiag(d, "using_ipv4_ranges", []string{}, diags)
	}
	if subnet.Status.UsingIPv6AddressCount != nil {
		diags = setDiag(d, "using_ipv6_address_count", *subnet.Status.UsingIPv6AddressCount, diags)
	}
	if subnet.Status.UsingIPv6Ranges != nil {
		diags = setDiag(d, "using_ipv6_ranges", *subnet.Status.UsingIPv6Ranges, diags)
	} else {
		diags = setDiag(d, "using_ipv6_ranges", []string{}, diags)
	}
	if subnet.Status.AttachmentRefs != nil {
		diags = setDiag(d, "attachment_refs", *subnet.Status.AttachmentRefs, diags)
	} else {
		diags = setDiag(d, "attachment_refs", []string{}, diags)
	}

	if subnet.Metadata.UserLabels != nil && len(*subnet.Metadata.UserLabels) > 0 {
		diags = setDiag(d, "user_labels", flattenLabels(subnet.Metadata.UserLabels), diags)
	}

	diags = setDiag(d, "system_labels", flattenLabels(subnet.Metadata.SystemLabels), diags)
	// No SubnetRef helper in SDK — construct manually
	diags = setDiag(d, "fqid", "/networking/projects/"+resolveProject(d, config)+"/regions/"+derefString(subnet.Metadata.Region)+"/subnets/"+subnet.Metadata.Id, diags)

	return diags
}

func resourceSubnetUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	if d.HasChange("user_labels") {
		subnet, err := client.Networking().Subnets().Get(ctx, d.Id())
		if err != nil {
			return diag.Errorf("error reading subnet %s: %s", d.Id(), err)
		}

		if labels, ok := d.GetOk("user_labels"); ok {
			userLabels := make(networkingtypes.UserLabels)
			for k, v := range labels.(map[string]interface{}) {
				userLabels[k] = v.(string)
			}
			subnet.Metadata.UserLabels = &userLabels
		} else {
			subnet.Metadata.UserLabels = nil
		}

		if _, err := client.Networking().Subnets().Patch(ctx, d.Id(), subnet); err != nil {
			return diag.Errorf("error updating subnet %s: %s", d.Id(), err)
		}
	}

	return resourceSubnetRead(ctx, d, meta)
}

func resourceSubnetDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	err := client.Networking().Subnets().Delete(ctx, d.Id())
	if err != nil {
		if !isNotFoundError(err) {
			return diag.Errorf("error deleting subnet %s: %s", d.Id(), err)
		}
		d.SetId("")
		return nil
	}

	timeout := d.Timeout(schema.TimeoutDelete)
	if err := client.Networking().Subnets().WaitForDeleted(ctx, d.Id(), timeout); err != nil {
		return diag.Errorf("error waiting for subnet %s to be deleted: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}
