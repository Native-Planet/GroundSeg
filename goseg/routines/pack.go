package routines

import (
	"fmt"
	"goseg/click"
	"goseg/config"
	"goseg/docker"
	"goseg/logger"
	"goseg/structs"
	"strconv"
	"strings"
	"time"
)

func PackScheduleLoop() {
	// check once at start
	if err := queuePack(); err != nil {
		logger.Logger.Error(fmt.Sprintf("Failed to make initial pack queue: %v", err))
	}
	ticker := time.NewTicker(1 * time.Minute)
	for {
		select {
		case <-ticker.C:
			if err := queuePack(); err != nil {
				logger.Logger.Error(fmt.Sprintf("Failed to make initial pack queue: %v", err))
			}
		}
	}
}

func queuePack() error {
	logger.Logger.Debug("Updating pack schedule")
	conf := config.Conf()
	for _, patp := range conf.Piers {
		shipConf := config.UrbitConf(patp)
		// is scheduled
		if !shipConf.MeldSchedule {
			continue
		}
		// prep next pack
		var unixTime int64 = 0 // Default to 0

		// Check if string is empty or not a legitimate Unix time
		if shipConf.MeldLast != "" {
			var err error
			unixTime, err = strconv.ParseInt(shipConf.MeldLast, 10, 64)
			if err != nil {
				// If conversion fails, set to 0
				unixTime = 0
			}
		}
		// Convert int64 to time.Time
		meldNext := time.Unix(unixTime, 0)
		switch shipConf.MeldScheduleType {
		case "month":
			meldNext = meldNext.AddDate(0, shipConf.MeldFrequency, 0)
		case "week":
			meldNext = meldNext.Add(time.Hour * 24 * 7 * time.Duration(shipConf.MeldFrequency))
		case "day":
			meldNext = meldNext.Add(time.Hour * 24 * time.Duration(shipConf.MeldFrequency))
		default:
			logger.Logger.Warn(fmt.Sprintf("Pack schedule type for %s is not set. Defaulting to week", patp))
			meldNext = meldNext.Add(time.Hour * 24 * 7 * time.Duration(shipConf.MeldFrequency))
		}
		timeString := shipConf.MeldTime
		// Default to midnight
		hour, minute := 0, 0
		// Extract hour and minute from the string if it's valid
		if len(timeString) == 4 {
			var err1, err2 error
			hour, err1 = strconv.Atoi(timeString[0:2])
			minute, err2 = strconv.Atoi(timeString[2:4])
			if err1 != nil || err2 != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
				hour, minute = 0, 0
			}
		}
		// Create a new time object with the same date but new time
		meldNext = time.Date(
			meldNext.Year(),
			meldNext.Month(),
			meldNext.Day(),
			hour,
			minute,
			meldNext.Second(),
			meldNext.Nanosecond(),
			meldNext.Location(),
		)
		now := time.Now()
		// if less than 1 * time.Minute left, create routine with timer
		logger.Logger.Debug(fmt.Sprintf("Next pack for %s on %v", patp, meldNext))
		oneMinuteLater := now.Add(1 * time.Minute)
		if oneMinuteLater.After(meldNext) || oneMinuteLater.Equal(meldNext) {
			//go setScheduledPackTimer(patp, meldNext.Sub(now))
			logger.Logger.Debug("Scheduled pack temporarily turned off")
		}
	}
	return nil
}

func setScheduledPackTimer(patp string, delay time.Duration) {
	shipConf := config.UrbitConf(patp)
	if delay > 0 {
		logger.Logger.Info(fmt.Sprintf("Starting scheduled pack for %s in %v", patp, delay))
		time.Sleep(delay)
	} else {
		logger.Logger.Info(fmt.Sprintf("Starting scheduled pack for %s", patp))
	}
	// error handling
	packError := func(err error) {
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "pack", Event: "error"}
		return
	}
	// clear transition after end
	defer func() {
		time.Sleep(3 * time.Second)
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "pack", Event: ""}
	}()
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "pack", Event: "packing"}
	statuses, err := docker.GetShipStatus([]string{patp})
	if err != nil {
		packError(fmt.Errorf("Failed to get ship status for %p: %v", patp, err))
		return
	}
	status, exists := statuses[patp]
	if !exists {
		packError(fmt.Errorf("Failed to get ship status for %p: status doesn't exist!", patp))
		return
	}
	// running
	if strings.Contains(status, "Up") {
		// send |pack
		if err := click.SendPack(patp); err != nil {
			packError(fmt.Errorf("Failed to |pack to %s: %v", patp, err))
			return
		}
		// not running
	} else {
		// switch boot status to pack
		shipConf.BootStatus = "pack"
		update := make(map[string]structs.UrbitDocker)
		update[patp] = shipConf
		err := config.UpdateUrbitConfig(update)
		if err != nil {
			packError(fmt.Errorf("Failed to update %s urbit config to pack: %v", patp, err))
			return
		}
		_, err = docker.StartContainer(patp, "vere")
		if err != nil {
			packError(fmt.Errorf("Failed to urth pack %s: %v", patp, err))
			return
		}
	}
	// set last meld
	now := time.Now().Unix()
	shipConf.MeldLast = strconv.FormatInt(now, 10)
	update := make(map[string]structs.UrbitDocker)
	update[patp] = shipConf
	err = config.UpdateUrbitConfig(update)
	if err != nil {
		packError(fmt.Errorf("Failed to update %s urbit config with last meld time: %v", patp, err))
		return
	}
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "pack", Event: "success"}
	return
}
