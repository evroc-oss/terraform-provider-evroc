// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"path"
	"time"

	evroc "github.com/evroc-oss/evroc-go-sdk"
	"github.com/evroc-oss/evroc-go-sdk/compute"
	computetypes "github.com/evroc-oss/evroc-go-sdk/types/compute"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceVirtualMachine() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an evroc virtual machine resource. A boot disk is required at creation time. " +
			"Additional data disks can be attached at creation via `data_disks` or to a running VM using the `evroc_hotswap_disk_attachment` resource.",

		CreateContext: resourceVirtualMachineCreate,
		ReadContext:   resourceVirtualMachineRead,
		UpdateContext: resourceVirtualMachineUpdate,
		DeleteContext: resourceVirtualMachineDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(15 * time.Minute),
			Update: schema.DefaultTimeout(15 * time.Minute),
			Delete: schema.DefaultTimeout(15 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the virtual machine. Must be unique within the project.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Project this resource belongs to. Defaults to the provider project.",
			},
			"flavor": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "VM flavor/size (e.g., a1a.s, a1a.m, a1a.l, c1a.s). Changing this will stop the VM, resize, and restart it.",
			},
			"boot_disk": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				DiffSuppressFunc: suppressFQIDDiff,
				Description:      "Name of the disk to use as boot disk. Accepts FQID or plain name.",
			},
			"data_disks": {
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    true,
				Description: "List of additional data disks to attach at creation time. Accepts FQIDs or plain names.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"ssh_keys": {
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    true,
				Description: "List of SSH public keys to inject into the VM.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"cloud_config_user_data": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Cloud-init user data script.",
			},
			"public_ip": {
				Type:             schema.TypeString,
				Optional:         true,
				DiffSuppressFunc: suppressFQIDDiff,
				Description:      "Name of the public IP to attach to the VM. Accepts FQID or plain name.",
			},
			"security_groups": {
				Type:        schema.TypeSet,
				Optional:    true,
				Set:         securityGroupHash,
				Description: "Set of security group names to attach to the VM. Accepts FQIDs or plain names.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Computed:    true,
				Description: "Region where the VM is created. Defaults to provider region.",
			},
			"zone": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validateZone(),
				Description:      "Zone for the VM (e.g., a, b, c).",
			},
			"placement_group": {
				Type:             schema.TypeString,
				Optional:         true,
				DiffSuppressFunc: suppressFQIDDiff,
				Description:      "Placement group name for VM placement control (e.g., for HA configurations). Accepts FQID or plain name.",
			},
			"running": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether the VM should be running. Set to false to stop the VM.",
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
			"vm_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier (UUID) of the virtual machine.",
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
				Description: "Timestamp when the VM was created (RFC3339 format).",
			},
			"fqid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified resource ID (FQID). Use this to reference this resource from other resources.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Current status of the virtual machine (e.g., Running, Stopped, Creating).",
			},
			"public_ipv4_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Assigned public IPv4 address of the VM.",
			},
			"private_ipv4_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Assigned private IPv4 address of the VM.",
			},
		},
	}
}

func resourceVirtualMachineCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)
	flavor := d.Get("flavor").(string)
	bootDisk := d.Get("boot_disk").(string)

	// Get optional fields
	var sshKeys []string
	if keys, ok := d.GetOk("ssh_keys"); ok {
		for _, key := range keys.([]interface{}) {
			sshKeys = append(sshKeys, key.(string))
		}
	}

	var userData string
	if ud, ok := d.GetOk("cloud_config_user_data"); ok {
		userData = ud.(string)
	}

	var publicIP string
	if pip, ok := d.GetOk("public_ip"); ok {
		publicIP = pip.(string)
	}

	var zone string
	if z, ok := d.GetOk("zone"); ok {
		zone = z.(string)
	}

	var securityGroups []string
	if sgs, ok := d.GetOk("security_groups"); ok {
		for _, sg := range sgs.(*schema.Set).List() {
			securityGroups = append(securityGroups, sg.(string))
		}
	}

	var placementGroup string
	if pg, ok := d.GetOk("placement_group"); ok {
		placementGroup = pg.(string)
	}

	var dataDisks []string
	if dds, ok := d.GetOk("data_disks"); ok {
		for _, dd := range dds.([]interface{}) {
			dataDisks = append(dataDisks, dd.(string))
		}
	}

	running := d.Get("running").(bool)

	// Get user labels
	var userLabels map[string]string
	if labels, ok := d.GetOk("user_labels"); ok {
		userLabels = make(map[string]string)
		for k, v := range labels.(map[string]interface{}) {
			userLabels[k] = v.(string)
		}
	}

	// Build VM create request using SDK builder
	req := BuildVirtualMachineCreateRequest(client, name, flavor, bootDisk, dataDisks, sshKeys, userData, publicIP, zone, securityGroups, placementGroup, running, userLabels)

	vm, err := client.Compute().VirtualMachines().Create(ctx, req)
	if err != nil {
		return diag.Errorf("error creating virtual machine %s: %s", name, err)
	}

	d.SetId(vm.Metadata.Id)
	d.Set("project", resolveProject(d, config))

	// Wait for VM to be ready and capture the ready resource
	timeout := d.Timeout(schema.TimeoutCreate)
	readyVM, err := client.Compute().VirtualMachines().WaitForReady(ctx, name, timeout)
	if err != nil {
		return diag.Errorf("error waiting for virtual machine %s to be ready: %s", name, err)
	}

	// Use the ready resource's identity
	d.SetId(readyVM.Metadata.Id)

	return resourceVirtualMachineRead(ctx, d, meta)
}

func resourceVirtualMachineRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	vm, err := client.Compute().VirtualMachines().Get(ctx, d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading virtual machine: %s", err)
	}

	diags = setVMBasicFields(d, vm, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "fqid", string(client.Compute().VMRef(vm.Metadata.Id)), diags)
	diags = setVMDiskFields(d, vm, diags)
	diags = setVMOSSettings(d, vm, diags)
	diags = setVMNetworking(d, vm, diags)
	diags = setVMPlacement(d, vm, diags)
	diags = setVMLabels(d, vm, diags)
	diags = setVMStatus(d, vm, diags)

	return diags
}

func setVMBasicFields(d *schema.ResourceData, vm *computetypes.VirtualMachine, diags diag.Diagnostics) diag.Diagnostics {
	diags = setDiag(d, "name", vm.Metadata.Id, diags)
	diags = setDiag(d, "region", derefString(vm.Metadata.Region), diags)
	diags = setDiag(d, "vm_id", vm.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", vm.Metadata.CreationTimestamp.Format(time.RFC3339), diags)
	// Normalize flavor path - API returns "/compute/global/computeProfiles/a1a.s"
	flavor := path.Base(vm.Spec.ComputeProfileRef)
	diags = setDiag(d, "flavor", flavor, diags)
	diags = setDiag(d, "running", vm.Spec.Running, diags)
	return diags
}

func setVMDiskFields(d *schema.ResourceData, vm *computetypes.VirtualMachine, diags diag.Diagnostics) diag.Diagnostics {
	if vm.Spec.Disks == nil {
		return diags
	}

	var dataDisks []string
	for _, diskRef := range *vm.Spec.Disks {
		if diskRef.BootFrom != nil && *diskRef.BootFrom {
			diags = setDiag(d, "boot_disk", diskRef.DiskRef, diags)
		} else {
			dataDisks = append(dataDisks, diskRef.DiskRef)
		}
	}

	if len(dataDisks) > 0 {
		diags = setDiag(d, "data_disks", dataDisks, diags)
	}

	return diags
}

func setVMOSSettings(d *schema.ResourceData, vm *computetypes.VirtualMachine, diags diag.Diagnostics) diag.Diagnostics {
	if vm.Spec.OsSettings == nil {
		return diags
	}

	if vm.Spec.OsSettings.Ssh != nil && vm.Spec.OsSettings.Ssh.AuthorizedKeys != nil {
		keys := make([]string, 0, len(*vm.Spec.OsSettings.Ssh.AuthorizedKeys))
		for _, key := range *vm.Spec.OsSettings.Ssh.AuthorizedKeys {
			keys = append(keys, key.Value)
		}
		if len(keys) > 0 {
			diags = setDiag(d, "ssh_keys", keys, diags)
		}
	}

	if vm.Spec.OsSettings.CloudInitUserData != nil {
		diags = setDiag(d, "cloud_config_user_data", *vm.Spec.OsSettings.CloudInitUserData, diags)
	}

	return diags
}

func setVMNetworking(d *schema.ResourceData, vm *computetypes.VirtualMachine, diags diag.Diagnostics) diag.Diagnostics {
	// Public IP ref — clear state when removed
	publicIPSet := false
	if vm.Spec.Networking != nil {
		if vm.Spec.Networking.PublicIPv4Address != nil && vm.Spec.Networking.PublicIPv4Address.Static != nil {
			if vm.Spec.Networking.PublicIPv4Address.Static.PublicIPRef != nil {
				diags = setDiag(d, "public_ip", *vm.Spec.Networking.PublicIPv4Address.Static.PublicIPRef, diags)
				publicIPSet = true
			}
		}

		// Security groups — clear state when all removed
		if vm.Spec.Networking.SecurityGroupSettings != nil && vm.Spec.Networking.SecurityGroupSettings.SecurityGroupMemberRefs != nil {
			sgs := append([]string{}, *vm.Spec.Networking.SecurityGroupSettings.SecurityGroupMemberRefs...)
			diags = setDiag(d, "security_groups", sgs, diags)
		} else {
			diags = setDiag(d, "security_groups", []string{}, diags)
		}
	} else {
		diags = setDiag(d, "security_groups", []string{}, diags)
	}
	if !publicIPSet {
		diags = setDiag(d, "public_ip", "", diags)
	}

	// Status networking — clear when absent
	if vm.Status.Networking != nil {
		if vm.Status.Networking.PublicIPv4Address != nil {
			diags = setDiag(d, "public_ipv4_address", *vm.Status.Networking.PublicIPv4Address, diags)
		} else {
			diags = setDiag(d, "public_ipv4_address", "", diags)
		}
		if vm.Status.Networking.PrivateIPv4Address != nil {
			diags = setDiag(d, "private_ipv4_address", *vm.Status.Networking.PrivateIPv4Address, diags)
		} else {
			diags = setDiag(d, "private_ipv4_address", "", diags)
		}
	} else {
		diags = setDiag(d, "public_ipv4_address", "", diags)
		diags = setDiag(d, "private_ipv4_address", "", diags)
	}

	return diags
}

func setVMPlacement(d *schema.ResourceData, vm *computetypes.VirtualMachine, diags diag.Diagnostics) diag.Diagnostics {
	if vm.Spec.Placement.Zone != nil {
		diags = setDiag(d, "zone", *vm.Spec.Placement.Zone, diags)
	}
	if vm.Spec.Placement.PlacementGroupRef != nil {
		diags = setDiag(d, "placement_group", *vm.Spec.Placement.PlacementGroupRef, diags)
	} else {
		diags = setDiag(d, "placement_group", "", diags)
	}
	return diags
}

func setVMLabels(d *schema.ResourceData, vm *computetypes.VirtualMachine, diags diag.Diagnostics) diag.Diagnostics {
	if vm.Metadata.UserLabels != nil && len(*vm.Metadata.UserLabels) > 0 {
		diags = setDiag(d, "user_labels", flattenLabels(vm.Metadata.UserLabels), diags)
	}

	diags = setDiag(d, "system_labels", flattenLabels(vm.Metadata.SystemLabels), diags)

	return diags
}

func setVMStatus(d *schema.ResourceData, vm *computetypes.VirtualMachine, diags diag.Diagnostics) diag.Diagnostics {
	if vm.Status.VirtualMachineStatus != nil {
		diags = setDiag(d, "status", *vm.Status.VirtualMachineStatus, diags)
	}
	return diags
}

func resourceVirtualMachineUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	vmService := client.Compute().VirtualMachines()
	needsStop := d.HasChange("flavor") || d.HasChange("placement_group")

	// Phase 1: Stop VM and wait if any change requires it
	if needsStop {
		stopUpdater := compute.UpdateVM(d.Id(), vmService).Stop()
		if _, err := stopUpdater.Apply(ctx); err != nil {
			return diag.Errorf("error stopping virtual machine %s: %s", d.Id(), err)
		}
		if _, err := vmService.WaitForStopped(ctx, d.Id(), d.Timeout(schema.TimeoutUpdate)); err != nil {
			return diag.Errorf("error waiting for virtual machine %s to stop: %s", d.Id(), err)
		}
	}

	// Phase 2: Apply all changes while VM is stopped (or live for non-stop changes)
	updater := compute.UpdateVM(d.Id(), vmService)
	hasChanges := false

	if d.HasChange("flavor") {
		updater = updater.Resize(d.Get("flavor").(string))
		hasChanges = true
	}

	if d.HasChange("public_ip") {
		oldPIP, newPIP := d.GetChange("public_ip")
		oldVal := oldPIP.(string)
		newVal := newPIP.(string)

		// The API requires removing the old IP before attaching a new one.
		if oldVal != "" && newVal != "" {
			removeUpdater := compute.UpdateVM(d.Id(), vmService).RemovePublicIP()
			if _, err := removeUpdater.Apply(ctx); err != nil {
				return diag.Errorf("error removing old public IP from virtual machine %s: %s", d.Id(), err)
			}
		}

		if newVal != "" {
			pipFQID := string(client.Networking().PublicIPRef(newVal))
			if isFQID(newVal) {
				pipFQID = newVal
			}
			updater = updater.SetPublicIP(pipFQID)
		} else {
			updater = updater.RemovePublicIP()
		}
		hasChanges = true
	}

	if d.HasChange("placement_group") {
		if pg, ok := d.GetOk("placement_group"); ok {
			pgVal := pg.(string)
			pgFQID := string(client.Compute().PlacementGroupRef(pgVal))
			if isFQID(pgVal) {
				pgFQID = pgVal
			}
			updater = updater.SetPlacementGroup(pgFQID)
		} else {
			updater = updater.RemovePlacementGroup()
		}
		hasChanges = true
	}

	if d.HasChange("security_groups") {
		updater = applySecurityGroupChanges(updater, d, client)
		hasChanges = true
	}

	if d.HasChange("user_labels") {
		updater = applyUserLabelChanges(updater, d)
		hasChanges = true
	}

	if hasChanges {
		if _, err := updater.Apply(ctx); err != nil {
			return diag.Errorf("error updating virtual machine %s: %s", d.Id(), err)
		}
		// Wait for the async update to settle before proceeding.
		// The API processes changes asynchronously, so the object's resource
		// version may still be changing when we try the next operation.
		if needsStop {
			if _, err := vmService.WaitForStopped(ctx, d.Id(), d.Timeout(schema.TimeoutUpdate)); err != nil {
				return diag.Errorf("error waiting for virtual machine %s to settle after update: %s", d.Id(), err)
			}
		}
	}

	// Phase 3: Manage VM running state (restart after stop, or standalone toggle)
	if runDiags := ensureVMRunningState(ctx, d, vmService, needsStop); runDiags != nil {
		return runDiags
	}

	return resourceVirtualMachineRead(ctx, d, meta)
}

// applySecurityGroupChanges diffs old vs new security groups (normalizing to FQIDs)
// and applies the necessary add/remove operations to the updater.
func applySecurityGroupChanges(updater *compute.VirtualMachineUpdateBuilder, d *schema.ResourceData, client *evroc.Client) *compute.VirtualMachineUpdateBuilder {
	old, new := d.GetChange("security_groups")
	oldSet := make(map[string]bool)
	for _, sg := range old.(*schema.Set).List() {
		sgVal := sg.(string)
		sgFQID := string(client.Networking().SecurityGroupRef(sgVal))
		if isFQID(sgVal) {
			sgFQID = sgVal
		}
		oldSet[sgFQID] = true
	}
	newSet := make(map[string]bool)
	for _, sg := range new.(*schema.Set).List() {
		sgVal := sg.(string)
		sgFQID := string(client.Networking().SecurityGroupRef(sgVal))
		if isFQID(sgVal) {
			sgFQID = sgVal
		}
		newSet[sgFQID] = true
	}
	for sg := range oldSet {
		if !newSet[sg] {
			updater = updater.RemoveSecurityGroup(sg)
		}
	}
	for sg := range newSet {
		if !oldSet[sg] {
			updater = updater.AddSecurityGroup(sg)
		}
	}
	return updater
}

// applyUserLabelChanges diffs old vs new user labels and applies
// the necessary add/remove operations to the updater.
func applyUserLabelChanges(updater *compute.VirtualMachineUpdateBuilder, d *schema.ResourceData) *compute.VirtualMachineUpdateBuilder {
	old, new := d.GetChange("user_labels")
	oldLabels := old.(map[string]interface{})
	newLabels := new.(map[string]interface{})
	for k := range oldLabels {
		if _, exists := newLabels[k]; !exists {
			updater = updater.RemoveLabel(k)
		}
	}
	for k, v := range newLabels {
		updater = updater.AddLabel(k, v.(string))
	}
	return updater
}

// ensureVMRunningState handles Phase 3 (restarting after stop-required changes)
// and standalone running state toggles.
func ensureVMRunningState(ctx context.Context, d *schema.ResourceData, vmService *compute.VirtualMachinesService, needsStop bool) diag.Diagnostics {
	if needsStop {
		if d.Get("running").(bool) {
			startUpdater := compute.UpdateVM(d.Id(), vmService).Start()
			if _, err := startUpdater.Apply(ctx); err != nil {
				return diag.Errorf("error starting virtual machine %s: %s", d.Id(), err)
			}
			if _, err := vmService.WaitForReady(ctx, d.Id(), d.Timeout(schema.TimeoutUpdate)); err != nil {
				return diag.Errorf("error waiting for virtual machine %s to be ready: %s", d.Id(), err)
			}
		}
		return nil
	}

	if !d.HasChange("running") {
		return nil
	}

	if d.Get("running").(bool) {
		startUpdater := compute.UpdateVM(d.Id(), vmService).Start()
		if _, err := startUpdater.Apply(ctx); err != nil {
			return diag.Errorf("error starting virtual machine %s: %s", d.Id(), err)
		}
		if _, err := vmService.WaitForReady(ctx, d.Id(), d.Timeout(schema.TimeoutUpdate)); err != nil {
			return diag.Errorf("error waiting for virtual machine %s to be ready: %s", d.Id(), err)
		}
	} else {
		stopUpdater := compute.UpdateVM(d.Id(), vmService).Stop()
		if _, err := stopUpdater.Apply(ctx); err != nil {
			return diag.Errorf("error stopping virtual machine %s: %s", d.Id(), err)
		}
		if _, err := vmService.WaitForStopped(ctx, d.Id(), d.Timeout(schema.TimeoutUpdate)); err != nil {
			return diag.Errorf("error waiting for virtual machine %s to stop: %s", d.Id(), err)
		}
	}

	return nil
}

func resourceVirtualMachineDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	err := client.Compute().VirtualMachines().Delete(ctx, d.Id())
	if err != nil {
		if !isNotFoundError(err) {
			return diag.Errorf("error deleting virtual machine %s: %s", d.Id(), err)
		}
		d.SetId("")
		return nil
	}

	timeout := d.Timeout(schema.TimeoutDelete)
	if err := client.Compute().VirtualMachines().WaitForDeleted(ctx, d.Id(), timeout); err != nil {
		return diag.Errorf("error waiting for virtual machine %s to be deleted: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}
