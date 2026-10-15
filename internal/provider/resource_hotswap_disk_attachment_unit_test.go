// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	computetypes "github.com/evroc-oss/evroc-go-sdk/types/compute"
)

// setupBusyDiskHandler serves test-disk as attached to old-vm for the first
// busyPolls reads (all reads if busyPolls < 0) and records when the
// attachment is created relative to those reads.
func setupBusyDiskHandler(ms *mockServer, busyPolls int) (reads func() int, readsAtCreate func() int) {
	var mu sync.Mutex
	n, atCreate := 0, -1
	ms.mux.HandleFunc("/compute/v1beta2/projects/test-project/regions/se-sto/disks/test-disk", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		n++
		busy := busyPolls < 0 || n <= busyPolls
		mu.Unlock()
		disk := mockDisk("test-disk")
		if busy {
			disk.Status.DiskUsage = &[]computetypes.DiskStatusDiskUsageItem{{
				UseKind:           "hotswap",
				VirtualMachineRef: "/compute/projects/test-project/regions/se-sto/virtualMachines/old-vm",
			}}
		}
		respondJSON(w, http.StatusOK, disk)
	})
	attachment := mockDiskAttachment("test-attach")
	ms.mux.HandleFunc("/compute/v1beta2/projects/test-project/regions/se-sto/hotswapDiskAttachments", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		atCreate = n
		mu.Unlock()
		respondJSON(w, http.StatusCreated, attachment)
	})
	ms.mux.HandleFunc("/compute/v1beta2/projects/test-project/regions/se-sto/hotswapDiskAttachments/test-attach", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusOK, attachment)
	})
	return func() int { mu.Lock(); defer mu.Unlock(); return n },
		func() int { mu.Lock(); defer mu.Unlock(); return atCreate }
}

func TestResourceHotswapDiskAttachmentCreateWaitsForDetach(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	_, readsAtCreate := setupBusyDiskHandler(ms, 2)
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	d := newTestResourceData(t, resourceHotswapDiskAttachment())
	d.Set("name", "test-attach")
	d.Set("virtual_machine", "new-vm")
	d.Set("disk", "test-disk")

	diags := resourceHotswapDiskAttachmentCreate(context.Background(), d, config)
	if diags.HasError() {
		t.Fatalf("unexpected create error: %v", diagnosticsToString(diags))
	}
	if got := readsAtCreate(); got != 3 {
		t.Fatalf("attachment created after %d disk reads, want 3 (two busy, one free)", got)
	}
}

func TestResourceHotswapDiskAttachmentCreateTimesOutWhileAttached(t *testing.T) {
	ms := newMockServer()
	defer ms.close()
	reads, readsAtCreate := setupBusyDiskHandler(ms, -1)
	setupCatchAll(ms)

	config := newTestProviderConfig(t, ms.server.URL)
	d := newTestResourceData(t, resourceHotswapDiskAttachment())
	d.Set("name", "test-attach")
	d.Set("virtual_machine", "new-vm")
	d.Set("disk", "test-disk")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	diags := resourceHotswapDiskAttachmentCreate(ctx, d, config)
	if !diags.HasError() {
		t.Fatal("expected an error while the disk stays attached")
	}
	if msg := diagnosticsToString(diags); !strings.Contains(msg, "still attached to old-vm") {
		t.Fatalf("error %q does not name the VM holding the disk", msg)
	}
	if readsAtCreate() != -1 {
		t.Fatal("attachment must not be created while the disk is attached elsewhere")
	}
	if reads() < 2 {
		t.Fatalf("disk read %d times, want polling", reads())
	}
}

func TestResourceHotswapDiskAttachmentCreateAlreadyAttachedToTarget(t *testing.T) {
	for _, vmName := range []string{"old-vm", "/compute/projects/test-project/regions/se-sto/virtualMachines/old-vm"} {
		t.Run(vmName, func(t *testing.T) {
			ms := newMockServer()
			defer ms.close()
			_, readsAtCreate := setupBusyDiskHandler(ms, -1)
			setupCatchAll(ms)
			config := newTestProviderConfig(t, ms.server.URL)
			d := newTestResourceData(t, resourceHotswapDiskAttachment())
			d.Set("name", "test-attach")
			d.Set("virtual_machine", vmName)
			d.Set("disk", "test-disk")
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			diags := resourceHotswapDiskAttachmentCreate(ctx, d, config)
			if diags.HasError() {
				t.Fatalf("unexpected create error: %v", diagnosticsToString(diags))
			}
			if got := readsAtCreate(); got != 1 {
				t.Fatalf("attachment created after %d disk reads, want 1", got)
			}
		})
	}
}
