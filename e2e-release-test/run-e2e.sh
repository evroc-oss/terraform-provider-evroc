#!/usr/bin/env bash
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$DIR"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

pass() { echo -e "${GREEN}PASS${NC} $1"; }
fail() { echo -e "${RED}FAIL${NC} $1"; FAILURES=$((FAILURES+1)); }
info() { echo -e "${YELLOW}==>${NC} $1"; }

FAILURES=0
RUN_ID=$(date +%y%m%d%H%M%S)
SSH_KEY=$(cat ~/.ssh/id_ed25519.pub 2>/dev/null || cat ~/.ssh/id_rsa.pub 2>/dev/null)
if [ -z "$SSH_KEY" ]; then
  fail "No SSH public key found in ~/.ssh/id_ed25519.pub or ~/.ssh/id_rsa.pub"
  exit 1
fi
export TF_VAR_ssh_public_key="$SSH_KEY"
TF_VAR="-var=run_id=${RUN_ID}"

info "Phase 0: Building provider (run_id=${RUN_ID})"
cd "$DIR/.."
make install
cd "$DIR"

# ─────────────────────────────────────────────
info "Phase 1: terraform apply"
rm -f terraform.tfstate terraform.tfstate.backup
terraform apply -auto-approve $TF_VAR

VM_IP=$(terraform output -raw vm_ip)
VM_STATUS=$(terraform output -raw vm_status)
LB_IP=$(terraform output -raw lb_ip)
info "VM IP: $VM_IP  Status: $VM_STATUS  LB IP: $LB_IP"
pass "All resources created"

# ─────────────────────────────────────────────
info "Phase 2: Verifying labels"
STATE=$(terraform show -json)

check_label() {
  local addr="$1" key="$2" expected="$3"
  actual=$(echo "$STATE" | jq -r ".values.root_module.resources[] | select(.address == \"$addr\") | .values.user_labels[\"$key\"] // empty")
  [ "$actual" = "$expected" ] && pass "$addr.$key=$expected" || fail "$addr.$key expected '$expected' got '$actual'"
}

# Existing resources
check_label "evroc_placement_group.spread" "managed-by" "terraform"
check_label "evroc_placement_group.spread" "component" "compute"
check_label "evroc_security_group.web" "component" "networking"
check_label "evroc_public_ip.vm" "component" "networking"
check_label "evroc_disk.boot" "component" "compute"
check_label "evroc_disk.data" "component" "storage"
check_label "evroc_virtual_machine.web" "component" "compute"
check_label "evroc_hotswap_disk_attachment.data" "component" "storage"
check_label "evroc_bucket.assets" "component" "storage"
check_label "evroc_bucket.logs" "component" "storage"
check_label "evroc_bucket_service_account.sa" "component" "storage"

# New resources
check_label "evroc_vpc.main" "component" "networking"
check_label "evroc_subnet.app" "component" "networking"
check_label "evroc_public_ip.lb" "component" "networking"
check_label "evroc_lb_backend_pool.app" "component" "networking"
check_label "evroc_lb_backend_service.http" "component" "networking"
check_label "evroc_lb_l4_route.http" "component" "networking"
check_label "evroc_loadbalancer.app" "component" "networking"

# ─────────────────────────────────────────────
info "Phase 3: Verifying computed attributes"

check_attr() {
  local addr="$1" attr="$2"
  val=$(echo "$STATE" | jq -r ".values.root_module.resources[] | select(.address == \"$addr\") | .values[\"$attr\"] // empty")
  [ -n "$val" ] && [ "$val" != "null" ] && pass "$addr.$attr=$val" || fail "$addr.$attr is empty"
}

# Existing resources
check_attr "evroc_disk.boot" "disk_id"
check_attr "evroc_disk.boot" "fqid"
check_attr "evroc_disk.boot" "created_at"
check_attr "evroc_virtual_machine.web" "vm_id"
check_attr "evroc_virtual_machine.web" "fqid"
check_attr "evroc_virtual_machine.web" "public_ipv4_address"
check_attr "evroc_virtual_machine.web" "private_ipv4_address"
check_attr "evroc_virtual_machine.web" "status"
check_attr "evroc_virtual_machine.web" "subnet_ref"
check_attr "evroc_public_ip.vm" "ip_id"
check_attr "evroc_public_ip.vm" "ip_address"
check_attr "evroc_public_ip.vm" "fqid"
check_attr "evroc_security_group.web" "sg_id"
check_attr "evroc_security_group.web" "fqid"
check_attr "evroc_placement_group.spread" "pg_id"
check_attr "evroc_placement_group.spread" "fqid"
check_attr "evroc_bucket.assets" "bucket_id"
check_attr "evroc_bucket.assets" "created_at"
check_attr "evroc_bucket.logs" "bucket_id"
check_attr "evroc_bucket_service_account.sa" "service_account_id"
check_attr "evroc_bucket_service_account.sa" "credentials_secret"
check_attr "evroc_hotswap_disk_attachment.data" "attachment_id"
check_attr "evroc_hotswap_disk_attachment.data" "serial"
check_attr "data.evroc_disk_images.images" "ubuntu_minimal_24_04_1"
check_attr "data.evroc_compute_profiles.profiles" "a1a_xs"

# VPC / Subnet
check_attr "evroc_vpc.main" "vpc_id"
check_attr "evroc_vpc.main" "fqid"
check_attr "evroc_vpc.main" "created_at"
check_attr "evroc_subnet.app" "subnet_id"
check_attr "evroc_subnet.app" "fqid"
check_attr "evroc_subnet.app" "created_at"

# Load Balancer
check_attr "evroc_lb_backend_pool.app" "pool_id"
check_attr "evroc_lb_backend_pool.app" "fqid"
check_attr "evroc_lb_backend_pool.app" "created_at"
check_attr "evroc_lb_backend_service.http" "service_id"
check_attr "evroc_lb_backend_service.http" "fqid"
check_attr "evroc_lb_backend_service.http" "backend_count"
check_attr "evroc_lb_l4_route.http" "route_id"
check_attr "evroc_lb_l4_route.http" "fqid"
check_attr "evroc_loadbalancer.app" "lb_id"
check_attr "evroc_loadbalancer.app" "fqid"
check_attr "evroc_loadbalancer.app" "public_ipv4_address"

# Data sources for new resources
check_attr "data.evroc_vpc.main" "vpc_id"
check_attr "data.evroc_vpc.main" "fqid"
check_attr "data.evroc_subnet.app" "subnet_id"
check_attr "data.evroc_subnet.app" "fqid"
check_attr "data.evroc_subnet.app" "vpc_ref"
check_attr "data.evroc_loadbalancer.app" "lb_id"
check_attr "data.evroc_loadbalancer.app" "public_ipv4_address"
check_attr "data.evroc_lb_backend_pool.app" "pool_id"
check_attr "data.evroc_lb_backend_service.http" "service_id"
check_attr "data.evroc_lb_backend_service.http" "backend_count"
check_attr "data.evroc_lb_l4_route.http" "route_id"

# ─────────────────────────────────────────────
info "Phase 4: Idempotency"
if terraform plan -detailed-exitcode $TF_VAR 2>&1; then
  pass "Plan shows no changes"
else
  EC=$?
  if [ $EC -eq 2 ]; then
    fail "Plan shows unexpected changes"
    terraform plan -no-color $TF_VAR 2>&1 | head -60
  else
    fail "Plan failed (exit $EC)"
  fi
fi

# ─────────────────────────────────────────────
info "Phase 5: Import test"

get_name() { terraform state show "$1" | grep 'name ' | head -1 | awk '{print $NF}' | tr -d '"'; }

PG_ID=$(get_name evroc_placement_group.spread)
SG_ID=$(get_name evroc_security_group.web)
PIP_ID=$(get_name evroc_public_ip.vm)
BOOT_ID=$(get_name evroc_disk.boot)
BUCKET_ID=$(get_name evroc_bucket.assets)
SA_ID=$(get_name evroc_bucket_service_account.sa)
VPC_ID=$(get_name evroc_vpc.main)
SUBNET_ID=$(get_name evroc_subnet.app)
POOL_ID=$(get_name evroc_lb_backend_pool.app)
BSVC_ID=$(get_name evroc_lb_backend_service.http)
ROUTE_ID=$(get_name evroc_lb_l4_route.http)
LB_ID=$(get_name evroc_loadbalancer.app)

terraform state rm evroc_placement_group.spread
terraform state rm evroc_security_group.web
terraform state rm evroc_public_ip.vm
terraform state rm evroc_disk.boot
terraform state rm evroc_bucket.assets
terraform state rm evroc_bucket_service_account.sa
terraform state rm evroc_vpc.main
terraform state rm evroc_subnet.app
terraform state rm evroc_lb_backend_pool.app
terraform state rm evroc_lb_backend_service.http
terraform state rm evroc_lb_l4_route.http
terraform state rm evroc_loadbalancer.app

terraform import $TF_VAR 'evroc_placement_group.spread' "$PG_ID"
terraform import $TF_VAR 'evroc_security_group.web' "$SG_ID"
terraform import $TF_VAR 'evroc_public_ip.vm' "$PIP_ID"
terraform import $TF_VAR 'evroc_disk.boot' "$BOOT_ID"
terraform import $TF_VAR 'evroc_bucket.assets' "$BUCKET_ID"
terraform import $TF_VAR 'evroc_bucket_service_account.sa' "$SA_ID"
terraform import $TF_VAR 'evroc_vpc.main' "$VPC_ID"
terraform import $TF_VAR 'evroc_subnet.app' "$SUBNET_ID"
terraform import $TF_VAR 'evroc_lb_backend_pool.app' "$POOL_ID"
terraform import $TF_VAR 'evroc_lb_backend_service.http' "$BSVC_ID"
terraform import $TF_VAR 'evroc_lb_l4_route.http' "$ROUTE_ID"
terraform import $TF_VAR 'evroc_loadbalancer.app' "$LB_ID"

pass "12 resources re-imported"

if terraform plan -detailed-exitcode $TF_VAR 2>&1; then
  pass "Post-import plan shows no changes"
else
  EC=$?
  if [ $EC -eq 2 ]; then
    fail "Post-import plan shows changes"
    terraform plan -no-color $TF_VAR 2>&1 | head -80
  else
    fail "Post-import plan failed (exit $EC)"
  fi
fi

# ─────────────────────────────────────────────
info "Phase 6: Destroy"
terraform destroy -auto-approve $TF_VAR
pass "All resources destroyed"

# ─────────────────────────────────────────────
echo ""
echo "=================================="
if [ $FAILURES -eq 0 ]; then
  echo -e "${GREEN}ALL CHECKS PASSED${NC}"
else
  echo -e "${RED}$FAILURES CHECK(S) FAILED${NC}"
fi
echo "=================================="
exit $FAILURES
