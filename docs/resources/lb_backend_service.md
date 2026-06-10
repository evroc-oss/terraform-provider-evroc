---
page_title: "evroc_lb_backend_service Resource - evroc"
subcategory: ""
description: |-
  Provides an evroc load balancer backend service resource.
---

# evroc_lb_backend_service (Resource)

Provides an evroc load balancer backend service resource. A backend service defines the port, protocol options, and backend pool reference for routing traffic.

## Example Usage

```terraform
resource "evroc_lb_backend_service" "web" {
  name             = "web-svc"
  port             = 8080
  backend_pool_ref = evroc_lb_backend_pool.web.fqid
  proxy_protocol   = true

  user_labels = {
    environment = "production"
  }
}
```

## Schema

### Required

- `backend_pool_ref` (String) Fully qualified reference to the backend pool (e.g., `evroc_lb_backend_pool.my_pool.fqid`).
- `name` (String) Name of the backend service. Must be unique within the project.
- `port` (Number) Backend port to forward traffic to on the target instances.

### Optional

- `project` (String) Project this resource belongs to. Defaults to the provider project.
- `proxy_protocol` (Boolean) Enable PROXY protocol to pass the real client IP to backends. Defaults to `false`.
- `region` (String) Region where the backend service is created. Defaults to provider region.
- `timeouts` (Block, Optional)
- `user_labels` (Map of String) User-defined labels (key/value pairs) for organizing and selecting resources.

### Read-Only

- `created_at` (String) Timestamp when the backend service was created (RFC3339 format).
- `fqid` (String) Fully qualified resource ID (FQID). Use this to reference this resource from other resources.
- `id` (String) The ID of this resource.
- `service_id` (String) Unique identifier (UUID) of the backend service.
- `system_labels` (Map of String) System-managed labels automatically set by evroc (read-only).
