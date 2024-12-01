package routines

import (
	"fmt"
	"groundseg/config"
	"groundseg/roller"
	"groundseg/structs"
	"time"

	"go.uber.org/zap"
)

func CheckAzimuthLoop() {
	conf := config.Conf()
	config.AzimuthPoints = loadPointInfo(conf.Piers)
	var updateInterval int
	if conf.UpdateInterval < 60 {
		updateInterval = 60
	} else {
		updateInterval = conf.UpdateInterval
	}
	checkInterval := time.Duration(updateInterval) * time.Second
	ticker := time.NewTicker(checkInterval)
	for {
		select {
		case <-ticker.C:
			config.AzimuthPoints = loadPointInfo(conf.Piers)
		}
	}
}

// retrieve current point info from roller
func loadPointInfo(urbits []string) map[string]*structs.Point {
	points := make(map[string]*structs.Point)
	for _, ship := range urbits {
		point, err := roller.Client.GetPoint(config.Ctx, ship)
		if err != nil {
			zap.L().Error(fmt.Sprintf("Unable to get point for %s: %v", ship, err))
			continue
		}
		points[ship] = point
	}
	return points
}