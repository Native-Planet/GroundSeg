package routines

import (
	"goseg/config"
	"goseg/system"
	"time"
)

func AptUpdateLoop() {
	system.UpdateCheck()
	conf := config.Conf()
	val := time.Duration(conf.LinuxUpdates.Value)

	var interval time.Duration
	if interv := conf.LinuxUpdates.Interval; interv == "week" {
		interval = 7 * (time.Hour * 24)
	} else if interv == "day" {
		interval = time.Hour * 24
	} else {
		interval = 30 * (time.Hour * 24)
	}
	checkInterval := val * interval
	ticker := time.NewTicker(checkInterval)
	for {
		select {
		case <-ticker.C:
			system.UpdateCheck()
		}
	}
}
