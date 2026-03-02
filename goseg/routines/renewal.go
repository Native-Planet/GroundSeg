package routines

import (
	"context"
	"fmt"
	"groundseg/click"
	"groundseg/config"
	"groundseg/startram"
	"groundseg/structs"
	"time"

	"go.uber.org/zap"
)

var (
	withStartramReminderOneForRenewal   = config.WithStartramReminderOne
	withStartramReminderThreeForRenewal = config.WithStartramReminderThree
	withStartramReminderSevenForRenewal = config.WithStartramReminderSeven
	updateConfTypedForRenewal           = config.UpdateConfTyped
	urbitConfForRenewal                 = config.UrbitConf
	sendNotificationForRenewal          = click.SendNotification
)

func StartramRenewalReminder() {
	for {
		delay := startramRenewalDelay()
		time.Sleep(delay)
	}
}

func StartramRenewalReminderWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		delay := startramRenewalDelay()
		if err := waitForContext(ctx, delay); err != nil {
			return nil
		}
	}
}

func startramRenewalDelay() time.Duration {
	conf := config.Conf()
	zap.L().Debug(fmt.Sprintf("Checking StarTram renewal status.."))

	if !conf.WgRegistered {
		zap.L().Debug(fmt.Sprintf("Next StarTram renewal check in 10 minutes"))
		return 10 * time.Minute
	}

	retrieve, err := startram.Retrieve()
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to retrieve StarTram information: %v", err))
		zap.L().Debug(fmt.Sprintf("Next StarTram renewal check in 60 minutes"))
		return 60 * time.Minute
	}

	if retrieve.Ongoing == 1 {
		zap.L().Debug(fmt.Sprintf("Next StarTram renewal check in 12 hours"))
		setReminder("one", false)
		setReminder("three", false)
		setReminder("seven", false)
		return 12 * time.Hour
	}

	layout := "2006-01-02"
	expiryDate, err := time.Parse(layout, retrieve.Lease)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to parse expiry date %v: %v", retrieve.Lease, err))
		zap.L().Debug(fmt.Sprintf("Next StarTram renewal check in 12 hours"))
		return 12 * time.Hour
	}

	daysUntil := int(expiryDate.Sub(time.Now()).Hours() / 24)
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

	zap.L().Debug(fmt.Sprintf("Next StarTram renewal check in 12 hours"))
	return 12 * time.Hour
}

func waitForContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func setReminder(daysType string, reminded bool) {
	var option config.ConfUpdateOption
	switch daysType {
	case "one":
		option = withStartramReminderOneForRenewal(reminded)
	case "three":
		option = withStartramReminderThreeForRenewal(reminded)
	case "seven":
		option = withStartramReminderSevenForRenewal(reminded)
	default:
		zap.L().Warn(fmt.Sprintf("Unknown startram reminder type: %s", daysType))
		return
	}
	if err := updateConfTypedForRenewal(option); err != nil {
		zap.L().Error(fmt.Sprintf("Couldn't set startram reminder: %v", err))
	}
}

func sendStartramHarkNotification(daysLeft int, piers []string) {
	noti := structs.HarkNotification{Type: "startram-reminder", StartramDaysLeft: daysLeft}
	for _, patp := range piers {
		shipConf := urbitConfForRenewal(patp)
		if shipConf.StartramReminder == true {
			if err := sendNotificationForRenewal(patp, noti); err != nil {
				zap.L().Error(fmt.Sprintf("Failed to send dev startram reminder to %s: %v", patp, err))
			}
		}
	}
}
