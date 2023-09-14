package system

import (
	"fmt"
	"regexp"
	"goseg/logger"
	"goseg/structs"
	"os/exec"
)

var (
	SystemUpdates structs.SystemUpdates
)

func hasUpdates() (structs.SystemUpdates, error) {
	var updates structs.SystemUpdates
	cmd := exec.Command("apt", "update")
	err := cmd.Run()
	if err != nil {
		return updates, err
	}
	cmd = exec.Command("apt", "upgrade", "-s")
	out, err := cmd.Output()
	if err != nil {
		return updates, err
	}
	pattern := regexp.MustCompile(`(\d+) upgraded, (\d+) newly installed, (\d+) to remove and (\d+) not upgraded.`)
	matches := pattern.FindStringSubmatch(string(out))
	if matches == nil {
		return updates, fmt.Errorf("Pattern not found in apt upgrade -s output")
	}
	fmt.Sscanf(matches[1], "%d", &updates.Linux.Upgrade)
	fmt.Sscanf(matches[2], "%d", &updates.Linux.New)
	fmt.Sscanf(matches[3], "%d", &updates.Linux.Remove)
	fmt.Sscanf(matches[4], "%d", &updates.Linux.Ignore)
	return updates, nil
}

func RunUpgrade() error {
	cmd := exec.Command("apt", "upgrade", "-y")
	err := cmd.Run()
	UpdateCheck()
	return err
}

func UpdateCheck() {
	if updates, err := hasUpdates(); err != nil {
		logger.Logger.Error(fmt.Sprintf("Unable to check updates: %v",err))
	} else {
		SystemUpdates = updates
	}
}