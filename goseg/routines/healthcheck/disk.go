package healthcheck

import (
	"context"
	"fmt"
	"groundseg/click"
	"groundseg/config"
	"groundseg/structs"
	"groundseg/system"
	"math"
	"time"

	"go.uber.org/zap"
)

var (
	updateDiskWarningInConf = func(warnings map[string]structs.DiskWarning) error {
		return config.UpdateConfTyped(config.WithDiskWarning(warnings))
	}
	getHealthCheckSettings         = config.HealthCheckSettingsSnapshot
	sendNotificationForDiskWarning = click.SendNotification
)

func SmartDiskCheck() {
	for {
		disks, err := system.ListHardDisks()
		if err != nil {
			zap.L().Error(fmt.Sprintf("smart disk check error: %v", err))
		} else {
			res := system.SmartCheckAllDrives(disks)
			system.SetSmartResults(res)
			settings := getHealthCheckSettings()
			for k, v := range res {
				if !v {
					sendSmartFailHarkNotification(k, settings.Piers)
				}
			}
		}
		time.Sleep(6 * time.Hour)
		//time.Sleep(15 * time.Second) //dev
	}
}

func SmartDiskCheckWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		disks, err := system.ListHardDisks()
		if err != nil {
			zap.L().Error(fmt.Sprintf("smart disk check error: %v", err))
		} else {
			res := system.SmartCheckAllDrives(disks)
			system.SetSmartResults(res)
			settings := getHealthCheckSettings()
			for k, v := range res {
				if !v {
					sendSmartFailHarkNotification(k, settings.Piers)
				}
			}
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(6 * time.Hour):
		}
	}
}

func DiskUsageWarning() {
	for {
		if diskUsage, err := system.GetDisk(); err != nil {
			zap.L().Error(fmt.Sprintf("Error getting disk usage: %v", err))
			continue
		} else {
			evaluateDiskUsageWarnings(diskUsage)
		}
		time.Sleep(30 * time.Minute) // check every 30 minutes
	}
}

func DiskUsageWarningWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		if diskUsage, err := system.GetDisk(); err != nil {
			zap.L().Error(fmt.Sprintf("Error getting disk usage: %v", err))
		} else {
			evaluateDiskUsageWarnings(diskUsage)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(30 * time.Minute): // check every 30 minutes
		}
	}
}

func evaluateDiskUsageWarnings(diskUsage map[string][2]uint64) {
	settings := getHealthCheckSettings()
	for part, usage := range diskUsage {
		percentage := roundDiskUsage(float64(usage[0]) / float64(usage[1]) * 100)
		currentInfo, exists := settings.DiskWarnings[part]
		switch {
		case percentage < 80:
			// reset to default
			if err := setWarningInfo(settings, part, false, false, time.Time{}); err != nil {
				zap.L().Error(fmt.Sprintf("Failed to update disk warning info: %v", err))
			}
		case percentage >= 95:
			// if time longer than 1 day, send noti and reset timer
			now := time.Now()
			if !exists || now.Sub(currentInfo.NinetyFive) >= 24*time.Hour {
				// send hark notif
				sendDriveHarkNotification(part, percentage, settings.Piers)
				if err := setWarningInfo(settings, part, true, true, now); err != nil {
					zap.L().Error(fmt.Sprintf("Failed to update disk warning info: %v", err))
				}
			}
		case percentage >= 90:
			// if config is false, send noti and set to true
			if !exists || !currentInfo.Ninety {
				// send hark notif
				sendDriveHarkNotification(part, percentage, settings.Piers)
				if err := setWarningInfo(settings, part, true, true, time.Time{}); err != nil {
					zap.L().Error(fmt.Sprintf("Failed to update disk warning info: %v", err))
				}
			}
		case percentage >= 80:
			// if config is false, send noti and set to true
			if !exists || !currentInfo.Eighty {
				// send hark notif
				sendDriveHarkNotification(part, percentage, settings.Piers)
				if err := setWarningInfo(settings, part, true, false, time.Time{}); err != nil {
					zap.L().Error(fmt.Sprintf("Failed to update disk warning info: %v", err))
				}
			}
		default:
			if err := setWarningInfo(settings, part, false, false, time.Time{}); err != nil {
				zap.L().Error(fmt.Sprintf("Failed to update disk warning info: %v", err))
			}
		}
	}
}

func setWarningInfo(settings config.HealthCheckSettings, part string, eighty, ninety bool, ninetyFive time.Time) error {
	warnings := copyDiskWarningMap(settings.DiskWarnings)
	warnings[part] = structs.DiskWarning{
		Eighty:     eighty,
		Ninety:     ninety,
		NinetyFive: ninetyFive,
	}
	if err := updateDiskWarningInConf(warnings); err != nil {
		return fmt.Errorf("Couldn't set disk warning in config 80%%:%v 90%%:%v 95%%:%v: %v", eighty, ninety, ninetyFive, err)
	}
	return nil
}

func copyDiskWarningMap(source map[string]structs.DiskWarning) map[string]structs.DiskWarning {
	if source == nil {
		return map[string]structs.DiskWarning{}
	}
	warnings := make(map[string]structs.DiskWarning, len(source))
	for key, warning := range source {
		warnings[key] = warning
	}
	return warnings
}

func roundDiskUsage(value float64) float64 {
	return math.Round(value*100) / 100
}

func sendDriveHarkNotification(part string, percentage float64, piers []string) {
	noti := structs.HarkNotification{Type: "disk-warning", DiskName: part, DiskUsage: percentage}
	// Send notification
	for _, patp := range piers {
		if err := sendNotificationForDiskWarning(patp, noti); err != nil {
			zap.L().Error(fmt.Sprintf("Failed to send drive warning to %s: %v", patp, err))
		}
	}
}

func sendSmartFailHarkNotification(dev string, piers []string) {
	noti := structs.HarkNotification{Type: "smart-fail", DiskName: dev}
	// Send notification
	for _, patp := range piers {
		if err := sendNotificationForDiskWarning(patp, noti); err != nil {
			zap.L().Error(fmt.Sprintf("Failed to send smart check warning to %s: %v", patp, err))
		}
	}
}
