package routines

import (
	"fmt"
	"groundseg/click"
	"groundseg/config"
	"groundseg/startram"
	"groundseg/structs"
	"time"

	"go.uber.org/zap"
)

func StartramRenewalReminder() {
	for {
		zap.L().Debug(fmt.Sprintf("Checking StarTram renewal status.."))
		conf := config.Conf()
		// check again in 10 minutes
		if !conf.WgRegistered {
			zap.L().Debug(fmt.Sprintf("Next StarTram renewal check in 10 minutes"))
			time.Sleep(10 * time.Minute)
			continue
		}
		retrieve, err := startram.Retrieve()
		if err != nil {
			zap.L().Error(fmt.Sprintf("Failed to retrieve StarTram information: %v", err))
			zap.L().Debug(fmt.Sprintf("Next StarTram renewal check in 60 minutes"))
			// check again in 60 minutes
			time.Sleep(60 * time.Minute)
			continue
		}
		if retrieve.Ongoing == 1 {
			// check again in 12 hours
			zap.L().Debug(fmt.Sprintf("Next StarTram renewal check in 12 hours"))
			setReminder("one", false)
			setReminder("three", false)
			setReminder("seven", false)
			time.Sleep(12 * time.Hour)
			continue
		}

		// Layout to parse the given date (must match the format of dateStr)
		layout := "2006-01-02"

		// Parse the date string into a time.Time object
		expiryDate, err := time.Parse(layout, retrieve.Lease)
		if err != nil {
			zap.L().Error(fmt.Sprintf("Failed to parse expiry date %v: %v", retrieve.Lease, err))
			// check again in 12 hours
			zap.L().Debug(fmt.Sprintf("Next StarTram renewal check in 12 hours"))
			time.Sleep(12 * time.Hour)
			continue
		}

		// Get the current time
		currentTime := time.Now()

		// Calculate the difference in days
		diff := expiryDate.Sub(currentTime).Hours() / 24

		// Round down the number of days
		daysUntil := int(diff)

		// the send function
		send := func() {
			zap.L().Warn(fmt.Sprintf("Send renew notification to hark for test %v", daysUntil))
			sendStartramHarkNotification(daysUntil, conf.Piers)
		}

		rem := conf.StartramSetReminder
		if !rem.One && daysUntil <= 1 {
			zap.L().Warn("Send renew notification to hark for less than 1 day")
			send()
			setReminder("one", true)
		} else if !rem.Three && daysUntil <= 3 {
			send()
			setReminder("three", true)
		} else if !rem.Seven && daysUntil <= 7 {
			send()
			setReminder("seven", true)
		}
		// check again in 12 hours
		zap.L().Debug(fmt.Sprintf("Next StarTram renewal check in 12 hours"))
		time.Sleep(12 * time.Hour)
		continue
	}
}

func setReminder(daysType string, reminded bool) {
	if err := config.UpdateConf(map[string]interface{}{
		"startramSetReminder": map[string]bool{
			daysType: reminded,
		},
	}); err != nil {
		zap.L().Error(fmt.Sprintf("Couldn't reset startram reminder: %v", err))
	}
}

func sendStartramHarkNotification(daysLeft int, piers []string) {
	noti := structs.HarkNotification{Type: "startram-reminder", StartramDaysLeft: daysLeft}
	// Send notification
	for _, patp := range piers {
		shipConf := config.UrbitConf(patp)
		if shipConf.StartramReminder == true {
			if err := click.SendNotification(patp, noti); err != nil {
				zap.L().Error(fmt.Sprintf("Failed to send dev startram reminder to %s: %v", patp, err))
			}
		}
	}
}
