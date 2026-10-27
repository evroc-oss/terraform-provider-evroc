// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"time"

	evroc "github.com/evroc-oss/evroc-go-sdk"
	"github.com/evroc-oss/evroc-go-sdk/kms"
	kmstypes "github.com/evroc-oss/evroc-go-sdk/types/kms"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	kmsKeyDeletionAcknowledged = "gone"
	kmsKeyPreActivated         = "PreActivated"
	kmsKeyActive               = "Active"
	kmsKeySuspended            = "Suspended"
	kmsKeyPendingImport        = "PendingImport"
	kmsKeyPendingDeletion      = "PendingDeletion"
	kmsKeyDeleted              = "Deleted"
)

func resourceKMSKey() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a KMS-generated encryption key in the provider region. Destroy schedules deletion and returns once acknowledged; the API retains the key during its deletion grace period. Immutable settings require replacement with a different name while the old key is pending deletion. Key material is never returned or stored in Terraform state. Terraform import adopts an existing key; it does not upload key material. Material import, manual rotation/trimming, and cancelling deletion use separate SDK operations.",
		CreateContext: resourceKMSKeyCreate, ReadContext: resourceKMSKeyRead,
		UpdateContext: resourceKMSKeyUpdate, DeleteContext: resourceKMSKeyDelete,
		Importer: &schema.ResourceImporter{StateContext: schema.ImportStatePassthroughContext},
		Timeouts: &schema.ResourceTimeout{Create: schema.DefaultTimeout(10 * time.Minute), Update: schema.DefaultTimeout(10 * time.Minute), Delete: schema.DefaultTimeout(10 * time.Minute)},
		Schema:   kmsKeySchema(),
	}
}

func kmsKeySchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name":                   {Type: schema.TypeString, Required: true, ForceNew: true, Description: "Key name, unique within the project and region.", ValidateDiagFunc: validation.ToDiagFunc(validation.StringMatch(regexp.MustCompile(`^[a-z][a-z0-9-]{1,61}[a-z0-9]$`), "must be 3-63 lowercase letters, digits or hyphens, starting with a letter and ending with a letter or digit"))},
		"project":                {Type: schema.TypeString, Optional: true, Computed: true, ForceNew: true, Description: "Project containing the key. Defaults to the provider project."},
		"region":                 {Type: schema.TypeString, Computed: true, Description: "Region selected by the provider configuration."},
		"algorithm":              {Type: schema.TypeString, Optional: true, Default: "aes256-gcm96", ForceNew: true, Description: "Encryption algorithm. Defaults to aes256-gcm96, the only supported algorithm.", ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"aes256-gcm96"}, false))},
		"auto_rotate_after_days": {Type: schema.TypeInt, Optional: true, ForceNew: true, Description: "Days between automatic rotations. Omit when creating the key to disable automatic rotation. Must be at least 1 when set. Immutable.", ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(1, 2147483647))},
		"expire_after_days":      {Type: schema.TypeInt, Optional: true, ForceNew: true, Description: "Days before key versions expire. Omit when creating the key for no automatic expiry. Must be at least 1 when set. Immutable.", ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(1, 2147483647))},
		"enabled":                {Type: schema.TypeBool, Optional: true, Default: true, Description: "Activate the key when true (the default); suspend it when false."},
		"imported":               {Type: schema.TypeBool, Computed: true, Description: "Whether key material was imported externally. This resource creates KMS-generated keys; existing imported keys may be adopted after material import."},
		"state":                  {Type: schema.TypeString, Computed: true, Description: "Observed KMS lifecycle state."},
		"fqid":                   {Type: schema.TypeString, Computed: true, Description: "Fully qualified key identifier."},
		"key_id":                 {Type: schema.TypeString, Computed: true, Description: "System-assigned UUID."},
		"created_at":             {Type: schema.TypeString, Computed: true, Description: "Creation timestamp in RFC3339 format."},
		"versions": {Type: schema.TypeList, Computed: true, Description: "Observed key versions; does not contain key material.", Elem: &schema.Resource{Schema: map[string]*schema.Schema{
			"version": {Type: schema.TypeInt, Computed: true}, "state": {Type: schema.TypeString, Computed: true},
			"created_at": {Type: schema.TypeString, Computed: true}, "expires_after": {Type: schema.TypeString, Computed: true},
		}}},
	}
}

func resourceKMSKeyCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, diags := resolveClient(d, meta.(*ProviderConfig))
	if diags.HasError() {
		return diags
	}
	req := &kmstypes.KeyRequest{ApiVersion: "kms/v1", Kind: "Key", Metadata: kmstypes.RegionalMetadataRequest{Id: d.Get("name").(string)}, Spec: kmstypes.KeySpec{Algorithm: kmstypes.KeySpecAlgorithm(d.Get("algorithm").(string))}}
	if n, ok := d.GetOk("auto_rotate_after_days"); ok {
		v := int32(n.(int))
		req.Spec.AutoRotateAfterDays = &v
	}
	if n, ok := d.GetOk("expire_after_days"); ok {
		v := int32(n.(int))
		req.Spec.ExpireAfterDays = &v
	}
	key, err := client.KMS().Keys().Create(ctx, req)
	if err != nil {
		return diag.Errorf("error creating KMS key: %s", err)
	}
	d.SetId(key.Metadata.Id)
	// Keep the ID on a subsequent action failure so Terraform can refresh or destroy it.
	if err := reconcileKMSKey(ctx, client.KMS().Keys(), d.Id(), d.Get("enabled").(bool), d.Timeout(schema.TimeoutCreate)); err != nil {
		return diag.FromErr(err)
	}
	return resourceKMSKeyRead(ctx, d, meta)
}

func resourceKMSKeyUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, diags := resolveClient(d, meta.(*ProviderConfig))
	if diags.HasError() {
		return diags
	}
	if err := reconcileKMSKey(ctx, client.KMS().Keys(), d.Id(), d.Get("enabled").(bool), d.Timeout(schema.TimeoutUpdate)); err != nil {
		return diag.FromErr(err)
	}
	return resourceKMSKeyRead(ctx, d, meta)
}

func reconcileKMSKey(ctx context.Context, keys *kms.KeysService, id string, enabled bool, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	key, err := waitKMSKey(ctx, keys, id, timeout, kmsKeyPreActivated, kmsKeyActive, kmsKeySuspended)
	if err != nil {
		return err
	}
	target := kmsKeySuspended
	if enabled {
		target = kmsKeyActive
	}
	if key.Status.State == target {
		return nil
	}
	// Newly generated keys enter PreActivated. Activate before suspending them.
	if enabled || key.Status.State == kmsKeyPreActivated {
		if err := keys.Activate(ctx, id); err != nil {
			return fmt.Errorf("activating KMS key: %w", err)
		}
		if _, err := waitKMSKey(ctx, keys, id, timeout, kmsKeyActive); err != nil {
			return err
		}
	}
	if !enabled {
		if err := keys.Suspend(ctx, id); err != nil {
			return fmt.Errorf("suspending KMS key: %w", err)
		}
		_, err = waitKMSKey(ctx, keys, id, timeout, kmsKeySuspended)
	}
	return err
}

func waitKMSKey(ctx context.Context, keys *kms.KeysService, id string, timeout time.Duration, targets ...string) (*kmstypes.Key, error) {
	// Actions use second-resolution reconciliation triggers. Waiting before the
	// next observation also separates successive actions by more than one second.
	wait := &retry.StateChangeConf{Target: targets, Timeout: timeout, Delay: 2 * time.Second, MinTimeout: 2 * time.Second,
		Refresh: func() (interface{}, string, error) {
			key, err := keys.Get(ctx, id)
			if err != nil {
				if isNotFoundError(err) {
					return nil, "", nil
				}
				return nil, "", err
			}
			for _, target := range targets {
				if key.Status.State == target {
					return key, target, nil
				}
			}
			switch key.Status.State {
			case kmsKeyPendingImport, kmsKeyPendingDeletion, kmsKeyDeleted:
				return nil, "", fmt.Errorf("KMS key %s is %s; resolve this lifecycle state before managing activation", id, key.Status.State)
			}
			return key, key.Status.State, nil
		},
	}
	result, err := wait.WaitForStateContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("waiting for KMS key %s: %w", id, err)
	}
	return result.(*kmstypes.Key), nil
}

func resourceKMSKeyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, diags := resolveClient(d, meta.(*ProviderConfig))
	if diags.HasError() {
		return diags
	}
	key, err := client.KMS().Keys().Get(ctx, d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	if key.Status.State == kmsKeyDeleted {
		d.SetId("")
		return nil
	}
	return flattenKMSKey(d, client, key)
}

func flattenKMSKey(d *schema.ResourceData, client *evroc.Client, key *kmstypes.Key) diag.Diagnostics {
	var diags diag.Diagnostics
	values := map[string]interface{}{
		"name":                   key.Metadata.Id,
		"project":                client.DefaultProject(),
		"region":                 client.DefaultRegion(),
		"algorithm":              string(key.Spec.Algorithm),
		"enabled":                key.Status.State == kmsKeyActive,
		"state":                  key.Status.State,
		"imported":               key.Spec.Imported != nil && *key.Spec.Imported,
		"auto_rotate_after_days": 0,
		"expire_after_days":      0,
		"key_id":                 key.Metadata.Uid.String(),
		"created_at":             key.Metadata.CreationTimestamp.Format(time.RFC3339),
		"fqid":                   fmt.Sprintf("/kms/projects/%s/regions/%s/keys/%s", client.DefaultProject(), client.DefaultRegion(), key.Metadata.Id),
	}
	if key.Spec.AutoRotateAfterDays != nil {
		values["auto_rotate_after_days"] = int(*key.Spec.AutoRotateAfterDays)
	}
	if key.Spec.ExpireAfterDays != nil {
		values["expire_after_days"] = int(*key.Spec.ExpireAfterDays)
	}
	versions := make([]interface{}, 0)
	if key.Status.Versions != nil {
		for _, v := range *key.Status.Versions {
			expiry := ""
			if v.ExpiresAfter != nil {
				expiry = v.ExpiresAfter.Format(time.RFC3339)
			}
			versions = append(versions, map[string]interface{}{"version": v.Version, "state": v.State, "created_at": v.CreatedAt.Format(time.RFC3339), "expires_after": expiry})
		}
	}
	values["versions"] = versions
	for field, value := range values {
		diags = setDiag(d, field, value, diags)
	}
	return diags
}

func resourceKMSKeyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, diags := resolveClient(d, meta.(*ProviderConfig))
	if diags.HasError() {
		return diags
	}
	keys := client.KMS().Keys()
	if err := keys.Delete(ctx, d.Id()); err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	// Deletion is scheduled, not immediate; do not wait through the grace period.
	wait := &retry.StateChangeConf{Target: []string{kmsKeyDeletionAcknowledged}, Timeout: d.Timeout(schema.TimeoutDelete), Delay: 2 * time.Second, MinTimeout: 2 * time.Second, Refresh: func() (interface{}, string, error) {
		key, err := keys.Get(ctx, d.Id())
		if isNotFoundError(err) {
			return true, kmsKeyDeletionAcknowledged, nil
		}
		if err != nil {
			return nil, "", err
		}
		if key.Status.State == kmsKeyPendingDeletion || key.Status.State == kmsKeyDeleted {
			return true, kmsKeyDeletionAcknowledged, nil
		}
		return key, key.Status.State, nil
	}}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}
