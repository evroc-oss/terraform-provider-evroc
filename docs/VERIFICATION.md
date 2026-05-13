# Artifact Verification Guide

This guide explains how to verify the authenticity and integrity of evroc Terraform Provider releases.

## Overview

All evroc Terraform Provider releases are cryptographically signed and attested:

1. **GPG Signing** - Signs the SHA256SUMS file (required by Terraform Registry) ✅
2. **Cosign Keyless Signing** - Signs checksums using GitHub OIDC (no keys to manage) ✅
3. **SBOM** - Software Bill of Materials for transparency ✅
4. **SLSA Provenance** - Build integrity attestation ✅

## Prerequisites

Install the required verification tools:

### Install Cosign

```bash
# macOS
brew install cosign

# Linux
wget "https://github.com/sigstore/cosign/releases/latest/download/cosign-linux-amd64"
sudo mv cosign-linux-amd64 /usr/local/bin/cosign
sudo chmod +x /usr/local/bin/cosign

# Verify installation
cosign version
```

## Verification Methods

### Method 1: Verify Cosign Signature (Recommended)

Cosign uses **keyless signing** via GitHub OIDC (no keys to manage).

**Download checksums and Cosign signature:**

```bash
VERSION="v0.4.2"  # Replace with desired version
BASE_URL="https://github.com/evroc-oss/terraform-provider-evroc/releases/download/${VERSION}"

curl -LO "${BASE_URL}/terraform-provider-evroc_${VERSION#v}_SHA256SUMS"
curl -LO "${BASE_URL}/terraform-provider-evroc_${VERSION#v}_SHA256SUMS.cosign.sig"
curl -LO "${BASE_URL}/terraform-provider-evroc_${VERSION#v}_SHA256SUMS.pem"
```

**Verify with Cosign:**

```bash
cosign verify-blob \
  terraform-provider-evroc_${VERSION#v}_SHA256SUMS \
  --signature terraform-provider-evroc_${VERSION#v}_SHA256SUMS.cosign.sig \
  --certificate terraform-provider-evroc_${VERSION#v}_SHA256SUMS.pem \
  --certificate-identity-regexp="^https://github.com/evroc-oss/terraform-provider-evroc/" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com"
```

**Expected output:**
```
Verified OK
```

**What this verifies:**
- ✅ Artifact was built by the official GitHub Actions workflow
- ✅ Build happened in the evroc-oss/terraform-provider-evroc repository
- ✅ Artifact has not been tampered with since signing

---

### Method 3: Verify SBOM (Software Bill of Materials)

The SBOM provides transparency about all dependencies and components.

**Download SBOM:**

```bash
VERSION="v0.4.2"
BASE_URL="https://github.com/evroc-oss/terraform-provider-evroc/releases/download/${VERSION}"

curl -LO "${BASE_URL}/sbom.spdx.json"
curl -LO "${BASE_URL}/sbom.spdx.json.sig"
curl -LO "${BASE_URL}/sbom.spdx.json.pem"
```

**Verify SBOM signature:**

```bash
cosign verify-blob \
  sbom.spdx.json \
  --signature sbom.spdx.json.sig \
  --certificate sbom.spdx.json.pem \
  --certificate-identity-regexp="^https://github.com/evroc-oss/terraform-provider-evroc/" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com"
```

**Inspect SBOM contents:**

```bash
# View as JSON
jq . sbom.spdx.json | less

# List all packages
jq -r '.packages[].name' sbom.spdx.json

# Find specific dependency
jq '.packages[] | select(.name | contains("terraform-plugin-sdk"))' sbom.spdx.json
```

**What the SBOM tells you:**
- All Go dependencies and their versions
- License information
- CVE information (if scanning tools are used)

---

### Method 4: Verify SLSA Provenance

SLSA provenance proves the artifact was built by the expected workflow in a trusted environment.

**Download provenance:**

```bash
VERSION="v0.4.2"
BASE_URL="https://github.com/evroc-oss/terraform-provider-evroc/releases/download/${VERSION}"

curl -LO "${BASE_URL}/provenance.json"
curl -LO "${BASE_URL}/provenance.json.sig"
curl -LO "${BASE_URL}/provenance.json.pem"
```

**Verify provenance signature:**

```bash
cosign verify-blob \
  provenance.json \
  --signature provenance.json.sig \
  --certificate provenance.json.pem \
  --certificate-identity-regexp="^https://github.com/evroc-oss/terraform-provider-evroc/" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com"
```

**Inspect provenance:**

```bash
jq . provenance.json
```

**What provenance tells you:**
- Exact Git commit SHA that was built
- Build workflow that was used
- Build start and finish timestamps
- Build environment metadata

---

## Complete Verification Example

Here's a complete script to verify a release:

```bash
#!/bin/bash
set -e

VERSION="v0.4.2"
PLATFORM="linux_amd64"
BASE_URL="https://github.com/evroc-oss/terraform-provider-evroc/releases/download/${VERSION}"

echo "🔐 Verifying evroc Terraform Provider ${VERSION} for ${PLATFORM}"

# Download artifacts
echo "📥 Downloading artifacts..."
curl -sLO "${BASE_URL}/terraform-provider-evroc_${VERSION#v}_${PLATFORM}.zip"
curl -sLO "${BASE_URL}/terraform-provider-evroc_${VERSION#v}_SHA256SUMS"
curl -sLO "${BASE_URL}/terraform-provider-evroc_${VERSION#v}_SHA256SUMS.cosign.sig"
curl -sLO "${BASE_URL}/terraform-provider-evroc_${VERSION#v}_SHA256SUMS.pem"

# Verify Cosign signature
echo "✍️  Verifying Cosign signature..."
cosign verify-blob \
  terraform-provider-evroc_${VERSION#v}_SHA256SUMS \
  --signature terraform-provider-evroc_${VERSION#v}_SHA256SUMS.cosign.sig \
  --certificate terraform-provider-evroc_${VERSION#v}_SHA256SUMS.pem \
  --certificate-identity-regexp="^https://github.com/evroc-oss/terraform-provider-evroc/" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com"

# Verify binary checksum
echo "🔍 Verifying binary checksum..."
grep "${PLATFORM}.zip" terraform-provider-evroc_${VERSION#v}_SHA256SUMS | sha256sum -c

echo "✅ Verification successful! The binary is authentic and unmodified."
```

---

## Automated Verification in CI/CD

You can automate verification in your CI/CD pipelines:

```yaml
# GitHub Actions example
- name: Verify Terraform Provider
  run: |
    VERSION="v0.4.2"
    BASE_URL="https://github.com/evroc-oss/terraform-provider-evroc/releases/download/${VERSION}"

    # Install cosign
    curl -LO https://github.com/sigstore/cosign/releases/latest/download/cosign-linux-amd64
    chmod +x cosign-linux-amd64

    # Download and verify
    curl -LO "${BASE_URL}/terraform-provider-evroc_${VERSION#v}_SHA256SUMS"
    curl -LO "${BASE_URL}/terraform-provider-evroc_${VERSION#v}_SHA256SUMS.cosign.sig"
    curl -LO "${BASE_URL}/terraform-provider-evroc_${VERSION#v}_SHA256SUMS.pem"

    ./cosign-linux-amd64 verify-blob \
      terraform-provider-evroc_${VERSION#v}_SHA256SUMS \
      --signature terraform-provider-evroc_${VERSION#v}_SHA256SUMS.cosign.sig \
      --certificate terraform-provider-evroc_${VERSION#v}_SHA256SUMS.pem \
      --certificate-identity-regexp="^https://github.com/evroc-oss/terraform-provider-evroc/" \
      --certificate-oidc-issuer="https://token.actions.githubusercontent.com"
```

---

## Reporting Security Issues

If you discover a security vulnerability, please email: **security@evroc.com**

Do **not** open public GitHub issues for security vulnerabilities.
