// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestAnnotateForceNew(t *testing.T) {
	res := map[string]*schema.Resource{
		"test": {
			Schema: map[string]*schema.Schema{
				"replace":  {Type: schema.TypeString, ForceNew: true, Description: "Replaced."},
				"in_place": {Type: schema.TypeString, Description: "Updated."},
				"empty":    {Type: schema.TypeString, ForceNew: true},
				"block": {
					Type: schema.TypeList,
					Elem: &schema.Resource{Schema: map[string]*schema.Schema{
						"inner": {Type: schema.TypeString, ForceNew: true, Description: "Inner."},
					}},
				},
			},
		},
	}

	annotateForceNew(res)
	annotateForceNew(res) // idempotent

	s := res["test"].Schema
	want := map[string]string{
		"replace":  "Replaced. " + forceNewNote,
		"in_place": "Updated.",
		"empty":    forceNewNote,
	}
	for name, desc := range want {
		if got := s[name].Description; got != desc {
			t.Errorf("%s: got %q, want %q", name, got, desc)
		}
	}
	inner := s["block"].Elem.(*schema.Resource).Schema["inner"].Description
	if inner != "Inner. "+forceNewNote {
		t.Errorf("inner: got %q", inner)
	}
}

func TestProviderForceNewAttributesDocumented(t *testing.T) {
	for name, r := range New("test")().ResourcesMap {
		for attr, s := range r.Schema {
			if s.ForceNew && !strings.HasSuffix(s.Description, forceNewNote) {
				t.Errorf("%s.%s is ForceNew but its description does not say so", name, attr)
			}
		}
	}
}
