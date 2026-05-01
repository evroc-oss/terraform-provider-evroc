// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"time"

	computetypes "github.com/evroc-oss/evroc-go-sdk/types/compute"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourcePlacementGroup() *schema.Resource {
	return &schema.Resource{
		Description: "Manages an evroc placement group for VM placement control.",

		CreateContext: resourcePlacementGroupCreate,
		ReadContext:   resourcePlacementGroupRead,
		UpdateContext: resourcePlacementGroupUpdate,
		DeleteContext: resourcePlacementGroupDelete,

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
				Description: "Name of the placement group.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Project this resource belongs to. Defaults to the provider project.",
			},
			"strategy": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validatePlacementStrategy(),
				Description:      "Placement strategy: 'spread' or 'cluster'.",
			},
			"zone": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				ValidateDiagFunc: validateZone(),
				Description:      "Zone for the placement group (a, b, or c).",
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Region where the placement group will be created.",
			},
			"user_labels": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "User-defined labels (key/value pairs) for organizing and selecting resources.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"pg_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier of the placement group.",
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
				Description: "Timestamp when the placement group was created.",
			},
			"fqid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified resource ID (FQID). Use this to reference this resource from other resources.",
			},
		},
	}
}

func resourcePlacementGroupCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)
	strategy := d.Get("strategy").(string)
	zone := d.Get("zone").(string)

	// Get user labels
	var userLabels map[string]string
	if labels, ok := d.GetOk("user_labels"); ok {
		userLabels = make(map[string]string)
		for k, v := range labels.(map[string]interface{}) {
			userLabels[k] = v.(string)
		}
	}

	req := BuildPlacementGroupCreateRequest(name, strategy, zone, userLabels)

	pg, err := client.Compute().PlacementGroups().Create(ctx, req)
	if err != nil {
		return diag.Errorf("error creating placement group: %s", err)
	}

	d.SetId(pg.Metadata.Id)
	d.Set("project", resolveProject(d, config))

	// Wait for placement group to be ready and capture the ready resource
	timeout := d.Timeout(schema.TimeoutCreate)
	readyPG, err := client.Compute().PlacementGroups().WaitForReady(ctx, name, timeout)
	if err != nil {
		return diag.Errorf("error waiting for placement group %s to be ready: %s", name, err)
	}

	// Use the ready resource's identity
	d.SetId(readyPG.Metadata.Id)

	return resourcePlacementGroupRead(ctx, d, meta)
}

func resourcePlacementGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	pg, err := client.Compute().PlacementGroups().Get(ctx, d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading placement group: %s", err)
	}

	diags = setDiag(d, "name", pg.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "region", derefString(pg.Metadata.Region), diags)
	diags = setDiag(d, "pg_id", pg.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", pg.Metadata.CreationTimestamp.Format(time.RFC3339), diags)

	diags = setDiag(d, "strategy", string(pg.Spec.Strategy.Type), diags)

	if pg.Spec.Placement.Zone != nil {
		diags = setDiag(d, "zone", *pg.Spec.Placement.Zone, diags)
	}

	if pg.Metadata.UserLabels != nil && len(*pg.Metadata.UserLabels) > 0 {
		diags = setDiag(d, "user_labels", flattenLabels(pg.Metadata.UserLabels), diags)
	}

	diags = setDiag(d, "system_labels", flattenLabels(pg.Metadata.SystemLabels), diags)
	diags = setDiag(d, "fqid", string(client.Compute().PlacementGroupRef(pg.Metadata.Id)), diags)

	return diags
}

func resourcePlacementGroupUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	if d.HasChange("user_labels") {
		pg, err := client.Compute().PlacementGroups().Get(ctx, d.Id())
		if err != nil {
			return diag.Errorf("error reading placement group %s: %s", d.Id(), err)
		}

		if labels, ok := d.GetOk("user_labels"); ok {
			userLabels := make(computetypes.UserLabels)
			for k, v := range labels.(map[string]interface{}) {
				userLabels[k] = v.(string)
			}
			pg.Metadata.UserLabels = &userLabels
		} else {
			pg.Metadata.UserLabels = nil
		}

		_, err = client.Compute().PlacementGroups().Patch(ctx, d.Id(), pg)
		if err != nil {
			return diag.Errorf("error updating placement group %s: %s", d.Id(), err)
		}
	}

	return resourcePlacementGroupRead(ctx, d, meta)
}

func resourcePlacementGroupDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	err := client.Compute().PlacementGroups().Delete(ctx, d.Id())
	if err != nil {
		// If already deleted, that's okay
		if !isNotFoundError(err) {
			return diag.Errorf("error deleting placement group: %s", err)
		}
		d.SetId("")
		return nil
	}

	// Wait for deletion to complete
	timeout := d.Timeout(schema.TimeoutDelete)
	if err := client.Compute().PlacementGroups().WaitForDeleted(ctx, d.Id(), timeout); err != nil {
		return diag.Errorf("error waiting for placement group %s to be deleted: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}
