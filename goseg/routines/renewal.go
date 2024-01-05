package routines

import (
	"fmt"
	"groundseg/config"
	"groundseg/logger"
	"groundseg/startram"
	"time"
)

func StartramRenewalReminder() {
	for {
		logger.Logger.Debug(fmt.Sprintf("Checking StarTram renewal status.."))
		conf := config.Conf()
		// check again in 10 minutes
		if !conf.WgRegistered {
			logger.Logger.Debug(fmt.Sprintf("Next StarTram renewal check in 10 minutes"))
			time.Sleep(10 * time.Minute)
			continue
		}
		retrieve, err := startram.Retrieve()
		if err != nil {
			logger.Logger.Error("Failed to retrieve StarTram information: %v", err)
			// check again in 60 minutes
			logger.Logger.Debug(fmt.Sprintf("Next StarTram renewal check in 60 minutes"))
			time.Sleep(60 * time.Minute)
			continue
		}
		if retrieve.Ongoing != 0 {
			// check again in 12 hours
			logger.Logger.Debug(fmt.Sprintf("Next StarTram renewal check in 12 hours"))
			time.Sleep(12 * time.Hour)
			continue
		}

		// Layout to parse the given date (must match the format of dateStr)
		layout := "2006-01-02"

		// Parse the date string into a time.Time object
		expiryDate, err := time.Parse(layout, retrieve.Lease)
		if err != nil {
			logger.Logger.Error("Failed to parse expiry date %v: %v", retrieve.Lease, err)
			// check again in 12 hours
			logger.Logger.Debug(fmt.Sprintf("Next StarTram renewal check in 12 hours"))
			time.Sleep(12 * time.Hour)
			continue
		}

		// Get the current time
		currentTime := time.Now()

		// Calculate the difference in days
		diff := expiryDate.Sub(currentTime).Hours() / 24

		// Round down the number of days
		daysUntil := int(diff)
		if daysUntil <= 1 {
			logger.Logger.Warn("Send renew notification to hark for less than 1 day")
		} else if daysUntil <= 3 {
			logger.Logger.Warn("Send renew notification to hark for less than 3 days")
		} else if daysUntil <= 7 {
			logger.Logger.Warn("Send renew notification to hark for less than 7 days")
		} else if daysUntil <= 1000 {
			logger.Logger.Warn(fmt.Sprintf("Send renew notification to hark for test %v", daysUntil))
		}
		// check again in 12 hours
		logger.Logger.Debug(fmt.Sprintf("Next StarTram renewal check in 12 hours"))
		time.Sleep(12 * time.Hour)
		continue
	}
}
