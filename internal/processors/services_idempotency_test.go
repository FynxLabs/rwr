package processors

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/fynxlabs/rwr/internal/exectest"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// A delete already carried out on an earlier run is the state the blueprint
// asked for: reruns converge instead of failing.

func TestDeleteServiceFile_RerunConverges(t *testing.T) {
	file := filepath.Join(t.TempDir(), "foo.service")
	if err := os.WriteFile(file, []byte("[Unit]"), 0o644); err != nil {
		t.Fatal(err)
	}
	service := types.Service{Name: "foo", File: file}

	if err := deleteServiceFile(service); err != nil {
		t.Fatalf("first delete: %v", err)
	}
	if err := deleteServiceFile(service); err != nil {
		t.Fatalf("second delete must converge, got: %v", err)
	}
}

func TestDeleteLaunchDaemon_RerunConverges(t *testing.T) {
	file := filepath.Join(t.TempDir(), "com.foo.plist")
	if err := os.WriteFile(file, []byte("<plist/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	service := types.Service{Name: "foo", File: file}

	if err := deleteLaunchDaemon(service); err != nil {
		t.Fatalf("first delete: %v", err)
	}
	if err := deleteLaunchDaemon(service); err != nil {
		t.Fatalf("second delete must converge, got: %v", err)
	}
}

// sc delete fails for an unregistered service, so the delete is gated on an
// `sc query` probe: absent already means converged, not failed.
func TestDeleteWindowsService_AbsentServiceConverges(t *testing.T) {
	rec := exectest.New()
	rec.Err = errors.New("service does not exist")
	defer system.SetExecutor(rec)()

	err := deleteWindowsService(types.Service{Name: "foo"}, &types.InitConfig{})
	if err != nil {
		t.Fatalf("deleting an absent service must converge, got: %v", err)
	}
	// The probe ran; sc delete did not.
	if len(rec.Calls) != 1 || rec.Calls[0].Args[0] != "query" {
		t.Fatalf("recorded %v, want a single sc query probe", rec.Calls)
	}
}

func TestDeleteWindowsService_ExistingServiceIsDeleted(t *testing.T) {
	rec := exectest.New()
	defer system.SetExecutor(rec)()

	err := deleteWindowsService(types.Service{Name: "foo"}, &types.InitConfig{})
	if err != nil {
		t.Fatalf("deleteWindowsService: %v", err)
	}
	if len(rec.Calls) != 2 || rec.Calls[1].Args[0] != "delete" {
		t.Fatalf("recorded %v, want sc query then sc delete", rec.Calls)
	}
}

// sc create fails when the service already exists: the probe succeeding means
// converged, and the create is skipped.
func TestCreateWindowsService_ExistingServiceConverges(t *testing.T) {
	rec := exectest.New()
	defer system.SetExecutor(rec)()

	target := filepath.Join(t.TempDir(), "foo.exe")
	err := createWindowsService(types.Service{
		Name:    "foo",
		Content: "bin",
		Target:  target,
	}, &types.OSInfo{}, &types.InitConfig{})
	if err != nil {
		t.Fatalf("createWindowsService on an existing service must converge, got: %v", err)
	}
	if len(rec.Calls) != 1 || rec.Calls[0].Args[0] != "query" {
		t.Fatalf("recorded %v, want only the sc query probe", rec.Calls)
	}
}

func TestCreateWindowsService_MissingServiceIsCreated(t *testing.T) {
	rec := exectest.New()
	rec.Err = errors.New("service does not exist")
	defer system.SetExecutor(rec)()

	target := filepath.Join(t.TempDir(), "foo.exe")
	err := createWindowsService(types.Service{
		Name:    "foo",
		Content: "bin",
		Target:  target,
	}, &types.OSInfo{}, &types.InitConfig{})

	// The recorder fails every call, so sc create errors - what matters is
	// that it was attempted after the probe said the service is absent.
	if err == nil {
		t.Fatal("expected the failing sc create to surface")
	}
	if len(rec.Calls) != 2 || rec.Calls[1].Args[0] != "create" {
		t.Fatalf("recorded %v, want sc query then sc create", rec.Calls)
	}
}
