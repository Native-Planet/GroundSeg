package routines

import (
	"fmt"
	"groundseg/click"
	"groundseg/config"
	"groundseg/structs"
	"groundseg/system"
	"math"
	"time"

	"go.uber.org/zap"
)

func DiskUsageWarning() {
	for {
		if diskUsage, err := system.GetDisk(); err != nil {
			zap.L().Error(fmt.Sprintf("Error getting disk usage: %v", err))
			continue
		} else {
			conf := config.Conf()
			for part, usage := range diskUsage {
				percentage := math.Round(float64(usage[0]) / float64(usage[1]) * 100 * 100 / 100) // 2 decimal places
				switch {
				case percentage < 80:
					// reset to default
					if err := setWarningInfo(conf, part, false, false, time.Time{}); err != nil {
						zap.L().Error(fmt.Sprintf("Failed to update disk warning info: %v", err))
					}
				case percentage >= 95:
					now := time.Now()
					// if time longer than 1 day, send noti and reset timer
					if info, exists := conf.DiskWarning[part]; !exists || now.Sub(info.NinetyFive) >= 24*time.Hour {
						// send hark notif
						sendDriveHarkNotification(part, percentage, conf.Piers)
						if err := setWarningInfo(conf, part, true, true, now); err != nil {
							zap.L().Error(fmt.Sprintf("Failed to update disk warning info: %v", err))
						}
					}
				case percentage >= 90:
					// if config is false, send noti and set to true
					if info, exists := conf.DiskWarning[part]; !exists || !info.Ninety {
						// send hark notif
						sendDriveHarkNotification(part, percentage, conf.Piers)
						if err := setWarningInfo(conf, part, true, true, time.Time{}); err != nil {
							zap.L().Error(fmt.Sprintf("Failed to update disk warning info: %v", err))
						}
					}
				case percentage >= 80:
					// if config is false, send noti and set to true
					if info, exists := conf.DiskWarning[part]; !exists || !info.Eighty {
						// send hark notif
						sendDriveHarkNotification(part, percentage, conf.Piers)
						if err := setWarningInfo(conf, part, true, false, time.Time{}); err != nil {
							zap.L().Error(fmt.Sprintf("Failed to update disk warning info: %v", err))
						}
					}
				default:
					if err := setWarningInfo(conf, part, false, false, time.Time{}); err != nil {
						zap.L().Error(fmt.Sprintf("Failed to update disk warning info: %v", err))
					}
				}
			}
		}
		time.Sleep(30 * time.Minute) // check every 30 minutes
	}
}

func setWarningInfo(conf structs.SysConfig, part string, eighty, ninety bool, ninetyFive time.Time) error {
	if conf.DiskWarning == nil {
		conf.DiskWarning = make(map[string]structs.DiskWarning)
	}
	conf.DiskWarning[part] = structs.DiskWarning{
		Eighty:     eighty,
		Ninety:     ninety,
		NinetyFive: ninetyFive,
	}
	if err := config.UpdateConf(map[string]interface{}{
		"diskWarning": conf.DiskWarning,
	}); err != nil {
		return fmt.Errorf("Couldn't set disk warning in config 80%:%v 90%:%v 95%:%v: %v", err, eighty, ninety, ninetyFive)
	}
	return nil
}

func sendDriveHarkNotification(part string, percentage float64, piers []string) {
	noti := structs.HarkNotification{Type: "disk-warning", DiskName: part, DiskUsage: percentage}
	// Send notification
	for _, patp := range piers {
		if err := click.SendNotification(patp, noti); err != nil {
			zap.L().Error(fmt.Sprintf("Failed to send drive warning to %s: %v", patp, err))
		}
	}
}
