package apt

import (
	"fmt"
	"os/exec"
	"regexp"

	"groundseg/structs"

	"go.uber.org/zap"
)

var (
	systemUpdates structs.SystemUpdates

	execCommandForAPT        = exec.Command
	hasUpdatesForAPT         = hasUpdates
	updateCheckForRunUpgrade = UpdateCheck
)

func SystemUpdatesSnapshot() structs.SystemUpdates {
	return systemUpdates
}

func hasUpdates() (structs.SystemUpdates, error) {
	var updates structs.SystemUpdates
	cmd := execCommandForAPT("apt", "update")
	err := cmd.Run()
	if err != nil {
		return updates, err
	}
	cmd = execCommandForAPT("apt", "upgrade", "-s")
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
	cmd := execCommandForAPT("apt", "upgrade", "-y")
	err := cmd.Run()
	updateCheckForRunUpgrade()
	return err
}

func UpdateCheck() {
	if updates, err := hasUpdatesForAPT(); err != nil {
		zap.L().Error(fmt.Sprintf("Unable to check updates: %v", err))
	} else {
		systemUpdates = updates
	}
}
