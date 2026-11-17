// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

// Get does not preserve nested unknown values. Custom validation must wait
// until the complete block is known, including when Terraform plans at apply.
func newValueWhollyKnown(d *schema.ResourceDiff, key string) bool {
	if !d.NewValueKnown(key) {
		return false
	}
	config := d.GetRawConfig()
	return config.IsNull() || config.GetAttr(key).IsWhollyKnown()
}

func expandUserLabels(labels map[string]interface{}) map[string]string {
	result := make(map[string]string, len(labels))
	for key, value := range labels {
		result[key] = value.(string)
	}
	return result
}
