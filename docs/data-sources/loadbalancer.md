---
page_title: "evroc_loadbalancer Data Source - evroc"
subcategory: ""
description: |-
  Get information about an existing evroc load balancer.
---

# evroc_loadbalancer (Data Source)

Get information about an existing evroc load balancer.

## Example Usage

```terraform
data "evroc_loadbalancer" "web" {
  name = "web-lb"
}
```

## Schema

### Required

- `name` (String) Name of the load balancer to look up.

### Optional

- `project` (String) Project this resource belongs to. Defaults to the provider project.
- `region` (String) Region where the load balancer is located.

### Read-Only

- `created_at` (String) Timestamp when the load balancer was created (RFC3339 format).
- `fqid` (String) Fully qualified resource ID (FQID).
- `id` (String) The ID of this resource.
- `lb_id` (String) Unique identifier (UUID) of the load balancer.
- `listener` (List) List of listeners for the load balancer.
- `public_ip_ref` (String) Fully qualified reference to the public IP for the load balancer.
- `system_labels` (Map of String) System-managed labels (read-only).
- `user_labels` (Map of String) User-defined labels (key/value pairs).
