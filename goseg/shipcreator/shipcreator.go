package shipcreator

import (
	"fmt"
	"groundseg/config"
	"groundseg/defaults"
	"groundseg/logger"
	"path/filepath"
	"sort"
)

func CreateUrbitConfig(patp, customDrive string) error {
	// get unused http and ames ports
	httpPort, amesPort := getOpenUrbitPorts()
	// get default urbit config
	conf := defaults.UrbitConfig
	// replace values
	conf.PierName = patp
	conf.HTTPPort = httpPort
	conf.AmesPort = amesPort

	// custom dir specified
	if customDrive != "" {
		conf.CustomPierLocation = filepath.Join(customDrive, patp)
	}
	// get urbit config map
	urbConf := config.UrbitConfAll()
	// add to map
	urbConf[patp] = conf
	// persist config
	err := config.UpdateUrbitConfig(urbConf)
	return err
}

func getOpenUrbitPorts() (int, int) {
	// default ports
	// 8080 and 34343 is reserved
	httpPort := 8081
	amesPort := 34344

	// get piers
	conf := config.Conf()
	piers := conf.Piers

	// get used ports
	var amesAll []int
	var httpAll []int
	for _, pier := range piers {
		uConf := config.UrbitConf(pier)
		httpAll = append(httpAll, uConf.HTTPPort)
		amesAll = append(amesAll, uConf.AmesPort)
	}

	// sort them in ascending order
	sort.Ints(amesAll)
	sort.Ints(httpAll)

	findSmallestMissing := func(nums []int, defaultVal int) int {
		// Check if the slice is empty and return the default value if so.
		if len(nums) == 0 {
			return defaultVal
		}
		// Assuming nums is already sorted.
		for i := 0; i < len(nums)-1; i++ {
			// Check if the next number is not the consecutive number.
			if nums[i]+1 != nums[i+1] {
				return nums[i] + 1
			}
		}
		// If all numbers are consecutive, return the next number after the last element.
		return nums[len(nums)-1] + 1
	}
	httpPort = findSmallestMissing(httpAll, httpPort)
	amesPort = findSmallestMissing(amesAll, amesPort)
	logger.Logger.Info(fmt.Sprintf("Open Urbit Ports:  http: %v , ames: %v", httpPort, amesPort))
	return httpPort, amesPort
}

func contains(sortedSlice []int, val int) bool {
	index := sort.SearchInts(sortedSlice, val)
	return index < len(sortedSlice) && sortedSlice[index] == val
}

func AppendSysConfigPier(patp string) error {
	conf := config.Conf()
	piers := conf.Piers
	// Check if value already exists in slice
	exists := false
	for _, v := range piers {
		if v == patp {
			exists = true
			break
		}
	}
	// Append only if it doesn't exist yet
	if !exists {
		piers = append(piers, patp)
	}
	err := config.UpdateConf(map[string]interface{}{
		"piers": piers,
	})
	if err != nil {
		return err
	}
	return nil
}

// Remove all instances of patp from system config Piers
func RemoveSysConfigPier(patp string) error {
	conf := config.Conf()
	piers := conf.Piers
	var updated []string
	for _, memShip := range piers {
		if memShip != patp {
			updated = append(updated, memShip)
		}
	}
	err := config.UpdateConf(map[string]interface{}{
		"piers": updated,
	})
	return err
}
