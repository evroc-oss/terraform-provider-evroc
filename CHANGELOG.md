# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
- Token-based and username/password authentication
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

[Unreleased]: https://github.com/evroc-oss/terraform-provider-evroc/compare/v0.4.3...HEAD
[0.4.2]: https://github.com/evroc-oss/terraform-provider-evroc/releases/tag/v0.4.2
[0.4.1]: https://github.com/evroc-oss/terraform-provider-evroc/releases/tag/v0.4.1
[0.4.0]: https://github.com/evroc-oss/terraform-provider-evroc/releases/tag/v0.4.0
[0.4.3]: https://github.com/evroc-oss/terraform-provider-evroc/releases/tag/v0.4.3
