#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
# SPDX-FileCopyrightText: 2026 evroc
#
# Generate Terraform import blocks from an existing evroc project.
# Enumerates all resources via the evroc CLI and outputs provider.tf + imports.tf
# ready for: terraform plan -generate-config-out=generated.tf
#
# Usage: ./scripts/generate-imports.sh [output-dir]
#        ./scripts/generate-imports.sh ./my-project-import
#
# Prerequisites:
#   - evroc CLI installed and authenticated (evroc login)
#   - ~/.evroc/config.yaml configured with project/region

set -euo pipefail

# --- Configuration ---
EVROC_CLI="${EVROC_CLI:-evroc}"
OUTPUT_DIR="${1:-.}"

# Verify evroc CLI is available
if ! command -v "$EVROC_CLI" &>/dev/null; then
    echo "Error: evroc CLI not found. Set EVROC_CLI to the path of the binary."
    echo "Example: EVROC_CLI=/usr/local/bin/evroc ./scripts/generate-imports.sh"
    exit 1
fi

mkdir -p "$OUTPUT_DIR"

echo "Discovering resources in evroc project..."
echo ""

# --- Helper: sanitize name to valid Terraform identifier ---
sanitize() {
    echo "$1" | sed 's/[^a-zA-Z0-9_]/_/g' | sed 's/^[0-9]/_&/'
}

# --- Enumerate resources ---

echo "  Listing virtual machines..."
VMS=$("$EVROC_CLI" compute virtualmachine list 2>/dev/null || true)

echo "  Listing disks..."
DISKS=$("$EVROC_CLI" compute disk list 2>/dev/null || true)

echo "  Listing placement groups..."
PGS=$("$EVROC_CLI" compute placementgroup list 2>/dev/null || true)

echo "  Listing public IPs..."
IPS=$("$EVROC_CLI" networking publicip list 2>/dev/null || true)

echo "  Listing security groups..."
SGS=$("$EVROC_CLI" networking securitygroup list 2>/dev/null || true)

echo "  Listing buckets..."
BUCKETS=$("$EVROC_CLI" storage bucket list 2>/dev/null || true)

echo "  Listing bucket service accounts..."
BUCKET_SAS=$("$EVROC_CLI" storage bucketserviceaccount list 2>/dev/null || true)

echo ""

# --- Count resources ---
count() { echo "$1" | grep -c . 2>/dev/null || echo 0; }

VM_COUNT=$(count "$VMS")
DISK_COUNT=$(count "$DISKS")
PG_COUNT=$(count "$PGS")
IP_COUNT=$(count "$IPS")
SG_COUNT=$(count "$SGS")
BUCKET_COUNT=$(count "$BUCKETS")
SA_COUNT=$(count "$BUCKET_SAS")
TOTAL=$((VM_COUNT + DISK_COUNT + PG_COUNT + IP_COUNT + SG_COUNT + BUCKET_COUNT + SA_COUNT))

echo "Found $TOTAL resources:"
echo "  Virtual Machines:       $VM_COUNT"
echo "  Disks:                  $DISK_COUNT"
echo "  Placement Groups:       $PG_COUNT"
echo "  Public IPs:             $IP_COUNT"
echo "  Security Groups:        $SG_COUNT"
echo "  Buckets:                $BUCKET_COUNT"
echo "  Bucket Service Accounts: $SA_COUNT"
echo ""

if [ "$TOTAL" -eq 0 ]; then
    echo "No resources found. Nothing to import."
    exit 0
fi

# --- Generate provider.tf ---
cat > "$OUTPUT_DIR/provider.tf" <<'EOF'
terraform {
  required_providers {
    evroc = {
      source = "registry.terraform.io/evroc/evroc"
    }
  }
}

provider "evroc" {}
EOF

echo "Generated $OUTPUT_DIR/provider.tf"

# --- Generate imports.tf ---
IMPORTS_FILE="$OUTPUT_DIR/imports.tf"
: > "$IMPORTS_FILE"

write_imports() {
    local resource_type="$1"
    local names="$2"
    local section_title="$3"

    if [ -z "$names" ]; then
        return
    fi

    echo "# ===================================================================" >> "$IMPORTS_FILE"
    echo "# $section_title" >> "$IMPORTS_FILE"
    echo "# ===================================================================" >> "$IMPORTS_FILE"
    echo "" >> "$IMPORTS_FILE"

    while IFS= read -r name; do
        [ -z "$name" ] && continue
        local tf_name
        tf_name=$(sanitize "$name")
        cat >> "$IMPORTS_FILE" <<EOF
import {
  to = ${resource_type}.${tf_name}
  id = "${name}"
}

EOF
    done <<< "$names"
}

write_imports "evroc_virtual_machine" "$VMS" "Virtual Machines"
write_imports "evroc_disk" "$DISKS" "Disks"
write_imports "evroc_placement_group" "$PGS" "Placement Groups"
write_imports "evroc_public_ip" "$IPS" "Public IPs"
write_imports "evroc_security_group" "$SGS" "Security Groups"
write_imports "evroc_bucket" "$BUCKETS" "Buckets"
write_imports "evroc_bucket_service_account" "$BUCKET_SAS" "Bucket Service Accounts"

echo "Generated $OUTPUT_DIR/imports.tf ($TOTAL import blocks)"
echo ""
echo "Next steps:"
echo "  cd $OUTPUT_DIR"
echo "  terraform plan -generate-config-out=generated.tf"
echo "  # Review generated.tf, then:"
echo "  terraform apply"
