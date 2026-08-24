// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"time"

	storagetypes "github.com/evroc-oss/evroc-go-sdk/types/storage"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceBucket() *schema.Resource {
	return &schema.Resource{
		Description: "Manages an evroc S3-compatible storage bucket with object retention and locking support.",

		CreateContext: resourceBucketCreate,
		ReadContext:   resourceBucketRead,
		UpdateContext: resourceBucketUpdate,
		DeleteContext: resourceBucketDelete,

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
				Description: "Name of the storage bucket.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Project this resource belongs to. Defaults to the provider project.",
			},
			"object_retention_mode": {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          "Disabled",
				ValidateDiagFunc: validateObjectRetentionMode(),
				Description:      "Object retention mode: Disabled, Versioned, or Locking.",
			},
			"object_locking": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Default object locking configuration.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"mode": {
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: validateObjectLockingMode(),
							Description:      "Lock mode: Soft or Immutable.",
						},
						"duration_days": {
							Type:             schema.TypeInt,
							Required:         true,
							ValidateDiagFunc: validatePositiveInt(),
							Description:      "Duration in days for the default lock.",
						},
					},
				},
			},
			"lifecycle_rule": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Lifecycle rules that determine how and when objects or object versions are automatically deleted.",
				Elem:        bucketLifecycleRuleSchema(),
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Region where the bucket will be created.",
			},
			"user_labels": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "User-defined labels (key/value pairs) for organizing and selecting resources.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"bucket_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier of the bucket.",
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
				Description: "Timestamp when the bucket was created.",
			},
		},
	}
}

func resourceBucketCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)
	retentionMode := d.Get("object_retention_mode").(string)

	var lockingMode string
	var lockingDuration int32

	if v, ok := d.GetOk("object_locking"); ok {
		locking := v.([]interface{})
		if len(locking) > 0 {
			lockingConfig := locking[0].(map[string]interface{})
			lockingMode = lockingConfig["mode"].(string)
			lockingDuration = int32(lockingConfig["duration_days"].(int))
		}
	}

	// Get user labels
	var userLabels map[string]string
	if labels, ok := d.GetOk("user_labels"); ok {
		userLabels = make(map[string]string)
		for k, v := range labels.(map[string]interface{}) {
			userLabels[k] = v.(string)
		}
	}

	req := BuildBucketCreateRequest(name, retentionMode, lockingMode, lockingDuration, userLabels)

	lifecyclePolicy, err := expandBucketLifecyclePolicy(d.Get("lifecycle_rule").([]interface{}))
	if err != nil {
		return diag.FromErr(err)
	}
	req.Spec.LifecyclePolicy = lifecyclePolicy

	bucket, err := client.Storage().Buckets().Create(ctx, req)
	if err != nil {
		return diag.Errorf("error creating bucket: %s", err)
	}

	d.SetId(bucket.Metadata.Id)
	d.Set("project", resolveProject(d, config))

	// Wait for bucket to be ready and capture the ready resource
	timeout := d.Timeout(schema.TimeoutCreate)
	readyBucket, err := client.Storage().Buckets().WaitForReady(ctx, name, timeout)
	if err != nil {
		return diag.Errorf("error waiting for bucket %s to be ready: %s", name, err)
	}

	// Use the ready resource's identity
	d.SetId(readyBucket.Metadata.Id)

	return resourceBucketRead(ctx, d, meta)
}

func resourceBucketRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	bucket, err := client.Storage().Buckets().Get(ctx, d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading bucket: %s", err)
	}

	diags = setDiag(d, "name", bucket.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "region", derefString(bucket.Metadata.Region), diags)
	diags = setDiag(d, "bucket_id", bucket.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", bucket.Metadata.CreationTimestamp.Format(time.RFC3339), diags)

	// Always set object_retention_mode (default to "Disabled" if nil)
	retentionMode := "Disabled"
	if bucket.Spec.ObjectRetentionMode != nil {
		retentionMode = string(*bucket.Spec.ObjectRetentionMode)
	}
	diags = setDiag(d, "object_retention_mode", retentionMode, diags)

	if bucket.Spec.DefaultObjectLocking != nil {
		locking := make(map[string]interface{})
		locking["mode"] = string(bucket.Spec.DefaultObjectLocking.Mode)
		locking["duration_days"] = int(bucket.Spec.DefaultObjectLocking.DurationDays)
		diags = setDiag(d, "object_locking", []interface{}{locking}, diags)
	}

	diags = setDiag(d, "lifecycle_rule", flattenBucketLifecyclePolicy(bucket.Spec.LifecyclePolicy), diags)

	if bucket.Metadata.UserLabels != nil && len(*bucket.Metadata.UserLabels) > 0 {
		diags = setDiag(d, "user_labels", flattenLabels(bucket.Metadata.UserLabels), diags)
	}

	diags = setDiag(d, "system_labels", flattenLabels(bucket.Metadata.SystemLabels), diags)

	return diags
}

func resourceBucketUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	if d.HasChanges("object_retention_mode", "object_locking", "lifecycle_rule", "user_labels") {
		// Get the current bucket
		bucket, err := client.Storage().Buckets().Get(ctx, d.Id())
		if err != nil {
			return diag.Errorf("error getting bucket for update: %s", err)
		}

		// Update the fields
		retentionMode := d.Get("object_retention_mode").(string)
		if retentionMode != "" {
			mode := storagetypes.BucketSpecObjectRetentionMode(retentionMode)
			bucket.Spec.ObjectRetentionMode = &mode
		}

		if d.HasChange("object_locking") {
			if v, ok := d.GetOk("object_locking"); ok {
				locking := v.([]interface{})
				if len(locking) > 0 {
					lockingConfig := locking[0].(map[string]interface{})
					lockingMode := storagetypes.BucketSpecDefaultObjectLockingMode(lockingConfig["mode"].(string))
					lockingDuration := int32(lockingConfig["duration_days"].(int))
					bucket.Spec.DefaultObjectLocking = &storagetypes.BucketSpecDefaultObjectLocking{
						Mode:         lockingMode,
						DurationDays: lockingDuration,
					}
				}
			} else {
				bucket.Spec.DefaultObjectLocking = nil
			}
		}

		if d.HasChange("lifecycle_rule") {
			lifecyclePolicy, err := expandBucketLifecyclePolicy(d.Get("lifecycle_rule").([]interface{}))
			if err != nil {
				return diag.FromErr(err)
			}
			bucket.Spec.LifecyclePolicy = lifecyclePolicy
		}

		// Update user labels
		if d.HasChange("user_labels") {
			if labels, ok := d.GetOk("user_labels"); ok {
				userLabels := make(storagetypes.UserLabels)
				for k, v := range labels.(map[string]interface{}) {
					userLabels[k] = v.(string)
				}
				bucket.Metadata.UserLabels = &userLabels
			} else {
				bucket.Metadata.UserLabels = nil
			}
		}

		_, err = client.Storage().Buckets().Patch(ctx, d.Id(), bucket)
		if err != nil {
			return diag.Errorf("error updating bucket: %s", err)
		}
	}

	return resourceBucketRead(ctx, d, meta)
}

func resourceBucketDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	err := client.Storage().Buckets().Delete(ctx, d.Id())
	if err != nil {
		if !isNotFoundError(err) {
			return diag.Errorf("error deleting bucket: %s", err)
		}
		d.SetId("")
		return nil
	}

	timeout := d.Timeout(schema.TimeoutDelete)
	if err := client.Storage().Buckets().WaitForDeleted(ctx, d.Id(), timeout); err != nil {
		return diag.Errorf("error waiting for bucket %s to be deleted: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}

// bucketLifecycleRule and bucketLifecycleTag alias the anonymous struct types
// used by the generated SDK for lifecycle policy rules and filter tags, so they
// can be constructed outside the SDK package.
type bucketLifecycleRule = struct {
	AbortIncompleteMultipart *storagetypes.BucketSpecLifecyclePolicyAbortIncompleteMultipart `json:"abortIncompleteMultipart,omitempty"`
	Disabled                 *bool                                                           `json:"disabled,omitempty"`
	ExpireCurrentVersion     *storagetypes.BucketSpecLifecyclePolicyExpireCurrentVersion     `json:"expireCurrentVersion,omitempty"`
	ExpireNonCurrentVersion  *storagetypes.BucketSpecLifecyclePolicyExpireNonCurrentVersion  `json:"expireNonCurrentVersion,omitempty"`
	Filter                   *storagetypes.BucketSpecLifecyclePolicyFilter                   `json:"filter,omitempty"`
	Id                       string                                                          `json:"id"` //nolint:staticcheck // must be "Id" for type identity with the generated SDK struct
}

type bucketLifecycleTag = struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func bucketLifecycleRuleSchema() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Unique identifier for the lifecycle rule.",
			},
			"disabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether the rule is excluded from lifecycle evaluation.",
			},
			"expire_current_version": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Deletes the object in a non-versioned bucket, or adds a deletion marker in a versioned bucket.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"days": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Number of days after which the current version of an object expires.",
						},
						"date": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Date (RFC3339) on which the current version of an object expires.",
						},
						"expire_orphaned_deletion_markers": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Whether orphaned deletion markers are expired.",
						},
					},
				},
			},
			"expire_non_current_version": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Removes old versions of an object.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"days": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Number of days after which a non-current version of an object expires.",
						},
						"max_num_versions": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Maximum number of non-current versions to retain.",
						},
					},
				},
			},
			"abort_incomplete_multipart": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Aborts in-progress multipart uploads.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"days": {
							Type:             schema.TypeInt,
							Required:         true,
							ValidateDiagFunc: validatePositiveInt(),
							Description:      "Number of days after which an incomplete multipart upload is aborted.",
						},
					},
				},
			},
			"filter": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Description: "Filters that determine which objects the rule applies to. If omitted, the rule applies to all objects. " +
					"Multiple filter properties are ANDed together.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"prefix": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Key prefix that objects must match for the rule to apply.",
						},
						"size_greater_than": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Minimum object size (in bytes) for the rule to apply.",
						},
						"size_less_than": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Maximum object size (in bytes) for the rule to apply.",
						},
						"tag": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "Key-value tags that objects must have for the rule to apply.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"key": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Tag key that objects must have for the rule to apply.",
									},
									"value": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Tag value that objects must have for the rule to apply.",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// firstBlock returns the map of the first entry of a MaxItems:1 nested block, or nil if unset.
func firstBlock(m map[string]interface{}, key string) map[string]interface{} {
	blocks, ok := m[key].([]interface{})
	if !ok || len(blocks) == 0 || blocks[0] == nil {
		return nil
	}
	return blocks[0].(map[string]interface{})
}

func expandBucketLifecyclePolicy(rules []interface{}) (*storagetypes.BucketSpecLifecyclePolicy, error) {
	if len(rules) == 0 {
		return nil, nil
	}

	policy := &storagetypes.BucketSpecLifecyclePolicy{}
	for _, r := range rules {
		m, ok := r.(map[string]interface{})
		if !ok {
			continue
		}

		rule := bucketLifecycleRule{Id: m["id"].(string)}

		if v, ok := m["disabled"].(bool); ok && v {
			disabled := v
			rule.Disabled = &disabled
		}

		ecv, err := expandLifecycleExpireCurrentVersion(firstBlock(m, "expire_current_version"), rule.Id)
		if err != nil {
			return nil, err
		}
		rule.ExpireCurrentVersion = ecv
		rule.ExpireNonCurrentVersion = expandLifecycleExpireNonCurrentVersion(firstBlock(m, "expire_non_current_version"))
		rule.Filter = expandLifecycleFilter(firstBlock(m, "filter"))

		if b := firstBlock(m, "abort_incomplete_multipart"); b != nil {
			rule.AbortIncompleteMultipart = &storagetypes.BucketSpecLifecyclePolicyAbortIncompleteMultipart{
				Days: int32(b["days"].(int)),
			}
		}

		policy.Rules = append(policy.Rules, rule)
	}

	return policy, nil
}

func expandLifecycleExpireCurrentVersion(b map[string]interface{}, ruleID string) (*storagetypes.BucketSpecLifecyclePolicyExpireCurrentVersion, error) {
	if b == nil {
		return nil, nil
	}
	ecv := &storagetypes.BucketSpecLifecyclePolicyExpireCurrentVersion{}
	if days, ok := b["days"].(int); ok && days > 0 {
		d := int32(days)
		ecv.Days = &d
	}
	if dateStr, ok := b["date"].(string); ok && dateStr != "" {
		date, err := time.Parse(time.RFC3339, dateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid date %q in lifecycle rule %q: %w", dateStr, ruleID, err)
		}
		ecv.Date = &date
	}
	if v, ok := b["expire_orphaned_deletion_markers"].(bool); ok && v {
		markers := v
		ecv.ExpireOrphanedDeletionMarkers = &markers
	}
	return ecv, nil
}

func expandLifecycleExpireNonCurrentVersion(b map[string]interface{}) *storagetypes.BucketSpecLifecyclePolicyExpireNonCurrentVersion {
	if b == nil {
		return nil
	}
	encv := &storagetypes.BucketSpecLifecyclePolicyExpireNonCurrentVersion{}
	if days, ok := b["days"].(int); ok && days > 0 {
		d := int32(days)
		encv.Days = &d
	}
	if maxVersions, ok := b["max_num_versions"].(int); ok && maxVersions > 0 {
		mv := int32(maxVersions)
		encv.MaxNumVersions = &mv
	}
	return encv
}

func expandLifecycleFilter(b map[string]interface{}) *storagetypes.BucketSpecLifecyclePolicyFilter {
	if b == nil {
		return nil
	}
	filter := &storagetypes.BucketSpecLifecyclePolicyFilter{}
	if prefix, ok := b["prefix"].(string); ok && prefix != "" {
		filter.Prefix = &prefix
	}
	if size, ok := b["size_greater_than"].(int); ok && size > 0 {
		s := int64(size)
		filter.SizeGreaterThan = &s
	}
	if size, ok := b["size_less_than"].(int); ok && size > 0 {
		s := int64(size)
		filter.SizeLessThan = &s
	}
	if tags, ok := b["tag"].([]interface{}); ok && len(tags) > 0 {
		tagList := make([]bucketLifecycleTag, 0, len(tags))
		for _, t := range tags {
			tm := t.(map[string]interface{})
			tagList = append(tagList, bucketLifecycleTag{
				Key:   tm["key"].(string),
				Value: tm["value"].(string),
			})
		}
		filter.Tag = &tagList
	}
	return filter
}

func flattenBucketLifecyclePolicy(policy *storagetypes.BucketSpecLifecyclePolicy) []interface{} {
	if policy == nil || len(policy.Rules) == 0 {
		return nil
	}

	rules := make([]interface{}, 0, len(policy.Rules))
	for _, rule := range policy.Rules {
		m := map[string]interface{}{
			"id":       rule.Id,
			"disabled": rule.Disabled != nil && *rule.Disabled,
		}

		if ecv := rule.ExpireCurrentVersion; ecv != nil {
			b := map[string]interface{}{}
			if ecv.Days != nil {
				b["days"] = int(*ecv.Days)
			}
			if ecv.Date != nil {
				b["date"] = ecv.Date.Format(time.RFC3339)
			}
			b["expire_orphaned_deletion_markers"] = ecv.ExpireOrphanedDeletionMarkers != nil && *ecv.ExpireOrphanedDeletionMarkers
			m["expire_current_version"] = []interface{}{b}
		}

		if encv := rule.ExpireNonCurrentVersion; encv != nil {
			b := map[string]interface{}{}
			if encv.Days != nil {
				b["days"] = int(*encv.Days)
			}
			if encv.MaxNumVersions != nil {
				b["max_num_versions"] = int(*encv.MaxNumVersions)
			}
			m["expire_non_current_version"] = []interface{}{b}
		}

		if rule.AbortIncompleteMultipart != nil {
			m["abort_incomplete_multipart"] = []interface{}{map[string]interface{}{
				"days": int(rule.AbortIncompleteMultipart.Days),
			}}
		}

		if filter := rule.Filter; filter != nil {
			b := map[string]interface{}{}
			if filter.Prefix != nil {
				b["prefix"] = *filter.Prefix
			}
			if filter.SizeGreaterThan != nil {
				b["size_greater_than"] = int(*filter.SizeGreaterThan)
			}
			if filter.SizeLessThan != nil {
				b["size_less_than"] = int(*filter.SizeLessThan)
			}
			if filter.Tag != nil && len(*filter.Tag) > 0 {
				tags := make([]interface{}, 0, len(*filter.Tag))
				for _, t := range *filter.Tag {
					tags = append(tags, map[string]interface{}{
						"key":   t.Key,
						"value": t.Value,
					})
				}
				b["tag"] = tags
			}
			m["filter"] = []interface{}{b}
		}

		rules = append(rules, m)
	}

	return rules
}
