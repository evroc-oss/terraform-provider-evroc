# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.7.2] - 2026-07-16

### Changed
- Adapt `evroc_roles` data source for regenerated SDK IAM types (`RoleInfo.Id`, `*RoleInfo.Description`, `RoleInfoScope`)

### Dependencies
- evroc Go SDK v0.7.2

## [0.7.1] - 2026-07-14

### Changed
- Documentation now recommends service account authentication for CI/CD instead of username/password
- Removed username/password from all README examples, env var tables, and config file samples
- Added `service_account_id` and `service_account_secret` to the provider env var reference table
- Added missing IAM resources and data sources to the README tables

### Deprecated
- `username` and `password` provider attributes — use `service_account_id` and `service_account_secret` instead

### Documentation
- Provider and resource docs updated

## [0.7.0] - 2026-07-14

### Added
- `evroc_service_account` resource and data source — create and manage service accounts for headless/CI workloads
- `evroc_service_account_credential` resource and data source — generate secrets for service account authentication
- `evroc_role_binding` resource — assign IAM roles to users or service accounts (accepts short names like `computeOperator` or full FQIDs)
- `evroc_roles` data source — browse the IAM role catalog
- `evroc_organization_quota` data source — organization-level limits and current usage
- `evroc_project_quota` data source — project-level limits and current usage
- `evroc_bucket_service_account_secret` data source — retrieve S3 credentials for bucket service accounts
- Service account provider authentication via `service_account_id` + `service_account_secret` fields (JWT bearer flow for CI/CD and GitOps)

### Changed
- `evroc_lb_backend_service`: removed HTTPS health check type; added `ip_protocol_selection` field (IPv4/IPv6)

### Fixed
- Subnet import no longer drifts on computed status fields

### Dependencies
- evroc Go SDK v0.7.0

## [0.6.0] - 2026-06-23

### Breaking Changes
- **`evroc_security_group`**: Now requires `vpc_ref` (the VPC the security group belongs to). For existing configs using the default VPC, add `vpc_ref = "default-se-sto"`.
- **`evroc_virtual_machine`**: Now requires `subnet_ref` (the subnet the VM is attached to). For existing configs using the default subnet, add `subnet_ref = "default-se-sto-<zone>"` (e.g. `default-se-sto-a`).
- All compute and networking APIs migrated from v1beta1 to v1beta2. Existing state files will need `terraform state replace-provider` if upgrading from 0.5.x.

### Added
- `evroc_loadbalancer` resource and data source with listeners and route references
- `evroc_lb_backend_pool` resource and data source with VM backend references
- `evroc_lb_backend_service` resource and data source with health checks (TCP/HTTP/HTTPS)
- `evroc_lb_l4_route` resource and data source for L4 traffic routing
- `evroc_vpc` resource and data source with RFC 1918 CIDR validation, status fields (assigned CIDRs, subnets)
- `evroc_subnet` resource and data source with cross-field validation and IPv4/IPv6 usage stats
- `evroc_snapshot` resource and data source for disk snapshots
- `subnet_ref` field on `evroc_virtual_machine`
- `vpc_ref` field on `evroc_security_group`
- `snapshot` field on `evroc_disk` (mutually exclusive with `image`)
- `health_check` block on `evroc_lb_backend_service` (TCP, HTTP, HTTPS)
- Validators: `validateVPCCIDR` (RFC 1918), `validateSubnetCIDR` (/16-/29), `validateVPCStackType`
- `WaitForDeleted` on VPC and subnet delete

### Dependencies
- evroc Go SDK v0.6.0

## [0.5.3] - 2026-06-18

### Fixed
- VM updates retry on 409 Conflict caused by concurrent resource modifications (e.g. SG detach racing with VM update)
- Public IP move between VMs works in a single apply (wait for release, verify attachment)
- Public IP removal now works correctly (SDK v0.5.1 fix for omitempty serialization)

### Dependencies
- evroc Go SDK v0.5.1

## [0.5.2] - 2026-06-12

### Fixed
- Security group deletion no longer blocks when removing a SG from a VM and deleting it in the same apply (removed synchronous wait)
- Security group ordering on VMs no longer causes spurious diffs and "no updates to apply" errors (changed to unordered set)
- Public IP changes on VMs no longer force an unnecessary stop/start cycle

## [0.5.1] - 2026-06-10

### Fixed
- Object locking mode validator accepted `GOVERNANCE` and `COMPLIANCE` instead of the API-accepted values `Soft` and `Immutable`

## [0.5.0] - 2026-06-01

### Added
- `evroc_filestore` resource and data source — managed file system for shared storage

### Dependencies
- evroc Go SDK v0.5.0

## [0.4.3] - 2026-05-15

### Fixed
- Security group rule removal causing 422 error due to null/empty ghost entries in Terraform's set diff

## [0.4.2] - 2026-05-13

### Added
- `data_disks` attribute on `evroc_virtual_machine` resource and data source for managing additional data disks

### Changed
- README updated for official Terraform and OpenTofu registry listings

## [0.4.1] - 2026-05-06

### Added
- GPG signing of release artifacts (required for Terraform Registry publishing)

### Fixed
- Go version badge not rendering (changed `go 1.25.0` to `go 1.25` in go.mod)
- Cosign signature filename collision with GPG signature (renamed to `*.cosign.sig`)
- Outdated version references in README (replaced hardcoded `0.1.10` with current version)

## [0.4.0] - 2026-05-01

Initial public release of the evroc Terraform Provider.

### Added
- Complete resource support for evroc Cloud Platform:
  - `evroc_virtual_machine` - Virtual machine lifecycle management
  - `evroc_disk` - Persistent block storage volumes
  - `evroc_hotswap_disk_attachment` - Hotswap disk attachment to running VMs
  - `evroc_public_ip` - Public IPv4 address allocation
  - `evroc_security_group` - Network firewall rules (ingress/egress)
  - `evroc_placement_group` - VM placement strategies (spread)
  - `evroc_bucket` - S3-compatible object storage
  - `evroc_bucket_service_account` - Bucket access credentials
  - `evroc_project` - Project management
  - `evroc_permission_set` - User access control
  - `evroc_think_instance` - Dedicated AI inference instances
  - `evroc_think_api_key` - API key management for Think
- Data sources for all resources plus:
  - `evroc_disk_images` - Available OS images with named attributes
  - `evroc_compute_profiles` - VM sizes/flavors with named attributes
  - `evroc_think_models` - Available AI models
  - `evroc_think_sizes` - Available instance sizes
- `user_labels` support on all resources
- Cross-project resource management from a single provider instance
- Project creation with automatic wait for auth propagation
- Project deletion with wait for async cleanup
- Token-based authentication
- Automatic token refresh via refresh tokens
- Resource import support for all resources
- Acceptance tests for all resources with full CRUD coverage
- Example configurations covering common use cases
- Cosign-signed releases with SBOM and SLSA provenance attestations
- Multi-platform binaries (Linux, macOS, Windows, FreeBSD)

### Dependencies
- evroc Go SDK v0.4.0
- Terraform Plugin SDK v2.40.0
- Go 1.25.0

[Unreleased]: https://github.com/evroc-oss/terraform-provider-evroc/compare/v0.7.2...HEAD
[0.7.1]: https://github.com/evroc-oss/terraform-provider-evroc/releases/tag/v0.7.1
[0.5.1]: https://github.com/evroc-oss/terraform-provider-evroc/releases/tag/v0.5.1
[0.4.2]: https://github.com/evroc-oss/terraform-provider-evroc/releases/tag/v0.4.2
[0.4.1]: https://github.com/evroc-oss/terraform-provider-evroc/releases/tag/v0.4.1
[0.4.0]: https://github.com/evroc-oss/terraform-provider-evroc/releases/tag/v0.4.0
[0.4.3]: https://github.com/evroc-oss/terraform-provider-evroc/releases/tag/v0.4.3
[0.5.0]: https://github.com/evroc-oss/terraform-provider-evroc/releases/tag/v0.5.0
[0.7.0]: https://github.com/evroc-oss/terraform-provider-evroc/releases/tag/v0.7.0
[0.7.2]: https://github.com/evroc-oss/terraform-provider-evroc/releases/tag/v0.7.2
