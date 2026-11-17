// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func requireLBSuccess(t *testing.T, diags diag.Diagnostics) {
	t.Helper()
	if diags.HasError() {
		t.Fatal(diags)
	}
}

func requireLBNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func decodeLBRequest(t *testing.T, w http.ResponseWriter, r *http.Request, target interface{}) bool {
	t.Helper()
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		t.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return false
	}
	return true
}

func checkLBValidation(t *testing.T, res *schema.Resource, cfg map[string]interface{}, wantError bool) string {
	t.Helper()
	config := terraform.NewResourceConfigRaw(cfg)
	diags := res.Validate(config)
	_, err := res.Diff(context.Background(), nil, config, nil)
	if got := diags.HasError() || err != nil; got != wantError {
		t.Fatalf("validation error = %v, want %v: diagnostics=%v diff error=%v", got, wantError, diags, err)
	}
	return fmt.Sprint(diags, err)
}

func checkLBNoDiff(t *testing.T, res *schema.Resource, d *schema.ResourceData, cfg map[string]interface{}, meta interface{}, prefixes ...string) {
	t.Helper()
	diff, err := res.Diff(context.Background(), d.State(), terraform.NewResourceConfigRaw(cfg), meta)
	requireLBNoError(t, err)
	if diff == nil {
		return
	}
	for key, attr := range diff.Attributes {
		for _, prefix := range prefixes {
			if strings.HasPrefix(key, prefix) {
				t.Errorf("unexpected drift: %s = %#v", key, attr)
			}
		}
	}
}

func checkLBJSON(t *testing.T, got interface{}, want string) {
	t.Helper()
	body, err := json.Marshal(got)
	if err != nil {
		t.Error(err)
		return
	}
	var actual, expected interface{}
	if err := json.Unmarshal(body, &actual); err != nil {
		t.Error(err)
		return
	}
	if err := json.Unmarshal([]byte(want), &expected); err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("JSON = %s, want %s", body, want)
	}
}

func newLBTest(t *testing.T) (*mockServer, *ProviderConfig) {
	t.Helper()
	server := newMockServer()
	t.Cleanup(server.close)
	return server, newTestProviderConfig(t, server.server.URL)
}
