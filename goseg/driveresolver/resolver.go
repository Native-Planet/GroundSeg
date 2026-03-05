package driveresolver

import (
	"fmt"
	systemdisk "groundseg/system/disk"
	"regexp"
)

type Resolution struct {
	SelectedDrive   string
	Mountpoint      string
	NeedsFormatting bool
}

var (
	groundsegMountpointPattern = regexp.MustCompile(`^/groundseg-\d+$`)
	listHardDisks              = systemdisk.ListHardDisks
	createGroundSegFilesystem  = systemdisk.CreateGroundSegFilesystem
)

func Resolve(selectedDrive string) (Resolution, error) {
	resolution := Resolution{SelectedDrive: selectedDrive}
	if selectedDrive == "" || selectedDrive == "system-drive" {
		return resolution, nil
	}

	blockDevices, err := listHardDisks()
	if err != nil {
		return resolution, fmt.Errorf("retrieve block devices: %w", err)
	}

	for _, device := range blockDevices.BlockDevices {
		if device.Name != selectedDrive {
			continue
		}
		for _, mountpoint := range device.Mountpoints {
			if groundsegMountpointPattern.MatchString(mountpoint) {
				resolution.Mountpoint = mountpoint
				return resolution, nil
			}
		}
		resolution.NeedsFormatting = true
		return resolution, nil
	}

	return resolution, fmt.Errorf("selected drive %q not found", selectedDrive)
}

func EnsureReady(resolution Resolution) (Resolution, error) {
	if resolution.SelectedDrive == "" || resolution.SelectedDrive == "system-drive" || !resolution.NeedsFormatting {
		return resolution, nil
	}
	mountpoint, err := createGroundSegFilesystem(resolution.SelectedDrive)
	if err != nil {
		return resolution, fmt.Errorf("create groundseg filesystem on %s: %w", resolution.SelectedDrive, err)
	}
	resolution.Mountpoint = mountpoint
	resolution.NeedsFormatting = false
	return resolution, nil
}
