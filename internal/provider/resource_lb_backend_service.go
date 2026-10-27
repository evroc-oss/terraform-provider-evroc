// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	lbtypes "github.com/evroc-oss/evroc-go-sdk/types/loadbalancer"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceLBBackendService() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an evroc load balancer backend service resource. A backend service defines the port, protocol options, and backend pool reference for routing traffic.",

		CreateContext: resourceLBBackendServiceCreate,
		ReadContext:   resourceLBBackendServiceRead,
		UpdateContext: resourceLBBackendServiceUpdate,
		DeleteContext: resourceLBBackendServiceDelete,

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
				Description: "Name of the backend service. Must be unique within the project.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Project this resource belongs to. Defaults to the provider project.",
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Region where the backend service is created. Defaults to provider region.",
			},
			"port": {
				Type:             schema.TypeInt,
				Required:         true,
				ValidateDiagFunc: validatePort(),
				Description:      "Backend port to forward traffic to on the target instances.",
			},
			"backend_pool_ref": {
				Type:             schema.TypeString,
				Required:         true,
				DiffSuppressFunc: suppressFQIDDiff,
				Description:      "Fully qualified reference to the backend pool (e.g., evroc_lb_backend_pool.my_pool.fqid).",
			},
			"proxy_protocol": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Enable PROXY protocol to pass the real client IP to backends.",
			},
			"ip_protocol_selection": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				Description:      "IP protocol selection for backends. Valid values are \"IPv4\" and \"IPv6\". Defaults to the server-side value (currently \"IPv4\") when unset.",
				ValidateDiagFunc: validateIPProtocolSelection(),
			},
			"health_check": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Active health check configuration for this backend service.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"disable_proxy_protocol": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Omit PROXY protocol headers from health checks while preserving the backend traffic setting. Defaults to false (inherit the backend traffic setting).",
						},
						"interval": {
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "Duration between health check attempts (e.g., \"5s\", \"10s\"). Defaults to the server-side value when unset.",
						},
						"timeout": {
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "Duration allowed for one health check attempt (e.g., \"3s\"). Defaults to the server-side value when unset.",
						},
						"target_port": {
							Type:             schema.TypeInt,
							Optional:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(1, 65535)),
							DiffSuppressFunc: suppressDefaultTargetPortDiff,
							Description:      "Backend endpoint port to check. Defaults to the backend service port.",
						},
						"healthy_threshold": {
							Type:        schema.TypeInt,
							Optional:    true,
							Computed:    true,
							Description: "Number of consecutive successes before marking an endpoint healthy. Defaults to the server-side value when unset.",
						},
						"unhealthy_threshold": {
							Type:        schema.TypeInt,
							Optional:    true,
							Computed:    true,
							Description: "Number of consecutive failures before marking an endpoint unhealthy. Defaults to the server-side value when unset.",
						},
						"tcp": {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "TCP health check configuration.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"send": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "Text payload written after connecting.",
									},
									"receive": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "Text payload expected in the response.",
									},
								},
							},
						},
						"http": {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "HTTP health check configuration.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"path": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Absolute request path (e.g., \"/healthz\").",
									},
									"method": {
										Type:        schema.TypeString,
										Optional:    true,
										Default:     "GET",
										Description: "HTTP method used for health checks (GET or HEAD).",
									},
									"host": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "Host header value for the health check request.",
									},
									"expected_statuses": {
										Type:        schema.TypeList,
										Optional:    true,
										Computed:    true,
										Description: "List of HTTP response status codes considered healthy. Defaults to the server-side value when unset.",
										Elem: &schema.Schema{
											Type: schema.TypeInt,
										},
									},
								},
							},
						},
					},
				},
			},
			"user_labels": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "User-defined labels (key/value pairs) for organizing and selecting resources.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			// Computed fields
			"service_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier (UUID) of the backend service.",
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
				Description: "Timestamp when the backend service was created (RFC3339 format).",
			},
			"fqid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified resource ID (FQID). Use this to reference this resource from other resources.",
			},
			"backends": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Resolved backend addresses for this service.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name":    {Type: schema.TypeString, Computed: true, Description: "Backend name."},
						"zone":    {Type: schema.TypeString, Computed: true, Description: "Availability zone."},
						"address": {Type: schema.TypeString, Computed: true, Description: "IP address."},
					},
				},
			},
		},
	}
}

func expandInt32Statuses(raw []interface{}) *[]int32 {
	s := make([]int32, len(raw))
	for i, v := range raw {
		s[i] = int32(v.(int))
	}
	return &s
}

func expandHealthCheckTCP(hcMap map[string]interface{}, hc *lbtypes.BackendserviceSpecHealthCheck) {
	tcpList, ok := hcMap["tcp"].([]interface{})
	if !ok || len(tcpList) == 0 {
		return
	}
	tcpMap, _ := tcpList[0].(map[string]interface{})
	if tcpMap == nil {
		tcpMap = map[string]interface{}{}
	}
	tcp := &struct {
		Receive *string `json:"receive,omitempty"`
		Send    *string `json:"send,omitempty"`
	}{}
	if v, ok := tcpMap["send"].(string); ok && v != "" {
		tcp.Send = &v
	}
	if v, ok := tcpMap["receive"].(string); ok && v != "" {
		tcp.Receive = &v
	}
	hc.Tcp = tcp
}

func expandHealthCheckHTTP(hcMap map[string]interface{}, hc *lbtypes.BackendserviceSpecHealthCheck) {
	httpList, ok := hcMap["http"].([]interface{})
	if !ok || len(httpList) == 0 || httpList[0] == nil {
		return
	}
	httpMap := httpList[0].(map[string]interface{})
	httpHC := &struct {
		ExpectedStatuses *[]int32                                         `json:"expectedStatuses,omitempty"`
		Host             *string                                          `json:"host,omitempty"`
		Method           *lbtypes.BackendserviceSpecHealthCheckHttpMethod `json:"method,omitempty"`
		Path             string                                           `json:"path"`
	}{
		Path: httpMap["path"].(string),
	}
	if v, ok := httpMap["method"].(string); ok && v != "" {
		m := lbtypes.BackendserviceSpecHealthCheckHttpMethod(v)
		httpHC.Method = &m
	}
	if v, ok := httpMap["host"].(string); ok && v != "" {
		httpHC.Host = &v
	}
	if statuses, ok := httpMap["expected_statuses"].([]interface{}); ok && len(statuses) > 0 {
		httpHC.ExpectedStatuses = expandInt32Statuses(statuses)
	}
	hc.Http = httpHC
}

func expandHealthCheck(d *schema.ResourceData) *lbtypes.BackendserviceSpecHealthCheck {
	v, ok := d.GetOk("health_check")
	if !ok {
		return nil
	}
	hcList := v.([]interface{})
	if len(hcList) == 0 || hcList[0] == nil {
		return nil
	}
	hcMap := hcList[0].(map[string]interface{})
	hc := &lbtypes.BackendserviceSpecHealthCheck{}
	// Send false explicitly so updates can restore inheritance after true.
	disableProxyProtocol, _ := hcMap["disable_proxy_protocol"].(bool)
	hc.DisableProxyProtocol = &disableProxyProtocol

	if v, ok := hcMap["interval"].(string); ok && v != "" {
		hc.Interval = &v
	}
	if v, ok := hcMap["timeout"].(string); ok && v != "" {
		hc.Timeout = &v
	}
	// The API stores an omitted target port as 0, which never passes, so
	// default it to the backend service port.
	targetPort, _ := hcMap["target_port"].(int)
	if targetPort == 0 {
		targetPort = d.Get("port").(int)
	}
	p := int32(targetPort)
	hc.TargetPort = &p
	if v, ok := hcMap["healthy_threshold"].(int); ok && v != 0 {
		t := int32(v)
		hc.HealthyThreshold = &t
	}
	if v, ok := hcMap["unhealthy_threshold"].(int); ok && v != 0 {
		t := int32(v)
		hc.UnhealthyThreshold = &t
	}

	expandHealthCheckTCP(hcMap, hc)
	expandHealthCheckHTTP(hcMap, hc)

	return hc
}

// suppressDefaultTargetPortDiff hides the diff when target_port is unset and
// the stored value is the backend service port the provider defaulted it to.
func suppressDefaultTargetPortDiff(_, old, new string, d *schema.ResourceData) bool {
	return (new == "" || new == "0") && old == strconv.Itoa(d.Get("port").(int))
}

func flattenHealthCheck(hc *lbtypes.BackendserviceSpecHealthCheck) []interface{} {
	if hc == nil {
		return nil
	}
	m := map[string]interface{}{"disable_proxy_protocol": false}
	if hc.DisableProxyProtocol != nil {
		m["disable_proxy_protocol"] = *hc.DisableProxyProtocol
	}

	if hc.Interval != nil {
		m["interval"] = *hc.Interval
	}
	if hc.Timeout != nil {
		m["timeout"] = *hc.Timeout
	}
	if hc.TargetPort != nil {
		m["target_port"] = int(*hc.TargetPort)
	}
	if hc.HealthyThreshold != nil {
		m["healthy_threshold"] = int(*hc.HealthyThreshold)
	}
	if hc.UnhealthyThreshold != nil {
		m["unhealthy_threshold"] = int(*hc.UnhealthyThreshold)
	}

	if hc.Tcp != nil {
		tcp := map[string]interface{}{}
		if hc.Tcp.Send != nil {
			tcp["send"] = *hc.Tcp.Send
		}
		if hc.Tcp.Receive != nil {
			tcp["receive"] = *hc.Tcp.Receive
		}
		m["tcp"] = []interface{}{tcp}
	}

	if hc.Http != nil {
		http := map[string]interface{}{
			"path": hc.Http.Path,
		}
		if hc.Http.Method != nil {
			http["method"] = string(*hc.Http.Method)
		}
		if hc.Http.Host != nil {
			http["host"] = *hc.Http.Host
		}
		if hc.Http.ExpectedStatuses != nil {
			statuses := make([]interface{}, len(*hc.Http.ExpectedStatuses))
			for i, s := range *hc.Http.ExpectedStatuses {
				statuses[i] = int(s)
			}
			http["expected_statuses"] = statuses
		}
		m["http"] = []interface{}{http}
	}

	return []interface{}{m}
}

func resourceLBBackendServiceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)
	port := int32(d.Get("port").(int))
	backendPoolRef := d.Get("backend_pool_ref").(string)
	proxyProtocol := d.Get("proxy_protocol").(bool)

	var userLabels map[string]string
	if labels, ok := d.GetOk("user_labels"); ok {
		userLabels = make(map[string]string)
		for k, v := range labels.(map[string]interface{}) {
			userLabels[k] = v.(string)
		}
	}

	req := BuildBackendServiceCreateRequest(name, port, backendPoolRef, proxyProtocol, userLabels)

	if v, ok := d.GetOk("ip_protocol_selection"); ok {
		sel := lbtypes.BackendserviceSpecIpProtocolSelection(v.(string))
		req.Spec.IpProtocolSelection = &sel
	}

	if hc := expandHealthCheck(d); hc != nil {
		req.Spec.HealthCheck = hc
	}

	svc, err := client.LoadBalancer().BackendServices().Create(ctx, req)
	if err != nil {
		return diag.Errorf("error creating backend service %s: %s", name, err)
	}

	d.SetId(svc.Metadata.Id)
	d.Set("project", resolveProject(d, config))

	return resourceLBBackendServiceRead(ctx, d, meta)
}

func resourceLBBackendServiceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	client, clientDiags := resolveClient(d, config)
	if clientDiags.HasError() {
		return clientDiags
	}

	svc, err := client.LoadBalancer().BackendServices().Get(ctx, d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading backend service: %s", err)
	}

	diags = setDiag(d, "name", svc.Metadata.Id, diags)
	diags = setDiag(d, "project", resolveProject(d, config), diags)
	diags = setDiag(d, "region", derefString(svc.Metadata.Region), diags)
	diags = setDiag(d, "service_id", svc.Metadata.Uid.String(), diags)
	diags = setDiag(d, "created_at", svc.Metadata.CreationTimestamp.Format(time.RFC3339), diags)
	diags = setDiag(d, "port", int(svc.Spec.Port), diags)
	diags = setDiag(d, "backend_pool_ref", derefString(svc.Spec.BackendPoolRef), diags)

	proxyProtocol := false
	if svc.Spec.ProxyProtocol != nil {
		proxyProtocol = *svc.Spec.ProxyProtocol
	}
	diags = setDiag(d, "proxy_protocol", proxyProtocol, diags)

	if svc.Spec.IpProtocolSelection != nil {
		diags = setDiag(d, "ip_protocol_selection", string(*svc.Spec.IpProtocolSelection), diags)
	}

	if svc.Metadata.UserLabels != nil && len(*svc.Metadata.UserLabels) > 0 {
		diags = setDiag(d, "user_labels", flattenLabels(svc.Metadata.UserLabels), diags)
	}

	diags = setDiag(d, "system_labels", flattenLabels(svc.Metadata.SystemLabels), diags)
	diags = setDiag(d, "fqid", client.LoadBalancer().BackendServiceRef(svc.Metadata.Id), diags)

	if svc.Status.Backends != nil {
		backends := make([]map[string]interface{}, len(*svc.Status.Backends))
		for i, b := range *svc.Status.Backends {
			backends[i] = map[string]interface{}{
				"name":    b.Name,
				"zone":    b.Zone,
				"address": b.Address,
			}
		}
		diags = setDiag(d, "backends", backends, diags)
	} else {
		diags = setDiag(d, "backends", []map[string]interface{}{}, diags)
	}

	if svc.Spec.HealthCheck != nil {
		diags = setDiag(d, "health_check", flattenHealthCheck(svc.Spec.HealthCheck), diags)
	}

	return diags
}

func resourceLBBackendServiceUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	lbClient := client.LoadBalancer()

	// GET-then-PATCH: a concurrent write (e.g. the controller updating status)
	// invalidates the fetched resourceVersion and the PATCH fails with 409.
	err := retryOnConflict(ctx, d.Timeout(schema.TimeoutUpdate), func() error {
		svc, err := lbClient.BackendServices().Get(ctx, d.Id())
		if err != nil {
			return fmt.Errorf("error reading backend service %s: %w", d.Id(), err)
		}

		if d.HasChange("port") {
			svc.Spec.Port = int32(d.Get("port").(int))
		}

		if d.HasChange("backend_pool_ref") {
			ref := d.Get("backend_pool_ref").(string)
			svc.Spec.BackendPoolRef = &ref
		}

		if d.HasChange("proxy_protocol") {
			pp := d.Get("proxy_protocol").(bool)
			svc.Spec.ProxyProtocol = &pp
		}

		if d.HasChange("ip_protocol_selection") {
			if v, ok := d.GetOk("ip_protocol_selection"); ok {
				sel := lbtypes.BackendserviceSpecIpProtocolSelection(v.(string))
				svc.Spec.IpProtocolSelection = &sel
			} else {
				svc.Spec.IpProtocolSelection = nil
			}
		}

		if d.HasChange("health_check") {
			svc.Spec.HealthCheck = expandHealthCheck(d)
		}

		if d.HasChange("user_labels") {
			if labels, ok := d.GetOk("user_labels"); ok {
				userLabels := make(lbtypes.UserLabels)
				for k, v := range labels.(map[string]interface{}) {
					userLabels[k] = v.(string)
				}
				svc.Metadata.UserLabels = &userLabels
			} else {
				svc.Metadata.UserLabels = nil
			}
		}

		if _, err := lbClient.BackendServices().Patch(ctx, d.Id(), svc); err != nil {
			return fmt.Errorf("error updating backend service %s: %w", d.Id(), err)
		}
		return nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceLBBackendServiceRead(ctx, d, meta)
}

func resourceLBBackendServiceDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	client, diags := resolveClient(d, config)
	if diags.HasError() {
		return diags
	}

	err := client.LoadBalancer().BackendServices().Delete(ctx, d.Id())
	if err != nil {
		if !isNotFoundError(err) {
			return diag.Errorf("error deleting backend service %s: %s", d.Id(), err)
		}
		d.SetId("")
		return nil
	}

	timeout := d.Timeout(schema.TimeoutDelete)
	if err := client.LoadBalancer().BackendServices().WaitForDeleted(ctx, d.Id(), timeout); err != nil {
		return diag.Errorf("error waiting for backend service %s to be deleted: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}
