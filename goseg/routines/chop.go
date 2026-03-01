package routines

import (
	"fmt"
	"groundseg/chopsvc"
	"groundseg/config"
	"groundseg/docker"
	"time"

	"go.uber.org/zap"
)

const bytesPerGiB = 1 << 30

var (
	confForChop              = config.Conf
	urbitConfForChop         = config.UrbitConf
	getContainerStatsForChop = docker.GetContainerStats
	chopPierForChop          = chopsvc.ChopPier
	sleepForChop             = time.Sleep
)

func runChopAtLimitPass() {
	conf := confForChop()
	for _, patp := range conf.Piers {
		urbConf := urbitConfForChop(patp)
		if urbConf.SizeLimit == 0 {
			continue
		}
		currentSize := int64(getContainerStatsForChop(patp).DiskUsage / bytesPerGiB)
		zap.L().Info(fmt.Sprintf("Auto chop: Checking if %s requires a chop. Limit: %v GB, Current Size (rounded) %v GB", patp, urbConf.SizeLimit, currentSize))
		if int64(urbConf.SizeLimit) <= currentSize {
			zap.L().Info(fmt.Sprintf("Auto chop: Attempting to chop %s", patp))
			go chopPierForChop(patp, urbConf)
		}
	}
}

func ChopAtLimit() {
	for {
		runChopAtLimitPass()
		// check every 30 minutes
		sleepForChop(30 * time.Minute)
	}
}
