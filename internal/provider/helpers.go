// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 evroc

package provider

import (
	"fmt"
	"path"
	"time"

	evroc "github.com/evroc-oss/evroc-go-sdk"
	"github.com/evroc-oss/evroc-go-sdk/compute"
	"github.com/evroc-oss/evroc-go-sdk/iam"
	"github.com/evroc-oss/evroc-go-sdk/loadbalancer"
	"github.com/evroc-oss/evroc-go-sdk/networking"
	"github.com/evroc-oss/evroc-go-sdk/storage"
	"github.com/evroc-oss/evroc-go-sdk/think"
	computetypes "github.com/evroc-oss/evroc-go-sdk/types/compute"
	iamtypes "github.com/evroc-oss/evroc-go-sdk/types/iam"
	lbtypes "github.com/evroc-oss/evroc-go-sdk/types/loadbalancer"
	networkingtypes "github.com/evroc-oss/evroc-go-sdk/types/networking"
	storagetypes "github.com/evroc-oss/evroc-go-sdk/types/storage"
	thinktypes "github.com/evroc-oss/evroc-go-sdk/types/think"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// BuildDiskCreateRequest creates a properly formatted Disk request using SDK builder
func BuildDiskCreateRequest(name string, sizeGB int, image, snapshot, zone string, userLabels map[string]string) *computetypes.DiskRequest {
	builder := compute.NewDiskBuilder(name).
		WithSizeGB(int32(sizeGB))

	if image != "" {
		builder = builder.WithImage(image)
	}

	if snapshot != "" {
		builder = builder.WithSnapshot(snapshot)
	}

	if zone != "" {
		builder = builder.WithZone(zone)
	}

	req := builder.Build()

	// Add user labels if provided
	if len(userLabels) > 0 {
		labels := make(computetypes.UserLabels)
		for k, v := range userLabels {
			labels[k] = v
		}
		req.Metadata.UserLabels = &labels
	}

	return req
}

// BuildPublicIPCreateRequest creates a properly formatted PublicIP request
func BuildPublicIPCreateRequest(name string, userLabels map[string]string) *networkingtypes.PublicIPRequest {
	req := networking.NewPublicIPBuilder(name).Build()

	// Add user labels if provided
	if len(userLabels) > 0 {
		labels := make(networkingtypes.UserLabels)
		for k, v := range userLabels {
			labels[k] = v
		}
		req.Metadata.UserLabels = &labels
	}

	return req
}

// BuildVirtualMachineCreateRequest creates a properly formatted VirtualMachine request using SDK builder.
// Reference fields (bootDisk, dataDisks, publicIP, securityGroups, placementGroup) accept either plain names
// or fully-qualified IDs (FQIDs). Plain names are resolved using the client's default project/region.
func BuildVirtualMachineCreateRequest(client *evroc.Client, name, flavor, bootDisk string, dataDisks []string, sshKeys []string, userData, publicIP, zone string, securityGroups []string, placementGroup, subnetRef, stackType string, running bool, userLabels map[string]string) *computetypes.VirtualMachineRequest {
	// Resolve boot disk ref: FQID pass-through or name → FQID
	diskRef := client.Compute().DiskRef(bootDisk)
	if isFQID(bootDisk) {
		diskRef = computetypes.DiskRef(bootDisk)
	}

	builder := compute.NewVirtualMachineBuilder(name).
		WithSize(flavor).
		WithBootDisk(diskRef).
		WithRunning(running)

	for _, dd := range dataDisks {
		ddRef := client.Compute().DiskRef(dd)
		if isFQID(dd) {
			ddRef = computetypes.DiskRef(dd)
		}
		builder = builder.WithDataDisk(ddRef)
	}

	for _, key := range sshKeys {
		builder = builder.WithSSHKey(key)
	}

	if userData != "" {
		builder = builder.WithCloudInit(userData)
	}

	if publicIP != "" {
		pipRef := client.Networking().PublicIPRef(publicIP)
		if isFQID(publicIP) {
			pipRef = networkingtypes.PublicIPRef(publicIP)
		}
		builder = builder.WithPublicIP(pipRef)
	}

	if zone != "" {
		builder = builder.WithZone(zone)
	}

	for _, sg := range securityGroups {
		sgRef := client.Networking().SecurityGroupRef(sg)
		if isFQID(sg) {
			sgRef = networkingtypes.SecurityGroupRef(sg)
		}
		builder = builder.WithSecurityGroup(sgRef)
	}

	if placementGroup != "" {
		pgRef := client.Compute().PlacementGroupRef(placementGroup)
		if isFQID(placementGroup) {
			pgRef = computetypes.PlacementGroupRef(placementGroup)
		}
		builder = builder.WithPlacementGroup(pgRef)
	}

	if subnetRef != "" {
		builder = builder.WithSubnet(subnetRef)
	}

	if stackType != "" {
		builder = builder.WithStackType(computetypes.VirtualMachineSpecNetworkingStackType(stackType))
	}

	req := builder.Build()

	// Add user labels if provided
	if len(userLabels) > 0 {
		labels := make(computetypes.UserLabels)
		for k, v := range userLabels {
			labels[k] = v
		}
		req.Metadata.UserLabels = &labels
	}

	return req
}

// isFQID returns true if the value looks like a fully-qualified resource ID
// (starts with "/"). Plain resource names don't start with "/".
func isFQID(s string) bool {
	return len(s) > 0 && s[0] == '/'
}

// suppressFQIDDiff is a DiffSuppressFunc that treats a plain resource name
// and its fully-qualified ID as equivalent. For example, "my-disk" and
// "/compute/projects/proj/regions/se-sto/disks/my-disk" are considered equal.
func suppressFQIDDiff(_, old, new string, _ *schema.ResourceData) bool {
	if old == new {
		return true
	}
	// If old is an FQID and new is a plain name, compare the base name
	if isFQID(old) && !isFQID(new) {
		return path.Base(old) == new
	}
	// If new is an FQID and old is a plain name, compare the base name
	if isFQID(new) && !isFQID(old) {
		return path.Base(new) == old
	}
	return false
}

// securityGroupHash normalizes security group references before hashing so
// that an FQID and the corresponding plain name produce the same hash value.
func securityGroupHash(v interface{}) int {
	s := v.(string)
	if isFQID(s) {
		s = path.Base(s)
	}
	return schema.HashString(s)
}

// derefString safely dereferences a string pointer
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// BuildSecurityGroupCreateRequest creates a properly formatted SecurityGroup request
func BuildSecurityGroupCreateRequest(name string, rules []networkingtypes.SecurityGroupSpecRulesItem, userLabels map[string]string) *networkingtypes.SecurityGroupRequest {
	builder := networking.NewSecurityGroupBuilder(name)

	// Add rules via builder's internal Build, then override with our rules
	req := builder.Build()
	if len(rules) > 0 {
		req.Spec.Rules = &rules
	}

	// Add user labels if provided
	if len(userLabels) > 0 {
		labels := make(networkingtypes.UserLabels)
		for k, v := range userLabels {
			labels[k] = v
		}
		req.Metadata.UserLabels = &labels
	}

	return req
}

// BuildPlacementGroupCreateRequest creates a properly formatted PlacementGroup request using SDK builder
func BuildPlacementGroupCreateRequest(name, strategy, zone string, userLabels map[string]string) *computetypes.PlacementGroupRequest {
	builder := compute.NewPlacementGroupBuilder(name, strategy)

	if zone != "" {
		builder = builder.WithZone(zone)
	}

	req := builder.Build()

	// Add user labels if provided
	if len(userLabels) > 0 {
		labels := make(computetypes.UserLabels)
		for k, v := range userLabels {
			labels[k] = v
		}
		req.Metadata.UserLabels = &labels
	}

	return req
}

// BuildDiskAttachmentCreateRequest creates a properly formatted HotswapDiskAttachment request using SDK builder.
// Reference fields (vmName, diskName) accept either plain names or FQIDs.
func BuildDiskAttachmentCreateRequest(client *evroc.Client, name, vmName, diskName string, userLabels map[string]string) *computetypes.HotswapDiskAttachmentRequest {
	vmRef := client.Compute().VMRef(vmName)
	if isFQID(vmName) {
		vmRef = computetypes.VMRef(vmName)
	}
	diskRef := client.Compute().DiskRef(diskName)
	if isFQID(diskName) {
		diskRef = computetypes.DiskRef(diskName)
	}
	req := compute.NewHotswapDiskAttachmentBuilder(name, vmRef, diskRef).Build()

	// Add user labels if provided
	if len(userLabels) > 0 {
		labels := make(computetypes.UserLabels)
		for k, v := range userLabels {
			labels[k] = v
		}
		req.Metadata.UserLabels = &labels
	}

	return req
}

// BuildBucketCreateRequest creates a properly formatted Bucket request using SDK builder
func BuildBucketCreateRequest(name, retentionMode, lockingMode string, lockingDuration int32, userLabels map[string]string) *storagetypes.BucketRequest {
	builder := storage.NewBucketBuilder(name)

	if retentionMode != "" && retentionMode != "Disabled" {
		builder = builder.WithObjectRetentionMode(retentionMode)
	}

	if lockingMode != "" && lockingDuration > 0 {
		builder = builder.WithDefaultObjectLocking(lockingMode, lockingDuration)
	}

	req := builder.Build()

	// Add user labels if provided
	if len(userLabels) > 0 {
		labels := make(storagetypes.UserLabels)
		for k, v := range userLabels {
			labels[k] = v
		}
		req.Metadata.UserLabels = &labels
	}

	return req
}

// BuildBucketServiceAccountCreateRequest creates a properly formatted BucketServiceAccount request using SDK builder
func BuildBucketServiceAccountCreateRequest(name string, buckets []string, userLabels map[string]string) *storagetypes.BucketServiceAccountRequest {
	builder := storage.NewBucketServiceAccountBuilder(name)

	for _, bucket := range buckets {
		builder = builder.WithBucket(bucket)
	}

	req := builder.Build()

	// Add user labels if provided
	if len(userLabels) > 0 {
		labels := make(storagetypes.UserLabels)
		for k, v := range userLabels {
			labels[k] = v
		}
		req.Metadata.UserLabels = &labels
	}

	return req
}

// BuildThinkInstanceCreateRequest creates a properly formatted Think Instance request using SDK builder
func BuildThinkInstanceCreateRequest(name, model, size string, stopped bool) *thinktypes.InstanceRequest {
	builder := think.NewInstanceBuilder(name).WithModel(model)

	if size != "" {
		builder = builder.WithSize(size)
	}

	if stopped {
		builder = builder.WithStopped(true)
	}

	return builder.Build()
}

// BuildThinkAPIKeyCreateRequest creates a properly formatted Think API key request using SDK builder
func BuildThinkAPIKeyCreateRequest(name, expiryStr string) (*thinktypes.ApikeyRequest, error) {
	builder := think.NewAPIKeyBuilder(name)

	if expiryStr != "" {
		expiry, err := time.Parse(time.RFC3339, expiryStr)
		if err != nil {
			return nil, fmt.Errorf("invalid expiry timestamp %q: %w", expiryStr, err)
		}
		builder = builder.WithExpiryTimestamp(expiry)
	}

	return builder.Build(), nil
}

// BuildProjectCreateRequest creates a properly formatted Project request using SDK builder
func BuildProjectCreateRequest(name, organization, displayName string, userLabels map[string]string) (*iamtypes.ProjectRequest, error) {
	builder := iam.NewProjectBuilder(name, organization)

	if displayName != "" {
		builder = builder.WithName(displayName)
	}

	if len(userLabels) > 0 {
		builder = builder.WithLabels(userLabels)
	}

	return builder.Build()
}

// BuildBackendPoolCreateRequest creates a properly formatted BackendPool request using SDK builder
func BuildBackendPoolCreateRequest(name string, backendRefs []string, userLabels map[string]string) *lbtypes.BackendpoolRequest {
	builder := loadbalancer.NewBackendPoolBuilder(name)

	if len(backendRefs) > 0 {
		builder = builder.WithBackendRefs(backendRefs)
	}

	req := builder.Build()

	if len(userLabels) > 0 {
		labels := make(lbtypes.UserLabels)
		for k, v := range userLabels {
			labels[k] = v
		}
		req.Metadata.UserLabels = &labels
	}

	return req
}

// BuildBackendServiceCreateRequest creates a properly formatted BackendService request using SDK builder
func BuildBackendServiceCreateRequest(name string, port int32, backendPoolRef string, proxyProtocol bool, userLabels map[string]string) *lbtypes.BackendserviceRequest {
	builder := loadbalancer.NewBackendServiceBuilder(name).
		WithPort(port).
		WithBackendPoolRef(backendPoolRef).
		WithProxyProtocol(proxyProtocol)

	req := builder.Build()

	if len(userLabels) > 0 {
		labels := make(lbtypes.UserLabels)
		for k, v := range userLabels {
			labels[k] = v
		}
		req.Metadata.UserLabels = &labels
	}

	return req
}

// BuildL4RouteCreateRequest creates a properly formatted L4Route request using SDK builder
func BuildL4RouteCreateRequest(name string, backendServiceRef string, userLabels map[string]string) *lbtypes.L4routeRequest {
	builder := loadbalancer.NewL4RouteBuilder(name).
		WithBackendServiceRef(backendServiceRef)

	req := builder.Build()

	if len(userLabels) > 0 {
		labels := make(lbtypes.UserLabels)
		for k, v := range userLabels {
			labels[k] = v
		}
		req.Metadata.UserLabels = &labels
	}

	return req
}
