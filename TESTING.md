# Acceptance Testing Guide

This guide explains how to run acceptance tests for the evroc Terraform provider.

## Prerequisites

1. **evroc Account** with API credentials
2. **Go** >= 1.25
3. **Make** (optional, for convenience)

## Setting Up Credentials

Acceptance tests require real evroc API credentials. **Never commit credentials to version control.**

### Authentication Methods

The provider supports two authentication methods:

#### Option 1: API Token (Recommended)

```bash
export EVROC_TOKEN="your-api-token"
export EVROC_ORGANIZATION="your-org-id"  # Optional
export EVROC_PROJECT="your-project-id"
export EVROC_REGION="se-sto"  # Optional, defaults to se-sto
```

#### Option 2: Username & Password

```bash
export EVROC_USERNAME="your-username"
export EVROC_PASSWORD="your-password"
export EVROC_PROJECT="your-project-id"
export EVROC_REGION="se-sto"  # Optional, defaults to se-sto
```

**Note:** Use either token OR username/password, not both.

### Using .envrc (with direnv)

1. Copy the example:
```bash
cp .envrc.example .envrc
```

2. Edit `.envrc` with your credentials (choose token OR username/password)

3. Load it:
```bash
# Install direnv first: https://direnv.net/
direnv allow .
```

**Important:** `.envrc` is already in `.gitignore` - never commit credentials!

## Running Tests

### All Acceptance Tests

```bash
make testacc
```

Or directly:

```bash
TF_ACC=1 go test ./internal/provider -v -timeout 120m
```

### Specific Resource Tests

```bash
# Test only disk resource
TF_ACC=1 go test ./internal/provider -v -run TestAccEvrocDisk

# Test only public IP resource
TF_ACC=1 go test ./internal/provider -v -run TestAccEvrocPublicIP

# Test only VM resource
TF_ACC=1 go test ./internal/provider -v -run TestAccEvrocVirtualMachine

# Test only security group resource
TF_ACC=1 go test ./internal/provider -v -run TestAccEvrocSecurityGroup
```

### Run a Single Test

```bash
TF_ACC=1 go test ./internal/provider -v -run TestAccEvrocDisk_Basic
```

## Test Timeouts

Acceptance tests can take time because they create real resources. Default timeout is 120 minutes.

Adjust if needed:

```bash
TF_ACC=1 go test ./internal/provider -v -timeout 180m
```

## What the Tests Do

Acceptance tests perform real operations against the evroc API:

1. **Create** - Provisions actual resources in your project
2. **Read** - Verifies the resource was created correctly
3. **Update** - Modifies the resource (if applicable)
4. **Read** - Verifies the update worked
5. **Delete** - Destroys the resource
6. **Verify Destroy** - Confirms the resource was deleted

### Resource Naming

All test resources are prefixed with `tf-test-` to identify them:

- `tf-test-disk-<random>`
- `tf-test-public-ip-<random>`
- `tf-test-vm-<random>`
- `tf-test-sg-<random>`

## Cost Awareness

⚠️ **Important:** Acceptance tests create real resources that may incur costs.

- Tests attempt to clean up after themselves
- Failed tests may leave orphaned resources
- Check your evroc console after tests to verify cleanup

### Manual Cleanup

If tests fail and leave resources:

```bash
# List all test resources
evroc-cli compute disks list | grep tf-test
evroc-cli networking public-ips list | grep tf-test
evroc-cli compute vms list | grep tf-test

# Delete manually if needed
evroc-cli compute disks delete tf-test-disk-xyz
```

## Test Coverage

Current acceptance tests:

- ✅ `evroc_disk` - Basic create, read, import, destroy
- ✅ `evroc_public_ip` - Basic create, read, import, destroy
- ✅ `evroc_virtual_machine` - Basic create, read, import, destroy
- ✅ `evroc_security_group` - Basic create with rules, read, import, destroy
- ✅ `evroc_placement_group` - Basic create, read, import, destroy
- ✅ `evroc_hotswap_disk_attachment` - Basic create with VM and disks, read, import, destroy
- ✅ `evroc_bucket` - Basic create, read, import, destroy
- ✅ `evroc_bucket_service_account` - Basic create with bucket dependency, read, import, destroy

## Debugging Failed Tests

### Enable Verbose Logging

```bash
TF_LOG=DEBUG TF_ACC=1 go test ./internal/provider -v -run TestAccEvrocDisk_Basic
```

### Check Test Logs

```bash
# Run with output to file
TF_ACC=1 go test ./internal/provider -v -run TestAccEvrocDisk_Basic 2>&1 | tee test.log
```

### Common Issues

**Error: "EVROC_USERNAME must be set"**
- Solution: Export environment variables before running tests

**Error: "context deadline exceeded"**
- Solution: Increase timeout with `-timeout 180m`

**Error: "resource still exists after destroy"**
- Solution: Check evroc console, may need manual cleanup

**Error: "authentication failed"**
- Solution: Verify credentials are correct

## CI/CD Integration

For automated testing in CI/CD:

```yaml
# Example: GitHub Actions
jobs:
  acceptance-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Run acceptance tests
        env:
          EVROC_USERNAME: ${{ secrets.EVROC_USERNAME }}
          EVROC_PASSWORD: ${{ secrets.EVROC_PASSWORD }}
          EVROC_PROJECT: ${{ secrets.EVROC_PROJECT }}
          EVROC_REGION: "se-sto"
          TF_ACC: "1"
        run: make testacc
```

## Best Practices

1. **Run tests in a dedicated project** - Use a separate evroc project for testing
2. **Check for orphaned resources** - Periodically audit test resources
3. **Don't run tests in production** - Never use production credentials
4. **Use cost alerts** - Set up billing alerts in your test project
5. **Clean up manually if needed** - Failed tests may leave resources

## Test Development

When adding new resources, create corresponding acceptance tests:

```go
func TestAccEvrocNewResource_Basic(t *testing.T) {
	resourceName := "evroc_new_resource.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckEvrocNewResourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEvrocNewResourceConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEvrocNewResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "tf-test-resource"),
					// Add more checks...
				),
			},
		},
	})
}
```

## Resources

- [Terraform Plugin SDK Testing](https://developer.hashicorp.com/terraform/plugin/sdkv2/testing)
- [Writing Acceptance Tests](https://developer.hashicorp.com/terraform/plugin/sdkv2/testing/acceptance-tests)
- [evroc API Documentation](https://docs.cloud.evroc.com)
