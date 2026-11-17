// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	evroc "github.com/evroc-oss/evroc-go-sdk"
	"github.com/evroc-oss/evroc-go-sdk/loadbalancer"
	lbtypes "github.com/evroc-oss/evroc-go-sdk/types/loadbalancer"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceLBHTTPRoute() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a load balancer HTTP route with hostname and request matching for HTTP and HTTPS listeners.",
		CreateContext: resourceLBHTTPRouteCreate,
		ReadContext:   resourceLBHTTPRouteRead,
		UpdateContext: resourceLBHTTPRouteUpdate,
		DeleteContext: resourceLBHTTPRouteDelete,
		CustomizeDiff: validateHTTPRoute,
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
				Description: "HTTP route name, unique within the project and region.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Project containing the route. Defaults to the provider project.",
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Region containing the route. Defaults to the provider region.",
			},
			"hostnames": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Hostnames to match, including optional leading *. wildcards. Omit to accept all hostnames accepted by the listener.",
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validation.ToDiagFunc(validation.StringMatch(regexp.MustCompile(`^(\*\.)?[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`), "must be a lowercase DNS hostname with an optional leading *. wildcard")),
				},
			},
			"rule": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "Ordered routing rules. A rule without matches is a catch-all.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:             schema.TypeString,
							Required:         true,
							Description:      "Unique rule name, stable when rules are reordered.",
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringMatch(regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`), "must be a lowercase DNS subdomain name")),
						},
						"backend_service_ref": {
							Type:             schema.TypeString,
							Required:         true,
							Description:      "Backend service reference in the same project and region.",
							DiffSuppressFunc: suppressFQIDDiff,
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
						},
						"match": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "Alternative request matches (OR). All fields within one match must match (AND).",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"method": {
										Type:             schema.TypeString,
										Optional:         true,
										Description:      "HTTP request method to match. Omit to match any method.",
										ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodConnect, http.MethodOptions, http.MethodTrace, http.MethodPatch}, false)),
									},
									"path": {
										Type:        schema.TypeList,
										Optional:    true,
										MaxItems:    1,
										Description: "Request path match. Omission matches the PathPrefix /.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"type": {
													Type:             schema.TypeString,
													Optional:         true,
													Default:          string(lbtypes.PathPrefix),
													Description:      "Path matching type: Exact or PathPrefix. Prefixes match complete path segments.",
													ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{string(lbtypes.Exact), string(lbtypes.PathPrefix)}, false)),
												},
												"value": {
													Type:             schema.TypeString,
													Optional:         true,
													Default:          "/",
													Description:      "Absolute request path to match.",
													ValidateDiagFunc: validation.ToDiagFunc(validation.StringMatch(regexp.MustCompile(`^/.*`), "must start with /")),
												},
											},
										},
									},
									"header":      httpRouteValueSchema("Request headers that must all match. Names are case-insensitive and must be unique."),
									"query_param": httpRouteValueSchema("Request query parameters that must all match. Names must be unique."),
								},
							},
						},
					},
				},
			},
			"user_labels": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "User-defined resource labels.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"system_labels": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "System-managed resource labels.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"route_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "System-assigned route UUID.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Creation timestamp in RFC3339 format.",
			},
			"fqid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified HTTP route reference.",
			},
		},
	}
}

func httpRouteValueSchema(description string) *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeSet,
		Optional:    true,
		Description: description,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"name": {
					Type:             schema.TypeString,
					Required:         true,
					Description:      "Header or query parameter name.",
					ValidateDiagFunc: validation.ToDiagFunc(validation.StringMatch(regexp.MustCompile("^[A-Za-z0-9!#$%&'*+\\-.^_`|~]+$"), "must be an HTTP token")),
				},
				"value": {
					Type:        schema.TypeString,
					Required:    true,
					Description: "Exact value to match.",
				},
			},
		},
	}
}

func validateHTTPRoute(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
	if !newValueWhollyKnown(d, "rule") {
		return nil
	}
	names := make(map[string]bool)
	for _, item := range d.Get("rule").([]interface{}) {
		rule := item.(map[string]interface{})
		name := rule["name"].(string)
		if names[name] {
			return fmt.Errorf("HTTP route rule names must be unique: %q", name)
		}
		names[name] = true
		matches := rule["match"].([]interface{})
		for _, item := range matches {
			match := item.(map[string]interface{})
			for _, field := range []string{"header", "query_param"} {
				seen := make(map[string]bool)
				for _, entry := range match[field].(*schema.Set).List() {
					name := entry.(map[string]interface{})["name"].(string)
					if field == "header" {
						name = strings.ToLower(name)
					}
					if seen[name] {
						return fmt.Errorf("HTTP route %s names must be unique: %q", field, name)
					}
					seen[name] = true
				}
			}
		}
	}
	return nil
}

// The SDK represents matches as an anonymous struct; keep its shape local here.
type httpRouteMatch = struct {
	Headers     *[]lbtypes.HttprouteSpecRulesItemHeadersItem     `json:"headers,omitempty"`
	Method      *lbtypes.HttprouteSpecRulesItemMatchesMethod     `json:"method,omitempty"`
	Path        *lbtypes.HttprouteSpecRulesItemPath              `json:"path,omitempty"`
	QueryParams *[]lbtypes.HttprouteSpecRulesItemQueryParamsItem `json:"queryParams,omitempty"`
}

func expandHTTPRoute(d *schema.ResourceData) lbtypes.HttprouteSpec {
	spec := lbtypes.HttprouteSpec{Rules: make([]lbtypes.HttprouteSpecRulesItem, 0)}
	hosts := make([]string, 0)
	for _, host := range d.Get("hostnames").(*schema.Set).List() {
		hosts = append(hosts, host.(string))
	}
	if len(hosts) > 0 {
		spec.Hostnames = &hosts
	}
	for _, item := range d.Get("rule").([]interface{}) {
		block := item.(map[string]interface{})
		rule := lbtypes.HttprouteSpecRulesItem{Name: block["name"].(string), BackendServiceRef: block["backend_service_ref"].(string)}
		matches := make([]httpRouteMatch, 0)
		for _, item := range block["match"].([]interface{}) {
			block := item.(map[string]interface{})
			match := httpRouteMatch{}
			if method := block["method"].(string); method != "" {
				value := lbtypes.HttprouteSpecRulesItemMatchesMethod(method)
				match.Method = &value
			}
			if paths := block["path"].([]interface{}); len(paths) > 0 {
				path := paths[0].(map[string]interface{})
				typ := lbtypes.HttprouteSpecRulesItemPathType(path["type"].(string))
				value := path["value"].(string)
				match.Path = &lbtypes.HttprouteSpecRulesItemPath{Type: &typ, Value: &value}
			}
			headers := make([]lbtypes.HttprouteSpecRulesItemHeadersItem, 0)
			for _, item := range block["header"].(*schema.Set).List() {
				entry := item.(map[string]interface{})
				headers = append(headers, lbtypes.HttprouteSpecRulesItemHeadersItem{Name: entry["name"].(string), Value: entry["value"].(string)})
			}
			if len(headers) > 0 {
				match.Headers = &headers
			}
			params := make([]lbtypes.HttprouteSpecRulesItemQueryParamsItem, 0)
			for _, item := range block["query_param"].(*schema.Set).List() {
				entry := item.(map[string]interface{})
				params = append(params, lbtypes.HttprouteSpecRulesItemQueryParamsItem{Name: entry["name"].(string), Value: entry["value"].(string)})
			}
			if len(params) > 0 {
				match.QueryParams = &params
			}
			matches = append(matches, match)
		}
		if len(matches) > 0 {
			rule.Matches = &matches
		}
		spec.Rules = append(spec.Rules, rule)
	}
	return spec
}

func flattenHTTPRouteRules(rules []lbtypes.HttprouteSpecRulesItem) []interface{} {
	result := make([]interface{}, 0, len(rules))
	for _, rule := range rules {
		matches := make([]interface{}, 0)
		if rule.Matches != nil {
			for _, match := range *rule.Matches {
				block := map[string]interface{}{"method": "", "path": []interface{}{}, "header": []interface{}{}, "query_param": []interface{}{}}
				if match.Method != nil {
					block["method"] = string(*match.Method)
				}
				if match.Path != nil {
					typ, value := string(lbtypes.PathPrefix), "/"
					if match.Path.Type != nil {
						typ = string(*match.Path.Type)
					}
					if match.Path.Value != nil {
						value = *match.Path.Value
					}
					block["path"] = []interface{}{map[string]interface{}{"type": typ, "value": value}}
				}
				headers := make([]interface{}, 0)
				if match.Headers != nil {
					for _, entry := range *match.Headers {
						headers = append(headers, map[string]interface{}{"name": entry.Name, "value": entry.Value})
					}
				}
				block["header"] = headers
				params := make([]interface{}, 0)
				if match.QueryParams != nil {
					for _, entry := range *match.QueryParams {
						params = append(params, map[string]interface{}{"name": entry.Name, "value": entry.Value})
					}
				}
				block["query_param"] = params
				matches = append(matches, block)
			}
		}
		result = append(result, map[string]interface{}{"name": rule.Name, "backend_service_ref": rule.BackendServiceRef, "match": matches})
	}
	return result
}

func resourceLBHTTPRouteCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}
	request := loadbalancer.NewHTTPRouteBuilder(d.Get("name").(string)).Build()
	request.Spec = expandHTTPRoute(d)
	labels := lbtypes.UserLabels(expandUserLabels(d.Get("user_labels").(map[string]interface{})))
	request.Metadata.UserLabels = &labels
	route, err := client.LoadBalancer().HTTPRoutes().Create(ctx, request)
	if err != nil {
		return diag.Errorf("error creating HTTP route: %s", err)
	}
	d.SetId(route.Metadata.Id)
	d.Set("project", resolveProject(d, config))
	return resourceLBHTTPRouteRead(ctx, d, meta)
}

func resourceLBHTTPRouteRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, diags := resolveClient(d, meta.(*ProviderConfig))
	if diags.HasError() {
		return diags
	}
	route, err := client.LoadBalancer().HTTPRoutes().Get(ctx, d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading HTTP route: %s", err)
	}
	return flattenHTTPRoute(d, client, route)
}

func flattenHTTPRoute(d *schema.ResourceData, client *evroc.Client, route *lbtypes.Httproute) diag.Diagnostics {
	var diags diag.Diagnostics
	hosts := []string{}
	if route.Spec.Hostnames != nil {
		hosts = *route.Spec.Hostnames
	}
	values := map[string]interface{}{
		"name":          route.Metadata.Id,
		"project":       client.DefaultProject(),
		"region":        derefString(route.Metadata.Region),
		"route_id":      route.Metadata.Uid.String(),
		"created_at":    route.Metadata.CreationTimestamp.Format(time.RFC3339),
		"fqid":          client.LoadBalancer().HTTPRouteRef(route.Metadata.Id),
		"user_labels":   flattenLabels(route.Metadata.UserLabels),
		"system_labels": flattenLabels(route.Metadata.SystemLabels),
		"hostnames":     hosts,
		"rule":          flattenHTTPRouteRules(route.Spec.Rules),
	}
	for field, value := range values {
		diags = setDiag(d, field, value, diags)
	}
	return diags
}

func resourceLBHTTPRouteUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, diags := resolveClient(d, meta.(*ProviderConfig))
	if diags.HasError() {
		return diags
	}
	routes := client.LoadBalancer().HTTPRoutes()
	// Preserve optimistic concurrency while explicitly clearing omitted mutable fields.
	err := retryOnConflict(ctx, d.Timeout(schema.TimeoutUpdate), func() error {
		route, err := routes.Get(ctx, d.Id())
		if err != nil {
			return fmt.Errorf("reading HTTP route: %w", err)
		}
		spec := expandHTTPRoute(d)
		// JSON Merge Patch preserves omitted map keys. Delete old labels explicitly
		// so the configured map owns the complete set, including partial removals.
		labels := make(map[string]interface{})
		if route.Metadata.UserLabels != nil {
			for key := range *route.Metadata.UserLabels {
				labels[key] = nil
			}
		}
		for key, value := range d.Get("user_labels").(map[string]interface{}) {
			labels[key] = value
		}
		patch := map[string]interface{}{"metadata": map[string]interface{}{"resourceVersion": route.Metadata.ResourceVersion, "userLabels": labels}, "spec": map[string]interface{}{"hostnames": spec.Hostnames, "rules": spec.Rules}}
		if _, err := routes.Patch(ctx, d.Id(), patch); err != nil {
			return fmt.Errorf("updating HTTP route: %w", err)
		}
		return nil
	})
	if err != nil {
		return diag.FromErr(err)
	}
	return resourceLBHTTPRouteRead(ctx, d, meta)
}

func resourceLBHTTPRouteDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, diags := resolveClient(d, meta.(*ProviderConfig))
	if diags.HasError() {
		return diags
	}
	routes := client.LoadBalancer().HTTPRoutes()
	if err := routes.Delete(ctx, d.Id()); err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error deleting HTTP route: %s", err)
	}
	if err := routes.WaitForDeleted(ctx, d.Id(), d.Timeout(schema.TimeoutDelete)); err != nil {
		return diag.Errorf("error waiting for HTTP route deletion: %s", err)
	}
	d.SetId("")
	return nil
}
