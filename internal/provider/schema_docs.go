// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// forceNewNote is appended to the description of every ForceNew attribute so
// the generated documentation states which changes replace the resource.
const forceNewNote = "Changing this forces a new resource to be created."

// annotateForceNew appends forceNewNote to the description of every ForceNew
// attribute in the given resources, including attributes of nested blocks.
func annotateForceNew(resources map[string]*schema.Resource) {
	for _, r := range resources {
		annotateForceNewSchema(r.Schema)
	}
}

func annotateForceNewSchema(s map[string]*schema.Schema) {
	for _, attr := range s {
		if attr.ForceNew && !strings.HasSuffix(attr.Description, forceNewNote) {
			attr.Description = strings.TrimSpace(attr.Description + " " + forceNewNote)
		}
		if nested, ok := attr.Elem.(*schema.Resource); ok {
			annotateForceNewSchema(nested.Schema)
		}
	}
}
