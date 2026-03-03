package system

import (
	"fmt"
	"groundseg/structs"
	"regexp"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

// CollectBroadcastSystemInfo aggregates system-level metrics used by broadcast state.
func CollectBroadcastSystemInfo(swapFile int) (structs.System, error) {
	var ramUsage []uint64
	var sysInfo structs.System

	sysInfo.Info.Updates = SystemUpdates
	sysInfo.Info.Wifi = WifiInfoSnapshot()

	if usedRam, totalRam, err := GetMemory(); err != nil {
		zap.L().Warn(fmt.Sprintf("Error getting memory usage: %v", err))
	} else {
		sysInfo.Info.Usage.RAM = append(ramUsage, usedRam, totalRam)
	}
	if cpuUsage, err := GetCPU(); err != nil {
		zap.L().Warn(fmt.Sprintf("Error getting CPU usage: %v", err))
	} else {
		sysInfo.Info.Usage.CPU = cpuUsage
	}
	if cpuTemp, err := GetTemp(); err != nil {
		zap.L().Warn(fmt.Sprintf("Error reading CPU temperature: %v", err))
	} else {
		sysInfo.Info.Usage.CPUTemp = cpuTemp
	}
	if diskUsage, err := GetDisk(); err != nil {
		zap.L().Warn(fmt.Sprintf("Error getting disk usage: %v", err))
	} else {
		sysInfo.Info.Usage.Disk = diskUsage
	}
	sysInfo.Info.Usage.SwapFile = swapFile

	drives := make(map[string]structs.SystemDrive)
	if blockDevices, err := ListHardDisks(); err != nil {
		zap.L().Warn(fmt.Sprintf("Error getting block devices: %v", err))
	} else {
		for _, dev := range blockDevices.BlockDevices {
			if strings.HasPrefix(dev.Name, "mmcblk") {
				continue
			}
			if len(dev.Children) < 1 {
				if IsDevMounted(dev) {
					re := regexp.MustCompile(`^/groundseg-(\d+)$`)
					matches := re.FindStringSubmatch(dev.Mountpoints[0])
					if len(matches) > 1 {
						n, err := strconv.Atoi(matches[1])
						if err != nil {
							continue
						}
						drives[dev.Name] = structs.SystemDrive{
							DriveID: n,
						}
					}
				} else {
					drives[dev.Name] = structs.SystemDrive{
						DriveID: 0,
					}
				}
			}
		}
	}
	sysInfo.Info.Drives = drives
	sysInfo.Info.SMART = SmartResultsSnapshot()

	return sysInfo, nil
}
