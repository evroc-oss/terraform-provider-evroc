// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import (
	"context"
	"time"

	lbtypes "github.com/evroc-oss/evroc-go-sdk/types/loadbalancer"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
			"health_check": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Active health check configuration for this backend service.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"interval": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Duration between health check attempts (e.g., \"5s\", \"10s\").",
						},
						"timeout": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Duration allowed for one health check attempt (e.g., \"3s\").",
						},
						"target_port": {
							Type:             schema.TypeInt,
							Optional:         true,
							ValidateDiagFunc: validatePort(),
							Description:      "Backend endpoint port to check. Defaults to the backend service port.",
						},
						"healthy_threshold": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Number of consecutive successes before marking an endpoint healthy.",
						},
						"unhealthy_threshold": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Number of consecutive failures before marking an endpoint unhealthy.",
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
										Description: "List of HTTP response status codes considered healthy.",
										Elem: &schema.Schema{
											Type: schema.TypeInt,
										},
									},
								},
							},
						},
						"https": {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "HTTPS health check configuration.",
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
										Description: "List of HTTP response status codes considered healthy.",
										Elem: &schema.Schema{
											Type: schema.TypeInt,
										},
									},
									"insecure_skip_verify": {
										Type:        schema.TypeBool,
										Optional:    true,
										Default:     false,
										Description: "Disable TLS certificate verification for health checks.",
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
			"backend_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of resolved backend addresses for this service.",
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

func expandHealthCheckHTTPS(hcMap map[string]interface{}, hc *lbtypes.BackendserviceSpecHealthCheck) {
	httpsList, ok := hcMap["https"].([]interface{})
	if !ok || len(httpsList) == 0 || httpsList[0] == nil {
		return
	}
	httpsMap := httpsList[0].(map[string]interface{})
	httpsHC := &struct {
		ExpectedStatuses *[]int32                                          `json:"expectedStatuses,omitempty"`
		Host             *string                                           `json:"host,omitempty"`
		Method           *lbtypes.BackendserviceSpecHealthCheckHttpsMethod `json:"method,omitempty"`
		Path             string                                            `json:"path"`
		Tls              *lbtypes.BackendserviceSpecHealthCheckTls         `json:"tls,omitempty"` //nolint:staticcheck,revive // matches SDK anonymous struct field name
	}{
		Path: httpsMap["path"].(string),
	}
	if v, ok := httpsMap["method"].(string); ok && v != "" {
		m := lbtypes.BackendserviceSpecHealthCheckHttpsMethod(v)
		httpsHC.Method = &m
	}
	if v, ok := httpsMap["host"].(string); ok && v != "" {
		httpsHC.Host = &v
	}
	if statuses, ok := httpsMap["expected_statuses"].([]interface{}); ok && len(statuses) > 0 {
		httpsHC.ExpectedStatuses = expandInt32Statuses(statuses)
	}
	if v, ok := httpsMap["insecure_skip_verify"].(bool); ok && v {
		httpsHC.Tls = &lbtypes.BackendserviceSpecHealthCheckTls{
			InsecureSkipVerify: &v,
		}
	}
	hc.Https = httpsHC
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

	if v, ok := hcMap["interval"].(string); ok && v != "" {
		hc.Interval = &v
	}
	if v, ok := hcMap["timeout"].(string); ok && v != "" {
		hc.Timeout = &v
	}
	if v, ok := hcMap["target_port"].(int); ok && v != 0 {
		p := int32(v)
		hc.TargetPort = &p
	}
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
	expandHealthCheckHTTPS(hcMap, hc)

	return hc
}

func flattenHealthCheck(hc *lbtypes.BackendserviceSpecHealthCheck) []interface{} {
	if hc == nil {
		return nil
	}
	m := map[string]interface{}{}

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

	if hc.Https != nil {
		https := map[string]interface{}{
			"path": hc.Https.Path,
		}
		if hc.Https.Method != nil {
			https["method"] = string(*hc.Https.Method)
		}
		if hc.Https.Host != nil {
			https["host"] = *hc.Https.Host
		}
		if hc.Https.ExpectedStatuses != nil {
			statuses := make([]interface{}, len(*hc.Https.ExpectedStatuses))
			for i, s := range *hc.Https.ExpectedStatuses {
				statuses[i] = int(s)
			}
			https["expected_statuses"] = statuses
		}
		skipVerify := false
		if hc.Https.Tls != nil && hc.Https.Tls.InsecureSkipVerify != nil {
			skipVerify = *hc.Https.Tls.InsecureSkipVerify
		}
		https["insecure_skip_verify"] = skipVerify
		m["https"] = []interface{}{https}
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

	if svc.Metadata.UserLabels != nil && len(*svc.Metadata.UserLabels) > 0 {
		diags = setDiag(d, "user_labels", flattenLabels(svc.Metadata.UserLabels), diags)
	}

	diags = setDiag(d, "system_labels", flattenLabels(svc.Metadata.SystemLabels), diags)
	diags = setDiag(d, "fqid", client.LoadBalancer().BackendServiceRef(svc.Metadata.Id), diags)

	diags = setDiag(d, "backend_count", svc.Status.Backends, diags)

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

	svc, err := lbClient.BackendServices().Get(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error reading backend service %s: %s", d.Id(), err)
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
		return diag.Errorf("error updating backend service %s: %s", d.Id(), err)
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
