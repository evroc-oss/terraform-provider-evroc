#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
# SPDX-FileCopyrightText: 2026 evroc

# Terraform Provider Validation Script
# Runs comprehensive validation checks on the evroc Terraform provider

set -e

echo "========================================="
echo "evroc Terraform Provider Validation"
echo "========================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

passed=0
failed=0
warnings=0

# Function to print test result
print_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓ PASSED${NC}: $2"
        ((passed++))
    else
        echo -e "${RED}✗ FAILED${NC}: $2"
        ((failed++))
    fi
}

print_warning() {
    echo -e "${YELLOW}⚠ WARNING${NC}: $1"
    ((warnings++))
}

echo "1. Go Code Quality Checks"
echo "-----------------------------------------"

# Go Vet
if go vet ./... 2>&1 > /dev/null; then
    print_result 0 "go vet"
else
    print_result 1 "go vet"
fi

# Go Fmt
if [ -z "$(gofmt -l .)" ]; then
    print_result 0 "go fmt"
else
    print_result 1 "go fmt (files need formatting)"
fi

# Build
if go build -o terraform-provider-evroc -ldflags="-X main.version=0.1.0" 2>&1 > /dev/null; then
    print_result 0 "go build"
    rm -f terraform-provider-evroc
else
    print_result 1 "go build"
fi

echo ""
echo "2. Provider Installation"
echo "-----------------------------------------"

# Install provider
if make install 2>&1 > /dev/null; then
    print_result 0 "make install"
else
    print_result 1 "make install"
fi

echo ""
echo "3. Terraform Validation"
echo "-----------------------------------------"

# Test terraform init in storage example
cd examples/storage
if terraform init -upgrade 2>&1 > /dev/null; then
    print_result 0 "terraform init"
else
    print_result 1 "terraform init"
fi

# Test terraform validate
if terraform validate 2>&1 > /dev/null; then
    print_result 0 "terraform validate"
else
    print_result 1 "terraform validate"
fi
cd ../..

echo ""
echo "4. Provider Schema Validation"
echo "-----------------------------------------"

cd examples/storage

# Check resources
resource_count=$(terraform providers schema -json 2>/dev/null | jq -r '.provider_schemas["registry.terraform.io/evroc/evroc"].resource_schemas | keys | length')
if [ "$resource_count" = "8" ]; then
    print_result 0 "Resource schemas (8/8 registered)"
else
    print_result 1 "Resource schemas (expected 8, got $resource_count)"
fi

# Check data sources
datasource_count=$(terraform providers schema -json 2>/dev/null | jq -r '.provider_schemas["registry.terraform.io/evroc/evroc"].data_source_schemas | keys | length')
if [ "$datasource_count" = "8" ]; then
    print_result 0 "Data source schemas (8/8 registered)"
else
    print_result 1 "Data source schemas (expected 8, got $datasource_count)"
fi

# Check provider config
provider_attrs=$(terraform providers schema -json 2>/dev/null | jq -r '.provider_schemas["registry.terraform.io/evroc/evroc"].provider.block.attributes | keys | length')
if [ "$provider_attrs" = "5" ]; then
    print_result 0 "Provider configuration (5/5 attributes)"
else
    print_result 1 "Provider configuration (expected 5, got $provider_attrs)"
fi

cd ../..

echo ""
echo "5. Example Validation"
echo "-----------------------------------------"

# Validate all examples
for example in examples/*/; do
    example_name=$(basename "$example")
    cd "$example"
    if terraform init -upgrade 2>&1 > /dev/null && terraform validate 2>&1 > /dev/null; then
        print_result 0 "Example: $example_name"
    else
        print_result 1 "Example: $example_name"
    fi
    cd ../..
done

echo ""
echo "========================================="
echo "Validation Summary"
echo "========================================="
echo -e "${GREEN}Passed:${NC}   $passed"
echo -e "${RED}Failed:${NC}   $failed"
echo -e "${YELLOW}Warnings:${NC} $warnings"
echo ""

if [ $failed -eq 0 ]; then
    echo -e "${GREEN}✓ All validation checks passed!${NC}"
    echo "The provider is properly structured and ready for use."
    exit 0
else
    echo -e "${RED}✗ Some validation checks failed.${NC}"
    echo "Please review the failures above and fix issues."
    exit 1
fi
