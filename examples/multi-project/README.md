# Multi-Project Security Groups

This example shows how to manage security groups across multiple evroc projects from a single Terraform configuration.

## Problem

Security groups are project-scoped — a group created in project A cannot be directly applied to a VM in project B. In a multi-project setup (e.g. separate dev and prod projects) this means duplicating firewall rules by hand, which causes drift over time.

## Solution

Use **provider aliases** (one per project) and **shared `locals`** for rule definitions:

- A dedicated `shared` project holds the canonical security group objects. It acts as the source of record for security policy and can be audited independently.
- Workload projects (`dev`, `prod`) create their own security group instances using the same `locals`, so every project enforces identical rules.
- The security group `name` is taken from the shared group's `.name` attribute, keeping names consistent across projects for dashboards and audit logs.
- A `data` source shows how to look up an existing security group in any project without managing its lifecycle.

Changing a rule in `locals` (for example, restricting SSH to a bastion CIDR) propagates to every project on the next `terraform apply` — no manual updates required.

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│  Terraform state                                             │
│                                                              │
│  provider "evroc" alias="shared"  ──►  shared project       │
│    evroc_security_group.shared_web   (canonical definition)  │
│                                                              │
│  provider "evroc" alias="dev"     ──►  dev project          │
│    evroc_security_group.dev_web      (mirrored from locals)  │
│    evroc_virtual_machine.dev_web     (uses dev_web SG)       │
│                                                              │
│  provider "evroc" alias="prod"    ──►  prod project         │
│    evroc_security_group.prod_web     (mirrored from locals)  │
│    evroc_virtual_machine.prod_web    (uses prod_web SG)      │
└──────────────────────────────────────────────────────────────┘
```

## Prerequisites

- Three existing evroc projects (shared, dev, prod). See the `project` example to create them with Terraform.
- API credentials with access to all three projects.

## Usage

**1. Create a `terraform.tfvars` file:**

```hcl
organization_id   = "your-org-id"
shared_project_id = "your-shared-project-id"
dev_project_id    = "your-dev-project-id"
prod_project_id   = "your-prod-project-id"
ssh_public_key    = "ssh-ed25519 AAAA... user@host"
```

**2. Export credentials:**

```bash
export EVROC_TOKEN="your-token"
export EVROC_REFRESH_TOKEN="your-refresh-token"
```

**3. Deploy:**

```bash
terraform init
terraform plan
terraform apply
```

## Updating security policy

To change firewall rules across all projects at once, edit the relevant `local` value in `main.tf`:

```hcl
# Before: SSH open to the world
rules_ssh_ingress = {
  ...
  remote_ip = "0.0.0.0/0"
}

# After: SSH restricted to a bastion host
rules_ssh_ingress = {
  ...
  remote_ip = "203.0.113.10/32"
}
```

Run `terraform apply` — all three security groups (shared, dev, prod) update atomically.

## Adding a new project

1. Add a new provider alias block targeting the new project ID.
2. Add a new `evroc_security_group` resource using `provider = evroc.<new_alias>` and the same `locals`.
3. Deploy workload resources under the new alias.

## Related examples

- [project/](../project/) — Creating and managing evroc projects
- [networking/](../networking/) — Security groups and public IP management
- [complete/](../complete/) — Full production VM setup
