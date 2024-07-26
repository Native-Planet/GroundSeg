package routines

import (
	"fmt"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/handler"
	"time"

	"go.uber.org/zap"
)

func ChopAtLimit() {
	const GB = 1 << 30 // 1024 * 1024 * 1024
	for {
		conf := config.Conf()
		for _, patp := range conf.Piers {
			urbConf := config.UrbitConf(patp)
			if urbConf.SizeLimit == 0 {
				continue
			}
			currentSize := int64(docker.GetContainerStats(patp).DiskUsage / GB)
			zap.L().Info(fmt.Sprintf("Auto chop: Checking if %s requires a chop. Limit: %v GB, Current Size (rounded) %v GB", patp, urbConf.SizeLimit, currentSize))
			if int64(urbConf.SizeLimit) <= currentSize {
				zap.L().Info(fmt.Sprintf("Auto chop: Attempting to chop %s", patp))
				go handler.ChopPier(patp, urbConf)
			}
		}

		// check every 30 minutes
		time.Sleep(30 * time.Minute)
	}
}
