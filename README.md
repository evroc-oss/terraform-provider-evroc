<div align="center">

<img src="docs/images/evroc-logo.png" alt="evroc" width="200"/>

# Terraform Provider for evroc

Official Terraform provider for the evroc Cloud Platform

Manage your evroc infrastructure as code with [Terraform](https://www.terraform.io/) or [OpenTofu](https://opentofu.org/)

[![Tests](https://github.com/evroc-oss/terraform-provider-evroc/actions/workflows/ci.yml/badge.svg)](https://github.com/evroc-oss/terraform-provider-evroc/actions/workflows/ci.yml)
[![Release](https://github.com/evroc-oss/terraform-provider-evroc/actions/workflows/release.yml/badge.svg)](https://github.com/evroc-oss/terraform-provider-evroc/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/evroc-oss/terraform-provider-evroc)](https://goreportcard.com/report/github.com/evroc-oss/terraform-provider-evroc)
[![Go Version](https://img.shields.io/badge/go-1.25-blue)](https://github.com/evroc-oss/terraform-provider-evroc/blob/main/go.mod)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Terraform Registry](https://img.shields.io/badge/Terraform-Registry-purple.svg)](https://registry.terraform.io/providers/evroc-oss/evroc/latest)
[![OpenTofu Registry](https://img.shields.io/badge/OpenTofu-Registry-purple.svg)](https://github.com/opentofu/registry/blob/main/providers/e/evroc-oss/evroc.json)
[![Security](https://img.shields.io/badge/Security-Signed%20%26%20Attested-green.svg)](docs/VERIFICATION.md)
[![SLSA](https://img.shields.io/badge/SLSA-Provenance-blue.svg)](https://slsa.dev/)

</div>

---

Declaratively manage your evroc cloud infrastructure — provision VMs, configure networks, manage storage, and control access. Version control your infrastructure, preview changes before applying, and collaborate with your team.

> **OpenTofu compatible:** This provider works with both Terraform and [OpenTofu](https://opentofu.org/). All examples below use `terraform` commands, but `tofu` works identically.

## Capabilities

| Resource | What you can manage |
|----------|-------------------|
| **Compute** | Virtual machines, disks, disk attachments, placement groups |
| **Networking** | Public IPs, security groups |
| **Storage** | S3-compatible buckets and service accounts |
| **IAM** | Projects, permission sets |

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) 1.0+ or [OpenTofu](https://opentofu.org/) 1.9+
- evroc cloud account with API access
- Go 1.25+ (for building from source only)

## Installation

### Option 1: From Registry (Recommended)

The provider is published on both the [OpenTofu Registry](https://search.opentofu.org/provider/evroc-oss/evroc/latest) and the [Terraform Registry](https://registry.terraform.io/providers/evroc-oss/evroc/latest). No manual download needed — just declare it in your configuration.

**Configure your project** — create a `main.tf` file:

```hcl
terraform {
  required_providers {
    evroc = {
      source  = "evroc-oss/evroc"
      version = "~> 0.4"
    }
  }
}

provider "evroc" {}
```

Then initialize the provider:

```bash
terraform init
```

This downloads and installs the provider from the registry. No infrastructure is created yet — you'll define resources in the next sections, then iterate with `terraform plan` and `terraform apply` to preview and apply changes.

#### Upgrading the Provider

To upgrade to a newer version of the provider, update the `version` constraint in your `main.tf`:

```hcl
terraform {
  required_providers {
    evroc = {
      source  = "evroc-oss/evroc"
      version = "~> 0.4"  # Update to desired version
    }
  }
}
```

Then run:

```bash
terraform init -upgrade
```

This fetches the new version and updates the lock file (`.terraform.lock.hcl`). Review the [CHANGELOG](CHANGELOG.md) before upgrading for any breaking changes.

> **Tip:** Use `terraform providers lock` to pre-generate lock file entries for multiple platforms (useful for CI/CD and team collaboration).

---

### Option 2: Download from GitHub Releases

For air-gapped environments or when you need a specific binary.

<details>
<summary><strong>Linux / macOS</strong></summary>

```bash
# Fetch latest version automatically
VERSION=$(curl -s https://api.github.com/repos/evroc-oss/terraform-provider-evroc/releases/latest | grep '"tag_name"' | cut -d'"' -f4 | sed 's/^v//')
OS="linux"        # or: darwin
ARCH="amd64"      # or: arm64

# Download and extract
curl -LO "https://github.com/evroc-oss/terraform-provider-evroc/releases/download/v${VERSION}/terraform-provider-evroc_${VERSION}_${OS}_${ARCH}.zip"
unzip terraform-provider-evroc_${VERSION}_${OS}_${ARCH}.zip

# Install to the local plugin directory
mkdir -p ~/.terraform.d/plugins/evroc-oss/evroc/${VERSION}/${OS}_${ARCH}/
mv terraform-provider-evroc_v${VERSION} ~/.terraform.d/plugins/evroc-oss/evroc/${VERSION}/${OS}_${ARCH}/
```

</details>

<details>
<summary><strong>Windows</strong></summary>

```powershell
$VERSION = (Invoke-RestMethod -Uri "https://api.github.com/repos/evroc-oss/terraform-provider-evroc/releases/latest").tag_name -replace '^v',''
New-Item -ItemType Directory -Force -Path "$env:APPDATA\terraform.d\plugins\evroc-oss\evroc\$VERSION\windows_amd64"
Move-Item terraform-provider-evroc_v$VERSION.exe "$env:APPDATA\terraform.d\plugins\evroc-oss\evroc\$VERSION\windows_amd64\"
```

</details>

---

### Option 3: Build from Source

For development or when you need the latest unreleased changes.

```bash
git clone https://github.com/evroc-oss/terraform-provider-evroc.git
cd terraform-provider-evroc
make install
```

This builds the provider and installs it to `~/.terraform.d/plugins/github.com/evroc-oss/evroc/<VERSION>/linux_amd64/` (or `%APPDATA%\terraform.d\plugins\` on Windows), where `<VERSION>` is the Makefile's `VERSION` variable.

**Configure Terraform to use the local build** — create (or update) `~/.terraformrc` (Linux/macOS) or `%APPDATA%\terraform.rc` (Windows) with a `dev_overrides` block. For OpenTofu, use `~/.tofurc` or `%APPDATA%\tofu.rc` instead.

```hcl
provider_installation {
  dev_overrides {
    "evroc-oss/evroc" = "/home/<your-user>/.terraform.d/plugins/evroc-oss/evroc/<VERSION>/linux_amd64"
  }
  direct {}
}
```

> **The version in the path must match the version used by `make install`.** Check the `VERSION` variable in the Makefile for the current default. If you override it (e.g. `make install VERSION=1.0.0`), update the path accordingly.

Adjust the path for your OS/architecture (e.g., `darwin_arm64` for Apple Silicon). On Windows, use forward slashes: `C:/Users/<your-user>/AppData/Roaming/terraform.d/plugins/...`.

> **Important:** When using `dev_overrides`, **skip `terraform init`** and go directly to `terraform plan` / `terraform apply`.

Use the same `main.tf` as Option 1 above, but omit the `version` constraint (dev_overrides bypasses version checks).

---

## Quick Start

### 1. Authenticate with evroc

The provider reads credentials automatically using the following priority:

1. **Explicit provider attributes** (in the `provider "evroc" {}` block)
2. **Environment variables** (`EVROC_TOKEN`, `EVROC_REFRESH_TOKEN`, etc.)
3. **evroc CLI config file** (`~/.evroc/config.yaml`)

#### Option A: Use the evroc CLI (Recommended)

Install the [evroc CLI](https://docs.evroc.com/cli.html) and log in:

```bash
evroc login
```

This creates `~/.evroc/config.yaml` with your credentials, project, and region. The Terraform provider reads this file automatically — no further configuration needed. Just use an empty provider block:

```hcl
provider "evroc" {}
```

> **Tip:** If you already use the evroc CLI, you're all set. The provider reads the same `~/.evroc/config.yaml` that `evroc login` creates. No additional setup is required.

#### Option B: Environment variables

Set credentials directly via environment variables. This is useful for CI/CD pipelines or when using service accounts:

```bash
# Token-based authentication (use evroc CLI to obtain tokens)
export EVROC_REFRESH_TOKEN="your-refresh-token"
export EVROC_PROJECT="your-project-uuid"
export EVROC_REGION="se-sto"

# OR username/password authentication (service accounts)
export EVROC_USERNAME="service-account-id"
export EVROC_PASSWORD="service-account-secret"
export EVROC_PROJECT="your-project-uuid"
export EVROC_REGION="se-sto"
```

Then use an empty provider block:

```hcl
provider "evroc" {}
```

#### Option C: Explicit provider attributes

```hcl
provider "evroc" {
  token         = var.evroc_token
  refresh_token = var.evroc_refresh_token
  project       = "your-project-id"
  region        = "se-sto"
  organization  = "your-organization-id"  # Only needed for IAM project creation
}
```

#### Option D: Custom config file (for CI/CD or service accounts)

Point the provider at a dedicated config file. This is useful for CI/CD pipelines or service account credentials that you don't want in `~/.evroc/config.yaml` (which is managed by `evroc login`):

```hcl
provider "evroc" {
  config_file = "/path/to/my-terraform-config.yaml"
}
```

The config file format:

```yaml
auth:
  # Token-based authentication
  refresh_token: "your-refresh-token"

  # OR username/password authentication (service accounts):
  # username: "service-account-id"
  # password: "service-account-secret"

context:
  project: "your-project-uuid"
  region: "se-sto"
  organization: "your-organization-uuid"  # Optional, only for IAM project creation
```

> **Note:** Do not edit `~/.evroc/config.yaml` manually — that file is managed by `evroc login`. If you need a custom config, create a separate file and reference it with `config_file`.

**Finding your IDs:** Your organization and project IDs are visible in the [evroc console](https://cloud.evroc.com) under the project settings page. You can also extract them from the console URL:
```
https://cloud.evroc.com/?rgFullPath=%2F{organization-id}%2F{project-id}
                                        ^^^^^^^^^^^^^^^^^   ^^^^^^^^^^
```

> **Note:** The console and CLI use *id* and *name* for what Terraform calls `name` and `display_name` on the `evroc_project` resource.

### 2. Create Your First VM

Create a file named `main.tf`:

```hcl
terraform {
  required_providers {
    evroc = {
      source  = "evroc-oss/evroc"
      version = "~> 0.4"
    }
  }
}

# Credentials are read automatically from ~/.evroc/config.yaml (created by `evroc login`)
provider "evroc" {}

# Query available disk images and compute profiles (optional)
data "evroc_disk_images" "available" {}
data "evroc_compute_profiles" "available" {}

# Create a security group allowing SSH access
resource "evroc_security_group" "allow_ssh" {
  name = "allow-ssh"

  rule {
    name      = "allow-ssh"
    direction = "Ingress"
    protocol  = "TCP"
    port      = 22
    remote_ip = "0.0.0.0/0"
  }

  rule {
    name      = "allow-all-egress"
    direction = "Egress"
    protocol  = "TCP"
    port      = 0  # 0 means all ports
    remote_ip = "0.0.0.0/0"
  }
}

# Create a public IP for the VM
resource "evroc_public_ip" "vm" {
  name = "vm-public-ip"
}

# Create a boot disk with Ubuntu 24.04
resource "evroc_disk" "boot" {
  name  = "vm-boot-disk"
  size  = 20
  image = data.evroc_disk_images.available.ubuntu_minimal_24_04_1
  zone  = "a"
}

# Create the virtual machine
resource "evroc_virtual_machine" "web" {
  name      = "web-server"
  flavor    = data.evroc_compute_profiles.available.a1a_s
  boot_disk = evroc_disk.boot.fqid
  zone      = "a"

  # Attach public IP
  public_ip = evroc_public_ip.vm.fqid

  # Attach security group
  security_groups = [evroc_security_group.allow_ssh.fqid]

  # Add your SSH public key
  ssh_keys = [
    "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI... user@example.com"
  ]

  # Custom cloud-init replaces the default, so you must include the SSH user
  # setup to keep SSH access working. Add your packages/files after the user block.
  cloud_config_user_data = <<-EOF
    ## template: jinja
    #cloud-config
    ssh_pwauth: false
    manage_etc_hosts: localhost
    users:
      - name: evroc-user
        gecos: evroc VM user
        lock_passwd: true
        sudo: ALL=(ALL) NOPASSWD:ALL
        groups:
          - sudo
        shell: /bin/bash
    {% if public_ssh_keys %}
        ssh_authorized_keys:
    {% for pubkey in public_ssh_keys %}
          - {{ pubkey }}
    {% endfor %}
    {% endif %}
    package_update: true
    packages:
      - nginx
    write_files:
      - path: /var/www/html/index.html
        content: "<h1>Hello from evroc!</h1>"
  EOF
}

# Output the public IP address and SSH command
output "web_server_ip" {
  value       = evroc_virtual_machine.web.public_ipv4_address
  description = "Public IP address of the web server"
}

output "ssh_command" {
  value       = "ssh evroc-user@${evroc_virtual_machine.web.public_ipv4_address}"
  description = "SSH command to connect to the VM"
}
```

### 3. Deploy Your Infrastructure

```bash
# Preview what will be created
# Preview what will be created
terraform plan

# Apply the configuration to create resources
terraform apply

# When prompted, type 'yes' to confirm
```

After a few minutes, your VM will be running! You can SSH into it:

```bash
# Get the SSH command from the output
# Get the SSH command from Terraform output
terraform output -raw ssh_command

# Or manually
ssh evroc-user@$(terraform output -raw web_server_ip)
```

### 4. Clean Up

When you're done testing:

```bash
# Destroy all resources
# Destroy all resources created by Terraform
terraform destroy
```

## Examples

Complete, production-ready examples in the [`examples/`](examples/) directory:

- **[basic](examples/basic/)** - Complete VM with public IP and security group (great starting point)
- **[virtual-machine](examples/virtual-machine/)** - Simple VM with nginx
- **[complete](examples/complete/)** - Full production setup with web server and cloud-init
- **[k3s-cluster](examples/k3s-cluster/)** - Kubernetes cluster with control plane and workers
- **[networking](examples/networking/)** - Security groups and public IP management
- **[multi-project](examples/multi-project/)** - Security policy shared across multiple projects using provider aliases
- **[storage](examples/storage/)** - S3-compatible buckets with service accounts
- **[disk](examples/disk/)** - Disk creation and management
- **[disk-attachment](examples/disk-attachment/)** - Hot-attach disks to running VMs
- **[placement-groups](examples/placement-groups/)** - VM placement strategies for high availability
- **[public-ip](examples/public-ip/)** - Public IP allocation
- **[project](examples/project/)** - Project management

## Documentation

### Provider Configuration

The provider supports multiple authentication methods (see [Quick Start](#1-authenticate-with-evroc) for details).

**Environment variable reference:**

| Provider Attribute | Environment Variable | Notes |
|-------------------|---------------------|-------|
| `token` | `EVROC_TOKEN` | Access token |
| `refresh_token` | `EVROC_REFRESH_TOKEN` | For automatic token renewal |
| `username` | `EVROC_USERNAME` | Service account auth |
| `password` | `EVROC_PASSWORD` | Service account auth |
| `project` | `EVROC_PROJECT` | Required |
| `region` | `EVROC_REGION` | Defaults to `se-sto` |
| `organization` | `EVROC_ORGANIZATION` | Only for IAM project creation |
| `config_file` | `EVROC_CONFIG_FILE` | Path to SDK config YAML |
| `api_endpoint` | `EVROC_API_ENDPOINT` | Defaults to `https://api.cloud.evroc.com` |

### Remote State Backend (evroc S3)

You can use an evroc S3-compatible bucket to store your state remotely. This enables team collaboration and state locking. Works with both OpenTofu and Terraform.

**1. Create a bucket and service account** (via Terraform or the console), then configure the backend:

```hcl
terraform {
  backend "s3" {
    bucket = "<bucket-name>"
    key    = "terraform.tfstate"
    region = "se-sto"

    endpoints = {
      s3 = "https://s3.se-sto.evroc.com"
    }

    access_key = "<access-key>"
    secret_key = "<secret-key>"

    skip_credentials_validation = true
    skip_metadata_api_check     = true
    skip_region_validation      = true
    skip_requesting_account_id  = true
    skip_s3_checksum            = true
    use_path_style              = true
  }
}
```

**2. Alternatively, set credentials via environment variables** to keep them out of your configuration:

```bash
export AWS_ACCESS_KEY_ID="<access-key>"
export AWS_SECRET_ACCESS_KEY="<secret-key>"
```

Then omit `access_key` and `secret_key` from the backend block.

> **Note:** The `skip_*` flags are required because evroc S3 is not AWS — they disable AWS-specific validation that would otherwise fail.

### Resources

| Resource | Description |
|----------|-------------|
| `evroc_virtual_machine` | Virtual machine instances with networking |
| `evroc_disk` | Persistent block storage volumes |
| `evroc_hotswap_disk_attachment` | Attach disks to VMs (hot-attach supported) |
| `evroc_public_ip` | Public IPv4 addresses |
| `evroc_security_group` | Network firewall rules (ingress/egress) |
| `evroc_placement_group` | VM placement strategies (spread) |
| `evroc_bucket` | S3-compatible object storage |
| `evroc_bucket_service_account` | S3 access credentials |
| `evroc_project` | Project management |

### Data Sources

| Data Source | Description |
|-------------|-------------|
| `evroc_disk_images` | List available OS images with named attributes |
| `evroc_compute_profiles` | List VM sizes/flavors with named attributes |
| `evroc_virtual_machine` | Look up existing VMs |
| `evroc_disk` | Look up existing disks |
| `evroc_hotswap_disk_attachment` | Look up existing disk attachments |
| `evroc_public_ip` | Look up existing public IPs |
| `evroc_security_group` | Look up existing security groups |
| `evroc_placement_group` | Look up existing placement groups |
| `evroc_bucket` | Look up existing buckets |
| `evroc_bucket_service_account` | Look up existing service accounts |
| `evroc_project` | Look up existing projects |

### Key Fields Reference

#### Zone (Required)
All disk and VM resources require a `zone` field:
- `zone = "a"` - Zone a
- `zone = "b"` - Zone b
- `zone = "c"` - Zone c

#### Image Names
Use data source for current images:
```hcl
data "evroc_disk_images" "available" {}

resource "evroc_disk" "boot" {
  image = data.evroc_disk_images.available.ubuntu_minimal_24_04_1
  # Other options: ubuntu_22_04_1, rocky_9_6_1, opensuse_15_6_1, etc.
}
```

#### VM Flavors
Use data source for available sizes:
```hcl
data "evroc_compute_profiles" "available" {}

resource "evroc_virtual_machine" "vm" {
  flavor = data.evroc_compute_profiles.available.a1a_s
  # Other options: a1a_m, a1a_l, c1a_s, m1a_s, etc.
}
```

#### Security Group Rules
```hcl
resource "evroc_security_group" "example" {
  name = "example-sg"

  rule {
    name      = "allow-https"
    direction = "Ingress"  # or "Egress"
    protocol  = "TCP"      # or "UDP"
    port      = 443        # 0 means all ports
    remote_ip = "0.0.0.0/0"
  }
}
```

### Testing

See [TESTING.md](TESTING.md) for comprehensive testing documentation.

### Importing Existing Infrastructure

Already have evroc resources deployed? You can bring them under management without recreating anything.

**Option A: Automated discovery (recommended)**

Use the included script to enumerate all resources in your project and generate import blocks:

```bash
# Requires: evroc CLI installed and authenticated (evroc login)
./scripts/generate-imports.sh ./my-project

# Output:
#   my-project/provider.tf   - Provider configuration
#   my-project/imports.tf    - Import blocks for all discovered resources

# Then generate the .tf configuration:
cd my-project
terraform plan -generate-config-out=generated.tf

# Review generated.tf, adjust as needed, then:
terraform apply
```

Set `EVROC_CLI` if the binary isn't in your `PATH`:

```bash
EVROC_CLI=/path/to/evroc ./scripts/generate-imports.sh ./my-project
```

**Option B: Manual import (single resources)**

For importing individual resources, use the native `import` block:

```hcl
import {
  to = evroc_virtual_machine.web
  id = "my-vm-name"
}
```

Then run:

```bash
terraform plan -generate-config-out=generated.tf
```

The provider's Read function is called and the full resource block is written into `generated.tf`.

## Features

- **Declarative Infrastructure** - Define your desired state, the tool handles the rest
- **Plan Before Apply** - Preview changes before making them
- **Dependency Management** - Automatic resource dependency resolution
- **State Management** - Track infrastructure state and detect drift
- **Import Existing Resources** - Bring existing evroc resources under Terraform management
- **Parallel Execution** - Create/update/delete resources concurrently when possible
- **Type Safety** - Validate configuration before deployment
- **Token Auto-Refresh** - Automatic token refresh using refresh tokens
- **Resource Waiters** - Built-in waiting for async operations (VM/disk creation)

## Security

All evroc Terraform Provider releases are cryptographically signed and attested to ensure authenticity and integrity.

### Release Security

Every release includes:

- **GPG Signatures** - Required by Terraform Registry, signs the SHA256SUMS file (`*.sig`)
- **Cosign Signatures** - Keyless signing using GitHub OIDC (`*.cosign.sig`)
- **SBOM** - Software Bill of Materials for dependency transparency
- **SLSA Provenance** - Build integrity attestations proving official build process

### Verify Before Use

**Quick verification with Cosign (recommended):**

```bash
# Fetch latest version automatically
VERSION="v$(curl -s https://api.github.com/repos/evroc-oss/terraform-provider-evroc/releases/latest | grep '"tag_name"' | cut -d'"' -f4 | sed 's/^v//')"
BASE_URL="https://github.com/evroc-oss/terraform-provider-evroc/releases/download/${VERSION}"

# Download checksums and signature
curl -LO "${BASE_URL}/terraform-provider-evroc_${VERSION#v}_SHA256SUMS"
curl -LO "${BASE_URL}/terraform-provider-evroc_${VERSION#v}_SHA256SUMS.cosign.sig"
curl -LO "${BASE_URL}/terraform-provider-evroc_${VERSION#v}_SHA256SUMS.pem"

# Verify (requires cosign: brew install cosign)
cosign verify-blob \
  terraform-provider-evroc_${VERSION#v}_SHA256SUMS \
  --signature terraform-provider-evroc_${VERSION#v}_SHA256SUMS.cosign.sig \
  --certificate terraform-provider-evroc_${VERSION#v}_SHA256SUMS.pem \
  --certificate-identity-regexp="^https://github.com/evroc-oss/terraform-provider-evroc/" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com"
```

**For complete verification instructions, see [docs/VERIFICATION.md](docs/VERIFICATION.md)**

### Reporting Security Issues

If you discover a security vulnerability, please contact security@evroc.com. Do not open public issues for security vulnerabilities.

---

## Repository Layout

```
.
├── main.go                      # Provider entry point
├── internal/provider/           # Provider implementation
│   ├── provider.go              # Provider configuration and client setup
│   ├── resource_*.go            # Resource implementations
│   ├── data_source_*.go         # Data source implementations
│   ├── resource_*_test.go       # Acceptance tests per resource
│   ├── helpers.go               # SDK builder helper functions
│   ├── utils.go                 # Shared utilities
│   └── validators.go            # Input validators
├── docs/                        # Generated documentation (tfplugindocs)
│   ├── resources/               # Resource documentation
│   └── data-sources/            # Data source documentation
├── examples/                    # Terraform example configurations
├── scripts/                     # Build and release scripts
├── .github/workflows/           # GitHub Actions CI/CD
├── .goreleaser.yml              # GoReleaser configuration
├── CHANGELOG.md                 # Version history (Keep a Changelog)
├── TESTING.md                   # Acceptance testing guide
└── LICENSE                      # Apache 2.0
```

## Development

### Building from Source

```bash
# Clone the repository
git clone https://github.com/evroc-oss/terraform-provider-evroc.git
cd terraform-provider-evroc

# Download dependencies
make deps

# Build the provider
make build

# Run tests
make test

# Run linters
make lint

# Install locally for testing (see Installation > Option 1 for full setup)
make install

# Run acceptance tests (requires evroc credentials)
make testacc
```

### Available Make Targets

```bash
make build      # Build the provider binary
make install    # Install locally for testing
make test       # Run unit tests
make testacc    # Run acceptance tests
make fmt        # Format code
make lint       # Run linters
make vet        # Run go vet
make clean      # Clean build artifacts
make deps       # Download Go dependencies
make docs       # Generate documentation (requires tfplugindocs)
make coverage   # Run tests with coverage report
```

## Contributing

This project does not accept external contributions. If you encounter a bug or
have a feature request, please report it through
[evroc support](mailto:support@evroc.com).

## Support

**Support level:** Best-effort — evroc will address issues as time permits, with
no guaranteed SLA. All issues should be reported through
[evroc support channels](mailto:support@evroc.com).

- **Support:** [support@evroc.com](mailto:support@evroc.com)
- **Documentation:** [docs/](docs/)
- **Examples:** [examples/](examples/)
- **Go SDK:** [github.com/evroc-oss/evroc-go-sdk](https://github.com/evroc-oss/evroc-go-sdk)

## Versioning and Deprecation

This project follows [Semantic Versioning](https://semver.org/).

- Breaking changes only occur in major version releases.
- Deprecated features are announced at least one minor version in advance.
- Deprecation notices are documented in the [CHANGELOG](CHANGELOG.md) and may
  include runtime warnings.
- Security fixes are provided for the current major version and one prior major
  version.

If this project reaches end-of-life, the README will be updated with archived
status and a final release will be made.

## License

Apache License 2.0 - see [LICENSE](LICENSE) file
