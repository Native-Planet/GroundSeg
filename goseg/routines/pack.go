package routines

import (
	"fmt"
	"groundseg/broadcast"
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/structs"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

func PackScheduleLoop() {
	// check once at start
	if err := queuePack(); err != nil {
		zap.L().Error(fmt.Sprintf("Failed to make initial pack queue: %v", err))
	}
	ticker := time.NewTicker(1 * time.Minute)
	//ticker := time.NewTicker(15 * time.Second)
	for {
		select {
		case <-broadcast.SchedulePackBus:
			if err := queuePack(); err != nil {
				zap.L().Error(fmt.Sprintf("Failed to make pack queue with channel: %v", err))
			}
		case <-ticker.C:
			if err := queuePack(); err != nil {
				zap.L().Error(fmt.Sprintf("Failed to make pack queue with ticker: %v", err))
			}
		}
	}
}

func queuePack() error {
	var err error
	zap.L().Debug("Updating pack schedule")
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
			unixTime, err = strconv.ParseInt(shipConf.MeldLast, 10, 64)
			if err != nil {
				// If conversion fails, set to 0
				unixTime = 0
			}
		}
		// Convert int64 to time.Time
		meldNext := time.Unix(unixTime, 0)
		// Check Pack type
		switch shipConf.MeldScheduleType {
		case "month":
			meldNext, err = setMonthSchedule(meldNext, shipConf.MeldFrequency, shipConf.MeldDate, shipConf.MeldTime)
			if err != nil {
				zap.L().Error(fmt.Sprintf("Pack scheduling for %s failed: %v", patp, err))
				continue
			}
		case "week":
			meldNext, err = setWeekSchedule(meldNext, shipConf.MeldFrequency, shipConf.MeldDay, shipConf.MeldTime)
			if err != nil {
				zap.L().Error(fmt.Sprintf("Pack scheduling for %s failed: %v", patp, err))
				continue
			}
		case "day":
			meldNext, err = setDaySchedule(meldNext, shipConf.MeldFrequency, shipConf.MeldTime)
			if err != nil {
				zap.L().Error(fmt.Sprintf("Pack scheduling for %s failed: %v", patp, err))
				continue
			}
		default:
			zap.L().Warn(fmt.Sprintf("Pack schedule type for %s is not set. Defaulting to week", patp))
			meldNext, err = setWeekSchedule(meldNext, shipConf.MeldFrequency, shipConf.MeldDay, shipConf.MeldTime)
			if err != nil {
				zap.L().Error(fmt.Sprintf("Pack scheduling for %s failed: %v", patp, err))
				continue
			}
		}
		if err := broadcast.UpdateScheduledPack(patp, meldNext); err != nil {
			zap.L().Error(fmt.Sprintf("Failed to update pack schedule struct for %s: %v", patp, err))
		}

		now := time.Now()
		// if less than 1 * time.Minute left, create routine with timer
		zap.L().Debug(fmt.Sprintf("Next pack for %s on %v", patp, meldNext))
		oneMinuteLater := now.Add(1 * time.Minute)
		if oneMinuteLater.After(meldNext) || oneMinuteLater.Equal(meldNext) {
			go setScheduledPackTimer(patp, meldNext.Sub(now))
		}
	}
	return nil
}

func setMonthSchedule(meldLast time.Time, freq, date int, meldTime string) (time.Time, error) {
	// convert time to int
	hour, minute, err := convertMeldTime(meldTime)
	if err != nil {
		return meldLast, err
	}
	meldNext := time.Date(meldLast.Year(), meldLast.Month(), date, hour, minute, 0, 0, meldLast.Location())
	if meldNext.Before(meldLast) {
		meldNext = meldNext.AddDate(0, freq, 0)
	}
	return meldNext, nil
}

func setDaySchedule(meldLast time.Time, freq int, meldTime string) (time.Time, error) {
	// convert time to int
	hour, minute, err := convertMeldTime(meldTime)
	if err != nil {
		return meldLast, err
	}
	meldNext := meldLast.AddDate(0, 0, freq)
	meldNext = time.Date(meldNext.Year(), meldNext.Month(), meldNext.Day(), hour, minute, 0, 0, meldLast.Location())
	return meldNext, nil
}

func setWeekSchedule(meldLast time.Time, freq int, dayStr, meldTime string) (time.Time, error) {
	// Map string weekday to time.Weekday
	weekdayMap := map[string]time.Weekday{
		"monday":    time.Monday,
		"tuesday":   time.Tuesday,
		"wednesday": time.Wednesday,
		"thursday":  time.Thursday,
		"friday":    time.Friday,
		"saturday":  time.Saturday,
		"sunday":    time.Sunday,
	}
	day, ok := weekdayMap[dayStr]
	if !ok {
		return meldLast, fmt.Errorf("Invalid weekday: %s", day)
	}
	// Calculate days to the next specific weekday
	daysUntilNext := (int(day) - int(meldLast.Weekday()) + 7) % 7
	// Add freq weeks to the days
	daysUntilNext += (freq - 1) * 7
	// Get next specific weekday
	nextWeekday := meldLast.AddDate(0, 0, daysUntilNext)
	// Reset time to midnight
	nextWeekday = time.Date(nextWeekday.Year(), nextWeekday.Month(), nextWeekday.Day(), 0, 0, 0, 0, nextWeekday.Location())
	// convert time to int
	hour, minute, err := convertMeldTime(meldTime)
	if err != nil {
		return meldLast, err
	}
	meldNext := time.Date(nextWeekday.Year(), nextWeekday.Month(), nextWeekday.Day(), hour, minute, 0, 0, nextWeekday.Location())
	return meldNext, nil
}

func convertMeldTime(meldTime string) (int, int, error) {
	hour, err := strconv.Atoi(meldTime[0:2])
	if err != nil {
		return 0, 0, fmt.Errorf("Invalid hour: %v", meldTime)
	}
	// convert minute to int
	minute, err := strconv.Atoi(meldTime[2:4])
	if err != nil {
		return 0, 0, fmt.Errorf("Invalid minute: %v", meldTime)
	}
	return hour, minute, nil
}

func setScheduledPackTimer(patp string, delay time.Duration) {
	shipConf := config.UrbitConf(patp)
	if delay > 0 {
		zap.L().Info(fmt.Sprintf("Starting scheduled pack for %s in %v", patp, delay))
		time.Sleep(delay)
	} else {
		zap.L().Info(fmt.Sprintf("Starting scheduled pack for %s", patp))
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
		packError(fmt.Errorf("Failed to get ship status for %s: %v", patp, err))
		return
	}
	status, exists := statuses[patp]
	if !exists {
		packError(fmt.Errorf("Failed to get ship status for %s: status doesn't exist!", patp))
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
