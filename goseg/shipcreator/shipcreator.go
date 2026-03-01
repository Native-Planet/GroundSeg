package shipcreator

import (
	"fmt"
	"groundseg/config"
	"groundseg/defaults"
	"groundseg/structs"
	"path/filepath"
	"sort"

	"go.uber.org/zap"
)

type ShipConfigPort interface {
	ExistingPiers() []string
	ExistingShipPorts() ([]int, []int)
	SaveShipConfig(patp string, conf structs.UrbitDocker) error
	SavePiers(piers []string) error
}

type configShipPort struct{}

func (configShipPort) ExistingPiers() []string {
	return config.ShipSettingsSnapshot().Piers
}

func (configShipPort) ExistingShipPorts() ([]int, []int) {
	piers := config.ShipSettingsSnapshot().Piers
	httpPorts := make([]int, 0, len(piers))
	amesPorts := make([]int, 0, len(piers))
	for _, pier := range piers {
		uConf := config.UrbitConf(pier)
		httpPorts = append(httpPorts, uConf.HTTPPort)
		amesPorts = append(amesPorts, uConf.AmesPort)
	}
	return httpPorts, amesPorts
}

func (configShipPort) SaveShipConfig(patp string, conf structs.UrbitDocker) error {
	urbConfs := config.UrbitConfAll()
	urbConfs[patp] = conf
	return config.UpdateUrbitConfig(urbConfs)
}

func (configShipPort) SavePiers(piers []string) error {
	return config.UpdateConfTyped(config.WithPiers(piers))
}

type Service struct {
	store ShipConfigPort
}

func NewService(cfg ShipConfigPort) *Service {
	if cfg == nil {
		cfg = configShipPort{}
	}
	return &Service{store: cfg}
}

var defaultService = NewService(nil)

func CreateUrbitConfig(patp, customDrive string) error {
	return defaultService.CreateUrbitConfig(patp, customDrive)
}

func (s *Service) CreateUrbitConfig(patp, customDrive string) error {
	// get unused http and ames ports
	httpPort, amesPort := s.getOpenUrbitPorts()
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
	// persist config
	err := s.store.SaveShipConfig(patp, conf)
	return err
}

func (s *Service) getOpenUrbitPorts() (int, int) {
	// default ports
	// 8080 and 34343 is reserved
	httpPort := 8081
	amesPort := 34344

	// get used ports
	httpAll, amesAll := s.store.ExistingShipPorts()

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
	zap.L().Info(fmt.Sprintf("Open Urbit Ports:  http: %v , ames: %v", httpPort, amesPort))
	return httpPort, amesPort
}

func contains(sortedSlice []int, val int) bool {
	index := sort.SearchInts(sortedSlice, val)
	return index < len(sortedSlice) && sortedSlice[index] == val
}

func AppendSysConfigPier(patp string) error {
	return defaultService.AppendSysConfigPier(patp)
}

func (s *Service) AppendSysConfigPier(patp string) error {
	piers := s.store.ExistingPiers()
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
	err := s.store.SavePiers(piers)
	if err != nil {
		return err
	}
	return nil
}

// Remove all instances of patp from system config Piers
func RemoveSysConfigPier(patp string) error {
	return defaultService.RemoveSysConfigPier(patp)
}

func (s *Service) RemoveSysConfigPier(patp string) error {
	piers := s.store.ExistingPiers()
	var updated []string
	for _, memShip := range piers {
		if memShip != patp {
			updated = append(updated, memShip)
		}
	}
	err := s.store.SavePiers(updated)
	return err
}
