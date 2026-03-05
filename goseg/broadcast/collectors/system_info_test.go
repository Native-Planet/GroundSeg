package collectors

import (
	"reflect"
	"testing"

	"groundseg/structs"
)

func TestCollectSystemInfoBuildsExpectedSnapshot(t *testing.T) {
	rt := collectorSystemRuntime{
		SystemUpdatesFn: func() structs.SystemUpdates {
			updates := structs.SystemUpdates{}
			updates.Linux.Upgrade = 3
			return updates
		},
		WiFiInfoSnapshotFn: func() structs.SystemWifi {
			return structs.SystemWifi{Status: true, Active: "Office", Networks: []string{"Office"}}
		},
		GetMemoryFn: func() (uint64, uint64, error) { return 11, 22, nil },
		GetCPUFn:    func() (int, error) { return 33, nil },
		GetTempFn:   func() (float64, error) { return 44.5, nil },
		GetDiskFn: func() (map[string][2]uint64, error) {
			return map[string][2]uint64{"/": {100, 50}}, nil
		},
		ListHardDisksFn: func() (structs.LSBLKDevice, error) {
			return structs.LSBLKDevice{
				BlockDevices: []structs.BlockDev{
					{Name: "sda", Mountpoints: []string{"/groundseg-2"}},
					{Name: "sdb", Mountpoints: []string{}},
					{Name: "mmcblk0", Mountpoints: []string{"/groundseg-9"}},
				},
			}, nil
		},
		IsDevMountedFn: func(dev structs.BlockDev) bool { return dev.Name == "sda" },
		SmartResultsSnapshotFn: func() map[string]bool {
			return map[string]bool{"sda": true}
		},
	}

	got, err := collectSystemInfo(rt, 128)
	if err != nil {
		t.Fatalf("collectSystemInfo returned error: %v", err)
	}

	if got.Info.Updates.Linux.Upgrade != 3 {
		t.Fatalf("unexpected updates snapshot: %+v", got.Info.Updates)
	}
	if !got.Info.Wifi.Status || got.Info.Wifi.Active != "Office" {
		t.Fatalf("unexpected wifi snapshot: %+v", got.Info.Wifi)
	}
	if !reflect.DeepEqual(got.Info.Usage.RAM, []uint64{11, 22}) {
		t.Fatalf("unexpected RAM usage: %+v", got.Info.Usage.RAM)
	}
	if got.Info.Usage.CPU != 33 || got.Info.Usage.CPUTemp != 44.5 {
		t.Fatalf("unexpected CPU usage snapshot: %+v", got.Info.Usage)
	}
	if got.Info.Usage.SwapFile != 128 {
		t.Fatalf("unexpected swap file value: %d", got.Info.Usage.SwapFile)
	}
	if got.Info.Drives["sda"].DriveID != 2 {
		t.Fatalf("expected mounted drive id parse for sda, got %+v", got.Info.Drives["sda"])
	}
	if got.Info.Drives["sdb"].DriveID != 0 {
		t.Fatalf("expected empty drive id for sdb, got %+v", got.Info.Drives["sdb"])
	}
	if _, exists := got.Info.Drives["mmcblk0"]; exists {
		t.Fatalf("expected mmc devices to be skipped from drive map")
	}
	if !got.Info.SMART["sda"] {
		t.Fatalf("expected SMART snapshot to be propagated, got %+v", got.Info.SMART)
	}
}
