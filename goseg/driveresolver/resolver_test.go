package driveresolver

import (
	"errors"
	"testing"

	"groundseg/structs"
)

func TestResolveSystemDrive(t *testing.T) {
	resolution, err := Resolve("system-drive")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if resolution.Mountpoint != "" || resolution.NeedsFormatting {
		t.Fatalf("unexpected system-drive resolution: %+v", resolution)
	}
}

func TestResolveMountedGroundsegDrive(t *testing.T) {
	originalListHardDisks := listHardDisks
	defer func() {
		listHardDisks = originalListHardDisks
	}()
	listHardDisks = func() (structs.LSBLKDevice, error) {
		return structs.LSBLKDevice{
			BlockDevices: []structs.BlockDev{
				{Name: "sdb", Mountpoints: []string{"/groundseg-2"}},
			},
		}, nil
	}

	resolution, err := Resolve("sdb")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if resolution.Mountpoint != "/groundseg-2" || resolution.NeedsFormatting {
		t.Fatalf("unexpected mounted resolution: %+v", resolution)
	}
}

func TestResolveMarksFormattingWhenNoGroundsegMountpoint(t *testing.T) {
	originalListHardDisks := listHardDisks
	defer func() {
		listHardDisks = originalListHardDisks
	}()
	listHardDisks = func() (structs.LSBLKDevice, error) {
		return structs.LSBLKDevice{
			BlockDevices: []structs.BlockDev{
				{Name: "sdc", Mountpoints: []string{"/media/data"}},
			},
		}, nil
	}

	resolution, err := Resolve("sdc")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if !resolution.NeedsFormatting || resolution.Mountpoint != "" {
		t.Fatalf("unexpected resolution: %+v", resolution)
	}
}

func TestResolveErrorsWhenDriveMissing(t *testing.T) {
	originalListHardDisks := listHardDisks
	defer func() {
		listHardDisks = originalListHardDisks
	}()
	listHardDisks = func() (structs.LSBLKDevice, error) {
		return structs.LSBLKDevice{
			BlockDevices: []structs.BlockDev{
				{Name: "sdd", Mountpoints: []string{"/groundseg-4"}},
			},
		}, nil
	}

	if _, err := Resolve("sde"); err == nil {
		t.Fatal("expected Resolve to return missing-drive error")
	}
}

func TestEnsureReadyFormatsWhenNeeded(t *testing.T) {
	originalCreate := createGroundSegFilesystem
	defer func() {
		createGroundSegFilesystem = originalCreate
	}()
	createGroundSegFilesystem = func(string) (string, error) {
		return "/groundseg-7", nil
	}

	resolution, err := EnsureReady(Resolution{
		SelectedDrive:   "sdf",
		NeedsFormatting: true,
	})
	if err != nil {
		t.Fatalf("EnsureReady returned error: %v", err)
	}
	if resolution.NeedsFormatting || resolution.Mountpoint != "/groundseg-7" {
		t.Fatalf("unexpected ensure-ready resolution: %+v", resolution)
	}
}

func TestEnsureReadyReturnsFormatError(t *testing.T) {
	originalCreate := createGroundSegFilesystem
	defer func() {
		createGroundSegFilesystem = originalCreate
	}()
	createGroundSegFilesystem = func(string) (string, error) {
		return "", errors.New("format failed")
	}

	if _, err := EnsureReady(Resolution{
		SelectedDrive:   "sdg",
		NeedsFormatting: true,
	}); err == nil {
		t.Fatal("expected EnsureReady to return formatting error")
	}
}
