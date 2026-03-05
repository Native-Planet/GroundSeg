package collectors

import (
	"fmt"
	"groundseg/structs"
	"regexp"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

func collectSystemInfo(runtime collectorSystemRuntime, swapFile int) (structs.System, error) {
	var ramUsage []uint64
	var sysInfo structs.System

	if runtime.SystemUpdatesFn != nil {
		sysInfo.Info.Updates = runtime.SystemUpdatesFn()
	}
	if runtime.WiFiInfoSnapshotFn != nil {
		sysInfo.Info.Wifi = runtime.WiFiInfoSnapshotFn()
	}

	if runtime.GetMemoryFn != nil {
		if usedRam, totalRam, err := runtime.GetMemoryFn(); err != nil {
			zap.L().Warn(fmt.Sprintf("error getting memory usage: %v", err))
		} else {
			sysInfo.Info.Usage.RAM = append(ramUsage, usedRam, totalRam)
		}
	}
	if runtime.GetCPUFn != nil {
		if cpuUsage, err := runtime.GetCPUFn(); err != nil {
			zap.L().Warn(fmt.Sprintf("error getting CPU usage: %v", err))
		} else {
			sysInfo.Info.Usage.CPU = cpuUsage
		}
	}
	if runtime.GetTempFn != nil {
		if cpuTemp, err := runtime.GetTempFn(); err != nil {
			zap.L().Warn(fmt.Sprintf("error reading CPU temperature: %v", err))
		} else {
			sysInfo.Info.Usage.CPUTemp = cpuTemp
		}
	}
	if runtime.GetDiskFn != nil {
		if diskUsage, err := runtime.GetDiskFn(); err != nil {
			zap.L().Warn(fmt.Sprintf("error getting disk usage: %v", err))
		} else {
			sysInfo.Info.Usage.Disk = diskUsage
		}
	}
	sysInfo.Info.Usage.SwapFile = swapFile

	drives := make(map[string]structs.SystemDrive)
	if runtime.ListHardDisksFn != nil {
		if blockDevices, err := runtime.ListHardDisksFn(); err != nil {
			zap.L().Warn(fmt.Sprintf("error getting block devices: %v", err))
		} else {
			for _, dev := range blockDevices.BlockDevices {
				if strings.HasPrefix(dev.Name, "mmcblk") {
					continue
				}
				if len(dev.Children) < 1 {
					if runtime.IsDevMountedFn != nil && runtime.IsDevMountedFn(dev) {
						re := regexp.MustCompile(`^/groundseg-(\d+)$`)
						matches := re.FindStringSubmatch(dev.Mountpoints[0])
						if len(matches) > 1 {
							n, err := strconv.Atoi(matches[1])
							if err != nil {
								continue
							}
							drives[dev.Name] = structs.SystemDrive{DriveID: n}
						}
					} else {
						drives[dev.Name] = structs.SystemDrive{DriveID: 0}
					}
				}
			}
		}
	}
	sysInfo.Info.Drives = drives
	if runtime.SmartResultsSnapshotFn != nil {
		sysInfo.Info.SMART = runtime.SmartResultsSnapshotFn()
	}

	return sysInfo, nil
}
