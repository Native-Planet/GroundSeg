package chop

import (
	"fmt"
	"groundseg/chopsvc"
	"groundseg/config"
	"groundseg/docker/orchestration"
	"groundseg/structs"
	"time"

	"go.uber.org/zap"
)

const BytesPerGiB = 1 << 30

var (
	ConfForChop           = config.Conf
	UrbitConfForChop      = config.UrbitConf
	ContainerStatsForChop = orchestration.GetContainerStats
	ChopPierForChop       = chopsvc.ChopPier
	sleepForChop          = time.Sleep
)

func RunAtLimitPass() {
	RunAtLimitPassWithDependencies(
		ConfForChop,
		UrbitConfForChop,
		ContainerStatsForChop,
		ChopPierForChop,
		BytesPerGiB,
	)
}

func RunAtLimitPassWithDependencies(
	confForChop func() structs.SysConfig,
	urbitConfForChop func(string) structs.UrbitDocker,
	getContainerStatsForChop func(string) structs.ContainerStats,
	chopPierForChop func(string) error,
	bytesPerGiB int64,
) {
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
			go chopPierForChop(patp)
		}
	}
}

func StartLoop(sleep func(time.Duration)) {
	for {
		RunAtLimitPass()
		sleep(30 * time.Minute)
	}
}

func StartChopRoutines() {
	go ChopAtLimit()
}

func ChopAtLimit() {
	StartLoop(sleepForChop)
}
