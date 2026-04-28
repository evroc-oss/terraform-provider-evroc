// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"errors"

	evroc "github.com/evroc-oss/evroc-go-sdk"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// setDiag is a helper to check errors from d.Set() calls
// It appends any error to the diagnostics and returns the updated diags
func setDiag(d *schema.ResourceData, key string, value interface{}, diags diag.Diagnostics) diag.Diagnostics {
	if err := d.Set(key, value); err != nil {
		return append(diags, diag.Errorf("error setting %s: %s", key, err)...)
	}
	return diags
}

// isNotFoundError checks if an error indicates a resource was not found (HTTP 404).
func isNotFoundError(err error) bool {
	return errors.Is(err, evroc.ErrNotFound)
}

// flattenLabels converts a *map[string]string (SDK label types) to
// map[string]interface{} for Terraform state. Returns an empty map if labels is nil.
func flattenLabels[T ~map[string]string](labels *T) map[string]interface{} {
	result := make(map[string]interface{})
	if labels != nil {
		for k, v := range *labels {
			result[k] = v
		}
	}
	return result
}
