package chop

import (
	"errors"
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
	ConfForChop           = config.Config
	UrbitConfForChop      = config.UrbitConf
	ContainerStatsForChop = orchestration.GetContainerStats
	ChopPierForChop       = chopsvc.ChopPier
	sleepForChop          = time.Sleep
)

func RunAtLimitPass() error {
	return RunAtLimitPassWithDependencies(
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
) error {
	conf := confForChop()
	var passErrs []error
	for _, patp := range conf.Connectivity.Piers {
		urbConf := urbitConfForChop(patp)
		if urbConf.SizeLimit == 0 {
			continue
		}
		currentSize := int64(getContainerStatsForChop(patp).DiskUsage / bytesPerGiB)
		zap.L().Info(fmt.Sprintf("Auto chop: Checking if %s requires a chop. Limit: %v GB, Current Size (rounded) %v GB", patp, urbConf.SizeLimit, currentSize))
		if int64(urbConf.SizeLimit) <= currentSize {
			zap.L().Info(fmt.Sprintf("Auto chop: Attempting to chop %s", patp))
			if err := chopPierForChop(patp); err != nil {
				passErrs = append(passErrs, fmt.Errorf("chop %s: %w", patp, err))
			}
		}
	}
	return errors.Join(passErrs...)
}

func StartLoop(sleep func(time.Duration)) {
	for {
		if err := RunAtLimitPass(); err != nil {
			zap.L().Error(fmt.Sprintf("auto chop pass failed: %v", err))
		}
		sleep(30 * time.Minute)
	}
}

func StartChopRoutines() error {
	go ChopAtLimit()
	return nil
}

func ChopAtLimit() {
	StartLoop(sleepForChop)
}
