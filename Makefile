.PHONY: build install test testacc fmt lint clean docs help deps coverage

HOSTNAME=github.com
NAMESPACE=evroc-oss
NAME=evroc
BINARY=terraform-provider-${NAME}
VERSION?=0.9.0
OS_ARCH?=linux_amd64

# Determine OS and architecture
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

ifeq ($(UNAME_S),Linux)
	OS=linux
endif
ifeq ($(UNAME_S),Darwin)
	OS=darwin
endif

ifeq ($(UNAME_M),x86_64)
	ARCH=amd64
endif
ifeq ($(UNAME_M),arm64)
	ARCH=arm64
endif
ifeq ($(UNAME_M),aarch64)
	ARCH=arm64
endif

OS_ARCH=${OS}_${ARCH}

default: help

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the provider binary
	go build -o ${BINARY} -ldflags="-X main.version=${VERSION}"

install: build ## Install the provider locally for testing
	mkdir -p ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}
	mv ${BINARY} ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}/

test: ## Run unit tests
	go test -v -cover -timeout=120s ./...

testacc: ## Run acceptance tests (requires EVROC credentials)
	TF_ACC=1 go test -v -cover -timeout=120m ./...

testaccopentofu: ## Run acceptance tests with OpenTofu (requires EVROC credentials and tofu binary)
	TF_ACC=1 TF_ACC_TERRAFORM_PATH=$$(which tofu) TF_ACC_PROVIDER_NAMESPACE=evroc TF_ACC_PROVIDER_HOST=registry.opentofu.org go test -v -cover -timeout=120m ./...

fmt: ## Format Go code and Terraform files
	go fmt ./...
	terraform fmt -recursive ./examples/

vet: ## Run go vet
	go vet ./...

lint: ## Run golangci-lint
	golangci-lint run ./...

clean: ## Clean build artifacts
	rm -f ${BINARY}
	rm -rf dist/

docs: ## Generate documentation
	@echo "Documentation generation requires tfplugindocs"
	@echo "Install: go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest"
	tfplugindocs generate --provider-name evroc

coverage: ## Run tests with coverage report
	go test -v -coverprofile=cover.out ./...
	go tool cover -func=cover.out

deps: ## Download Go dependencies
	go mod download
	go mod tidy

.PHONY: build install test testacc fmt lint clean docs help deps coverage
