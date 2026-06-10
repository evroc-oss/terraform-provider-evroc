---
page_title: "evroc_lb_backend_pool Data Source - evroc"
subcategory: ""
description: |-
  Get information about an existing evroc load balancer backend pool.
---

# evroc_lb_backend_pool (Data Source)

Get information about an existing evroc load balancer backend pool.

## Example Usage

```terraform
data "evroc_lb_backend_pool" "web" {
  name = "web-pool"
}
```

## Schema

### Required

- `name` (String) Name of the backend pool to look up.

### Optional

- `project` (String) Project this resource belongs to. Defaults to the provider project.
- `region` (String) Region where the backend pool is located.

### Read-Only

- `backend_refs` (List of String) VM references in the backend pool.
- `created_at` (String) Timestamp when the backend pool was created (RFC3339 format).
- `fqid` (String) Fully qualified resource ID (FQID).
- `id` (String) The ID of this resource.
- `pool_id` (String) Unique identifier (UUID) of the backend pool.
- `system_labels` (Map of String) System-managed labels (read-only).
- `user_labels` (Map of String) User-defined labels (key/value pairs).
