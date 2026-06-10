---
page_title: "evroc_lb_backend_service Data Source - evroc"
subcategory: ""
description: |-
  Get information about an existing evroc load balancer backend service.
---

# evroc_lb_backend_service (Data Source)

Get information about an existing evroc load balancer backend service.

## Example Usage

```terraform
data "evroc_lb_backend_service" "web" {
  name = "web-svc"
}
```

## Schema

### Required

- `name` (String) Name of the backend service to look up.

### Optional

- `project` (String) Project this resource belongs to. Defaults to the provider project.
- `region` (String) Region where the backend service is located.

### Read-Only

- `backend_pool_ref` (String) Fully qualified reference to the backend pool.
- `created_at` (String) Timestamp when the backend service was created (RFC3339 format).
- `fqid` (String) Fully qualified resource ID (FQID).
- `id` (String) The ID of this resource.
- `port` (Number) Backend port to forward traffic to.
- `proxy_protocol` (Boolean) Whether PROXY protocol is enabled.
- `service_id` (String) Unique identifier (UUID) of the backend service.
- `system_labels` (Map of String) System-managed labels (read-only).
- `user_labels` (Map of String) User-defined labels (key/value pairs).
