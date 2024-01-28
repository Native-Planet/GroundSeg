package routines

/*

import (
	"fmt"
	"groundseg/logger"
	"groundseg/structs"
	"groundseg/system"
	"time"
)

func GetDriveStatus() {
	for {
		var drives []string
		var parts []string

		// lsblk
		blockDevices, err := system.ListHardDisks()
		if err != nil {
			logger.Logger.Debug(fmt.Sprintf("Failed to retrieve block mounts: %v", err))
			continue
		}
		for _, dev := range blockDevices.BlockDevices {
			// empty drive
			if len(dev.Children) < 1 && !isDevMounted(dev)
				// is mounted, do nothing
				// not mounted, put disk on


			} else {
				logger.Logger.Debug(fmt.Sprintf("%+v has children!", dev.Name))
		// yes: check if partitions are mounted
				for _, part := range dev.Children {
					isMounted := isDevMounted(part)
				}
			}
		}
		time.Sleep(15 * time.Second)
		//time.Sleep(time.Minute)
	}
}

*/
