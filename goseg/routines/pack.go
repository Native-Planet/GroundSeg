package routines

import (
	"fmt"
	"groundseg/broadcast"
	"groundseg/config"
	"groundseg/handler"
	"groundseg/structs"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"
)

var scheduledPackState = struct {
	sync.Mutex
	active map[string]time.Time
}{
	active: make(map[string]time.Time),
}

const scheduledPackGrace = time.Minute

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
		scheduleType := shipConf.MeldScheduleType
		if scheduleType != "month" && scheduleType != "week" && scheduleType != "day" {
			scheduleType = "week"
		}
		scheduledAt := meldNext
		meldNext, skipped, err := skipMissedPackCycles(meldNext, time.Now(), scheduleType, shipConf.MeldFrequency)
		if err != nil {
			zap.L().Error(fmt.Sprintf("Pack scheduling for %s failed: %v", patp, err))
			continue
		}
		if skipped > 0 {
			zap.L().Debug(fmt.Sprintf(
				"Skipping %d missed pack cycle(s) for %s; scheduled time %v is more than %v in the past, next pack is %v",
				skipped, patp, scheduledAt, scheduledPackGrace, meldNext,
			))
		}
		if err := broadcast.UpdateScheduledPack(patp, meldNext); err != nil {
			zap.L().Error(fmt.Sprintf("Failed to update pack schedule struct for %s: %v", patp, err))
		}

		now := time.Now()
		// if less than 1 * time.Minute left, create routine with timer
		zap.L().Debug(fmt.Sprintf("Next pack for %s on %v", patp, meldNext))
		oneMinuteLater := now.Add(1 * time.Minute)
		if oneMinuteLater.After(meldNext) || oneMinuteLater.Equal(meldNext) {
			if reserveScheduledPack(patp, meldNext) {
				go setScheduledPackTimer(patp, meldNext.Sub(now))
			}
		}
	}
	return nil
}

func reserveScheduledPack(patp string, scheduledAt time.Time) bool {
	scheduledPackState.Lock()
	defer scheduledPackState.Unlock()
	if _, exists := scheduledPackState.active[patp]; exists {
		return false
	}
	scheduledPackState.active[patp] = scheduledAt
	return true
}

func releaseScheduledPack(patp string) {
	scheduledPackState.Lock()
	delete(scheduledPackState.active, patp)
	scheduledPackState.Unlock()
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
	if freq < 1 {
		return meldLast, fmt.Errorf("Invalid week frequency: %d", freq)
	}
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
	// When the last attempt was on the scheduled weekday at or after the
	// scheduled time, begin with the following week's occurrence.
	if !meldNext.After(meldLast) {
		meldNext = meldNext.AddDate(0, 0, 7)
	}
	// meldNext is the first occurrence after meldLast. Advance to the requested
	// interval from there.
	meldNext = meldNext.AddDate(0, 0, (freq-1)*7)
	return meldNext, nil
}

func skipMissedPackCycles(meldNext, now time.Time, scheduleType string, frequency int) (time.Time, int, error) {
	if frequency < 1 {
		return meldNext, 0, fmt.Errorf("Invalid pack frequency: %d", frequency)
	}

	skipped := 0
	overdueBefore := now.Add(-scheduledPackGrace)
	for meldNext.Before(overdueBefore) {
		switch scheduleType {
		case "month":
			meldNext = meldNext.AddDate(0, frequency, 0)
		case "week":
			meldNext = meldNext.AddDate(0, 0, frequency*7)
		case "day":
			meldNext = meldNext.AddDate(0, 0, frequency)
		default:
			return meldNext, skipped, fmt.Errorf("Invalid pack schedule type: %s", scheduleType)
		}
		skipped++
	}
	return meldNext, skipped, nil
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
	defer releaseScheduledPack(patp)
	shipConf := config.UrbitConf(patp)
	if delay > 0 {
		zap.L().Info(fmt.Sprintf("Starting scheduled pack for %s in %v", patp, delay))
		time.Sleep(delay)
	} else {
		zap.L().Info(fmt.Sprintf("Starting scheduled pack for %s", patp))
	}
	if err := config.LoadUrbitConfig(patp); err != nil {
		zap.L().Error(fmt.Sprintf("Scheduled pack failed to load %s config: %v", patp, err))
		return
	}
	shipConf = config.UrbitConf(patp)
	if !shipConf.MeldSchedule {
		zap.L().Info(fmt.Sprintf("Scheduled pack for %s was cancelled before it started", patp))
		return
	}
	// Persist the attempt before starting maintenance so this cycle cannot run
	// again if maintenance fails before the normal completion update.
	if err := config.UpdateUrbitConfigForPier(patp, func(shipConf *structs.UrbitDocker) {
		shipConf.MeldLast = strconv.FormatInt(time.Now().Unix(), 10)
	}); err != nil {
		zap.L().Error(fmt.Sprintf("Scheduled pack failed to record attempt for %s: %v", patp, err))
		return
	}
	shipConf = config.UrbitConf(patp)
	if err := handler.ScheduledPackPier(patp, shipConf); err != nil {
		zap.L().Error(fmt.Sprintf("Scheduled pack failed for %s: %v", patp, err))
	}
}
