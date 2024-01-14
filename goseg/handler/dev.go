package handler

import (
	"encoding/json"
	"fmt"
	"groundseg/click"
	"groundseg/config"
	"groundseg/logger"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/system"
	"time"
)

func DevHandler(msg []byte) error {
	var devPayload structs.WsDevPayload
	err := json.Unmarshal(msg, &devPayload)
	if err != nil {
		return fmt.Errorf("Couldn't unmarshal dev payload: %v", err)
	}
	switch devPayload.Payload.Action {
	case "reset-setup":
		logger.Logger.Warn("Dev reset-setup not allowed!")
	case "print-mounts":
		if blockDevices, err := system.ListHardDisks(); err != nil {
			logger.Logger.Error(fmt.Sprintf("Failed to print block mounts: %v", err))
		} else {
			logger.Logger.Debug(fmt.Sprintf("lsblk: %+v", blockDevices))
		}
	case "startram-reminder":
		conf := config.Conf()
		if !conf.WgRegistered {
			return fmt.Errorf("No startram registration")
		}
		retrieve, err := startram.Retrieve()
		if err != nil {
			return fmt.Errorf("Failed to retrieve StarTram information: %v", err)
		}
		//retrieve.Ongoing != 0
		// Layout to parse the given date (must match the format of dateStr)
		layout := "2006-01-02"

		// Parse the date string into a time.Time object
		expiryDate, err := time.Parse(layout, retrieve.Lease)
		if err != nil {
			return fmt.Errorf("Failed to parse expiry date %v: %v", retrieve.Lease, err)
		}

		// Get the current time
		currentTime := time.Now()

		// Calculate the difference in days
		diff := expiryDate.Sub(currentTime).Hours() / 24
		noti := structs.HarkNotification{Type: "startram-reminder", StartramDaysLeft: int(diff)}

		rem := conf.StartramSetReminder
		if !rem.One || !rem.Three || !rem.Seven {
			// Send notification
			for _, patp := range conf.Piers {
				shipConf := config.UrbitConf(patp)
				if shipConf.StartramReminder == true {
					if err := click.SendNotification(patp, noti); err != nil {
						logger.Logger.Error(fmt.Sprintf("Failed to send dev startram reminder to %s: %v", patp, err))
					}
				}
			}
		} else {
			logger.Logger.Debug("Dev not sending startram reminder. Already reminded!")
		}
	case "startram-reminder-toggle":
		reminded := devPayload.Payload.Reminded
		if err := config.UpdateConf(map[string]interface{}{
			"startramSetReminder": map[string]bool{
				"one":   reminded,
				"three": reminded,
				"seven": reminded,
			},
		}); err != nil {
			logger.Logger.Error(fmt.Sprintf("Couldn't reset startram reminder: %v", err))
		}
	default:
		return fmt.Errorf("Unknown Dev action: %v", devPayload.Payload.Action)
	}
	return nil
}
