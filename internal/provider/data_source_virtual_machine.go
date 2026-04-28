// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"path"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVirtualMachine() *schema.Resource {
	return &schema.Resource{
		Description: "Get information about an existing evroc virtual machine.",

		ReadContext: dataSourceVirtualMachineRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the virtual machine to look up.",
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
				Description: "Region where the VM is located.",
			},
			// Computed fields
			"vm_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier (UUID) of the virtual machine.",
			},
			"flavor": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "VM flavor/size.",
			},
			"boot_disk": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the boot disk.",
			},
			"ssh_keys": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of SSH public keys.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"cloud_config_user_data": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Cloud-init user data script.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the VM was created (RFC3339 format).",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Current status of the virtual machine (e.g., Running, Stopped, Creating).",
			},
			"fqid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified resource ID (FQID).",
			},
		},
	}
}

func dataSourceVirtualMachineRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	var diags diag.Diagnostics

	name := d.Get("name").(string)

	vm, err := client.Compute().VirtualMachines().Get(ctx, name)
	if err != nil {
		return diag.Errorf("error reading virtual machine %s: %s", name, err)
	}

	d.SetId(vm.Metadata.Id)
	diags = setDiag(d, "name", vm.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "region", derefString(vm.Metadata.Region), diags)
	diags = setDiag(d, "vm_id", vm.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", vm.Metadata.CreationTimestamp.Format(time.RFC3339), diags)

	// Set flavor (from ComputeProfileRef) - normalize path
	// API returns "/compute/global/computeProfiles/flavor", we want just "flavor"
	flavor := path.Base(vm.Spec.ComputeProfileRef)
	diags = setDiag(d, "flavor", flavor, diags)

	// Set boot disk (find the disk with bootFrom = true)
	if vm.Spec.Disks != nil {
		for _, diskRef := range *vm.Spec.Disks {
			if diskRef.BootFrom != nil && *diskRef.BootFrom {
				diags = setDiag(d, "boot_disk", diskRef.DiskRef, diags)
				break
			}
		}
	}

	// Set SSH keys from OsSettings
	if vm.Spec.OsSettings != nil && vm.Spec.OsSettings.Ssh != nil && vm.Spec.OsSettings.Ssh.AuthorizedKeys != nil {
		keys := make([]string, 0, len(*vm.Spec.OsSettings.Ssh.AuthorizedKeys))
		for _, key := range *vm.Spec.OsSettings.Ssh.AuthorizedKeys {
			keys = append(keys, key.Value)
		}
		if len(keys) > 0 {
			diags = setDiag(d, "ssh_keys", keys, diags)
		}
	}

	// Set user data from OsSettings
	if vm.Spec.OsSettings != nil && vm.Spec.OsSettings.CloudInitUserData != nil {
		diags = setDiag(d, "cloud_config_user_data", *vm.Spec.OsSettings.CloudInitUserData, diags)
	}

	// Set VM status
	if vm.Status.VirtualMachineStatus != nil {
		diags = setDiag(d, "status", *vm.Status.VirtualMachineStatus, diags)
	}

	diags = setDiag(d, "fqid", string(client.Compute().VMRef(vm.Metadata.Id)), diags)

	return diags
}
