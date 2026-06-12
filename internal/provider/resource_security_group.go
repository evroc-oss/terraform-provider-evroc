// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"time"

	evroc "github.com/evroc-oss/evroc-go-sdk"
	networkingtypes "github.com/evroc-oss/evroc-go-sdk/types/networking"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// securityGroupRuleResource returns the schema for a security group rule
// This is used for both the schema definition and hash function
func securityGroupRuleResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the rule.",
			},
			"direction": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateSecurityGroupDirection(),
				Description:      "Direction of traffic: 'Ingress' or 'Egress'.",
			},
			"protocol": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateSecurityGroupProtocol(),
				Description:      "Protocol: 'TCP', 'UDP', 'ICMP', or 'All'.",
			},
			"port": {
				Type:             schema.TypeInt,
				Optional:         true,
				ValidateDiagFunc: validatePort(),
				Description:      "Port number (0 for all ports). Only applicable when protocol is TCP or UDP.",
			},
			"end_port": {
				Type:             schema.TypeInt,
				Optional:         true,
				Computed:         true,
				ValidateDiagFunc: validatePort(),
				Description:      "End port for port ranges. Only applicable when protocol is TCP or UDP.",
			},
			"remote_ip": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateCIDR(),
				Description:      "Remote IP address or CIDR block. Mutually exclusive with remote_security_group and remote_subnet.",
			},
			"remote_security_group": {
				Type:             schema.TypeString,
				Optional:         true,
				DiffSuppressFunc: suppressFQIDDiff,
				Description:      "Name or FQID of a remote security group to allow traffic to/from. Mutually exclusive with remote_ip and remote_subnet.",
			},
			"remote_subnet": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Name of a remote subnet to allow traffic to/from. Mutually exclusive with remote_ip and remote_security_group.",
			},
		},
	}
}

func resourceSecurityGroup() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an evroc security group resource for network firewall rules.",

		CreateContext: resourceSecurityGroupCreate,
		ReadContext:   resourceSecurityGroupRead,
		UpdateContext: resourceSecurityGroupUpdate,
		DeleteContext: resourceSecurityGroupDelete,

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
				Description: "Name of the security group. Must be unique within the project.",
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
				Description: "Region where the security group is created. Defaults to provider region.",
			},
			"rule": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "List of security group rules.",
				Set:         schema.HashResource(securityGroupRuleResource()),
				Elem:        securityGroupRuleResource(),
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
			"sg_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier (UUID) of the security group.",
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
				Description: "Timestamp when the security group was created (RFC3339 format).",
			},
			"fqid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified resource ID (FQID). Use this to reference this resource from other resources.",
			},
		},
	}
}

func resourceSecurityGroupCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)
	rules := expandSecurityGroupRules(client, d.Get("rule").(*schema.Set).List())

	// Get user labels
	var userLabels map[string]string
	if labels, ok := d.GetOk("user_labels"); ok {
		userLabels = make(map[string]string)
		for k, v := range labels.(map[string]interface{}) {
			userLabels[k] = v.(string)
		}
	}

	req := BuildSecurityGroupCreateRequest(name, rules, userLabels)

	sg, err := client.Networking().SecurityGroups().Create(ctx, req)
	if err != nil {
		return diag.Errorf("error creating security group %s: %s", name, err)
	}

	d.SetId(sg.Metadata.Id)
	d.Set("project", resolveProject(d, config))

	// Wait for security group to be ready and capture the ready resource
	timeout := d.Timeout(schema.TimeoutCreate)
	readySG, err := client.Networking().SecurityGroups().WaitForReady(ctx, name, timeout)
	if err != nil {
		return diag.Errorf("error waiting for security group %s to be ready: %s", name, err)
	}

	// Use the ready resource's identity
	d.SetId(readySG.Metadata.Id)

	return resourceSecurityGroupRead(ctx, d, meta)
}

func resourceSecurityGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	sg, err := client.Networking().SecurityGroups().Get(ctx, d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading security group: %s", err)
	}

	diags = setDiag(d, "name", sg.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "region", derefString(sg.Metadata.Region), diags)
	diags = setDiag(d, "sg_id", sg.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", sg.Metadata.CreationTimestamp.Format(time.RFC3339), diags)

	// Set rules
	if sg.Spec.Rules != nil {
		diags = setDiag(d, "rule", flattenSecurityGroupRules(*sg.Spec.Rules), diags)
	}

	if sg.Metadata.UserLabels != nil && len(*sg.Metadata.UserLabels) > 0 {
		diags = setDiag(d, "user_labels", flattenLabels(sg.Metadata.UserLabels), diags)
	}

	diags = setDiag(d, "system_labels", flattenLabels(sg.Metadata.SystemLabels), diags)
	diags = setDiag(d, "fqid", string(client.Networking().SecurityGroupRef(sg.Metadata.Id)), diags)

	return diags
}

func resourceSecurityGroupUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	if d.HasChanges("rule", "user_labels") {
		sg, err := client.Networking().SecurityGroups().Get(ctx, d.Id())
		if err != nil {
			return diag.Errorf("error reading security group %s: %s", d.Id(), err)
		}

		// Update rules
		if d.HasChange("rule") {
			rules := expandSecurityGroupRules(client, d.Get("rule").(*schema.Set).List())
			sg.Spec.Rules = &rules
		}

		// Update user labels
		if d.HasChange("user_labels") {
			if labels, ok := d.GetOk("user_labels"); ok {
				userLabels := make(networkingtypes.UserLabels)
				for k, v := range labels.(map[string]interface{}) {
					userLabels[k] = v.(string)
				}
				sg.Metadata.UserLabels = &userLabels
			} else {
				sg.Metadata.UserLabels = nil
			}
		}

		_, err = client.Networking().SecurityGroups().Patch(ctx, d.Id(), sg)
		if err != nil {
			return diag.Errorf("error updating security group %s: %s", d.Id(), err)
		}
	}

	return resourceSecurityGroupRead(ctx, d, meta)
}

func resourceSecurityGroupDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	err := client.Networking().SecurityGroups().Delete(ctx, d.Id())
	if err != nil && !isNotFoundError(err) {
		return diag.Errorf("error deleting security group %s: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}

func expandSecurityGroupRules(client *evroc.Client, rules []interface{}) []networkingtypes.SecurityGroupSpecRulesItem {
	result := make([]networkingtypes.SecurityGroupSpecRulesItem, 0, len(rules))

	for _, r := range rules {
		rule := r.(map[string]interface{})

		name := rule["name"].(string)
		direction := rule["direction"].(string)
		protocol := rule["protocol"].(string)

		// Skip null/empty entries produced by Terraform's set diff when a rule is removed
		if name == "" && direction == "" && protocol == "" {
			continue
		}

		sgRule := networkingtypes.SecurityGroupSpecRulesItem{
			Name:      &name,
			Direction: networkingtypes.SecurityGroupSpecRulesItemDirection(direction),
		}

		// Protocol
		proto := networkingtypes.SecurityGroupSpecRulesItemProtocol(protocol)
		sgRule.Protocol = &proto

		// Port
		if port, ok := rule["port"].(int); ok && port > 0 {
			port32 := int32(port)
			sgRule.Port = &port32
		}

		// End port
		if endPort, ok := rule["end_port"].(int); ok && endPort > 0 {
			endPort32 := int32(endPort)
			sgRule.EndPort = &endPort32
		}

		// Remote — mutually exclusive: address, security group, or subnet
		if remoteIP, ok := rule["remote_ip"].(string); ok && remoteIP != "" {
			sgRule.Remote.Address = &networkingtypes.SecurityGroupSpecRulesItemAddress{
				IpAddressOrCIDR: remoteIP,
			}
		}
		if remoteSG, ok := rule["remote_security_group"].(string); ok && remoteSG != "" {
			sgFQID := string(client.Networking().SecurityGroupRef(remoteSG))
			if isFQID(remoteSG) {
				sgFQID = remoteSG
			}
			sgRule.Remote.SecurityGroupRef = &sgFQID
		}
		if remoteSubnet, ok := rule["remote_subnet"].(string); ok && remoteSubnet != "" {
			sgRule.Remote.SubnetRef = &remoteSubnet
		}

		result = append(result, sgRule)
	}

	return result
}

func flattenSecurityGroupRules(rules []networkingtypes.SecurityGroupSpecRulesItem) []interface{} {
	result := make([]interface{}, 0, len(rules))

	for _, rule := range rules {
		r := map[string]interface{}{
			"name":      derefString(rule.Name),
			"direction": string(rule.Direction),
		}

		if rule.Protocol != nil && string(*rule.Protocol) != "" {
			r["protocol"] = string(*rule.Protocol)
		} else {
			r["protocol"] = "All"
		}

		if rule.Port != nil {
			r["port"] = int(*rule.Port)
		}

		// TCP/UDP: always set end_port (including 0) so Read matches optional-attribute
		// defaults in state (e.g. ImportStateVerify after single-port rules).
		if rule.Protocol != nil {
			switch *rule.Protocol {
			case networkingtypes.SecurityGroupSpecRulesItemProtocol("TCP"), networkingtypes.SecurityGroupSpecRulesItemProtocol("UDP"):
				if rule.EndPort != nil {
					r["end_port"] = int(*rule.EndPort)
				} else {
					r["end_port"] = 0
				}
			}
		}

		if rule.Remote.Address != nil {
			r["remote_ip"] = rule.Remote.Address.IpAddressOrCIDR
		}
		if rule.Remote.SecurityGroupRef != nil {
			r["remote_security_group"] = *rule.Remote.SecurityGroupRef
		}
		if rule.Remote.SubnetRef != nil {
			r["remote_subnet"] = *rule.Remote.SubnetRef
		}

		result = append(result, r)
	}

	return result
}
