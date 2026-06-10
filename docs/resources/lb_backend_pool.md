---
page_title: "evroc_lb_backend_pool Resource - evroc"
subcategory: ""
description: |-
  Provides an evroc load balancer backend pool resource.
---

# evroc_lb_backend_pool (Resource)

Provides an evroc load balancer backend pool resource. A backend pool groups VM references that receive traffic from backend services.

## Example Usage

```terraform
resource "evroc_lb_backend_pool" "web" {
  name         = "web-pool"
  backend_refs = [evroc_virtual_machine.web1.fqid, evroc_virtual_machine.web2.fqid]

  user_labels = {
    environment = "production"
  }
}
```

## Schema

### Required

- `name` (String) Name of the backend pool. Must be unique within the project.

### Optional

- `backend_refs` (Set of String) Set of fully qualified VM references to use as backends (e.g., `evroc_virtual_machine.my_vm.fqid`).
- `project` (String) Project this resource belongs to. Defaults to the provider project.
- `region` (String) Region where the backend pool is created. Defaults to provider region.
- `timeouts` (Block, Optional)
- `user_labels` (Map of String) User-defined labels (key/value pairs) for organizing and selecting resources.

### Read-Only

- `created_at` (String) Timestamp when the backend pool was created (RFC3339 format).
- `fqid` (String) Fully qualified resource ID (FQID). Use this to reference this resource from other resources.
- `id` (String) The ID of this resource.
- `pool_id` (String) Unique identifier (UUID) of the backend pool.
- `system_labels` (Map of String) System-managed labels automatically set by evroc (read-only).
