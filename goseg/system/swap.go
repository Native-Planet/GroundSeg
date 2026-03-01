package system

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/disk"
)

var (
	cap = 32 // arbitrary max swap (gb)

	execCommandForSwap     = exec.Command
	osStatForSwap          = os.Stat
	osRemoveForSwap        = os.Remove
	startSwapForConfigure  = startSwap
	stopSwapForConfigure   = stopSwap
	makeSwapForConfigure   = makeSwap
	activeSwapForConfigure = ActiveSwap
	diskUsageForSwap       = disk.Usage
)

func ConfigureSwap(file string, val int) error {
	if val < 0 {
		return fmt.Errorf("Invalid value: %v", val)
	}
	if val == 0 {
		if err := stopSwapForConfigure(file); err != nil {
			return fmt.Errorf("Couldn't remove swap: %v", err)
		}
		return nil
	}
	if _, err := osStatForSwap(file); os.IsNotExist(err) {
		if err := makeSwapForConfigure(file, val); err != nil {
			return fmt.Errorf("Couldn't make swapfile: %v", err)
		}
	}
	if err := startSwapForConfigure(file); err != nil {
		return fmt.Errorf("Couldn't enable swap: %v", err)
	}
	swapSize := activeSwapForConfigure(file)
	if swapSize != val {
		if err := stopSwapForConfigure(file); err != nil {
			return fmt.Errorf("Couldn't remove swap: %v", err)
		}
		if err := osRemoveForSwap(file); err != nil {
			return fmt.Errorf("Couldn't remove old swap: %v", err)
		}
		if err := makeSwapForConfigure(file, val); err != nil {
			return fmt.Errorf("Couldn't make swap: %v", err)
		}
		if err := startSwapForConfigure(file); err != nil {
			return fmt.Errorf("Couldn't start swap: %v", err)
		}
	}
	return nil
}

func startSwap(loc string) error {
	cmd := execCommandForSwap("swapon", "--show")
	output, _ := cmd.Output()
	if strings.Contains(string(output), loc) {
		return nil
	}
	cmd = execCommandForSwap("swapon", loc)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("Failed to run swapon: %v", err)
	}
	return nil
}

func stopSwap(loc string) error {
	if err := execCommandForSwap("swapoff", loc).Run(); err != nil {
		return fmt.Errorf("Failed to run swapoff: %v\n", err)
	}
	return nil
}

func makeSwap(loc string, val int) error {
	if err := execCommandForSwap("fallocate", "-l", fmt.Sprintf("%dG", val), loc).Run(); err != nil {
		return fmt.Errorf("Failed to allocate space: %v\n", err)
	}
	if err := execCommandForSwap("chmod", "600", loc).Run(); err != nil {
		return fmt.Errorf("Failed to set permissions: %v\n", err)
	}
	if err := execCommandForSwap("mkswap", loc).Run(); err != nil {
		return fmt.Errorf("Failed to make swap: %v\n", err)
	}
	return nil
}

func ActiveSwap(loc string) int {
	cmd := execCommandForSwap("swapon", "--show")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, loc) {
			for _, item := range strings.Fields(line) {
				if strings.Contains(item, "M") || strings.Contains(item, "G") {
					num, _ := strconv.Atoi(strings.TrimRight(item, "MG"))
					if strings.Contains(item, "M") {
						return num / 1024
					}
					return num
				}
			}
		}
	}
	return 0
}

func MaxSwap(loc string, val int) int {
	usage, err := diskUsageForSwap(loc)
	if err != nil {
		return 0
	}
	freeSpaceGB := int(usage.Free / (1024 * 1024 * 1024))
	if freeSpaceGB > cap {
		return cap
	}
	return freeSpaceGB
}
