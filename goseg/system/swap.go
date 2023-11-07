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
)

func ConfigureSwap(file string, val int) error {
	if val < 0 {
		return fmt.Errorf("Invalid value: %v", val)
	}
	if val == 0 {
		if err := stopSwap(file); err != nil {
			return fmt.Errorf("Couldn't remove swap: %v", err)
		}
		return nil
	}
	if _, err := os.Stat(file); os.IsNotExist(err) {
		if err := makeSwap(file, val); err != nil {
			return fmt.Errorf("Couldn't make swapfile: %v", err)
		}
	}
	if err := startSwap(file); err != nil {
		return fmt.Errorf("Couldn't enable swap: %v", err)
	}
	swapSize := ActiveSwap(file)
	if swapSize != val {
		if err := stopSwap(file); err != nil {
			return fmt.Errorf("Couldn't remove swap: %v", err)
		}
		if err := os.Remove(file); err != nil {
			return fmt.Errorf("Couldn't remove old swap: %v", err)
		}
		if err := makeSwap(file, val); err != nil {
			return fmt.Errorf("Couldn't make swap: %v", err)
		}
		if err := startSwap(file); err != nil {
			return fmt.Errorf("Couldn't start swap: %v", err)
		}
	}
	return nil
}

func startSwap(loc string) error {
	cmd := exec.Command("swapon", "--show")
	output, _ := cmd.Output()
	if strings.Contains(string(output), loc) {
		return nil
	}
	cmd = exec.Command("swapon", loc)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("Failed to run swapon: %v", err)
	}
	return nil
}

func stopSwap(loc string) error {
	if err := exec.Command("swapoff", loc).Run(); err != nil {
		return fmt.Errorf("Failed to run swapoff: %v\n", err)
	}
	return nil
}

func makeSwap(loc string, val int) error {
	if err := exec.Command("fallocate", "-l", fmt.Sprintf("%dG", val), loc).Run(); err != nil {
		return fmt.Errorf("Failed to allocate space: %v\n", err)
	}
	if err := exec.Command("chmod", "600", loc).Run(); err != nil {
		return fmt.Errorf("Failed to set permissions: %v\n", err)
	}
	if err := exec.Command("mkswap", loc).Run(); err != nil {
		return fmt.Errorf("Failed to make swap: %v\n", err)
	}
	return nil
}

func ActiveSwap(loc string) int {
	cmd := exec.Command("swapon", "--show")
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
	usage, err := disk.Usage(loc)
	if err != nil {
		return 0
	}
	freeSpaceGB := int(usage.Free / (1024 * 1024 * 1024))
	if freeSpaceGB > cap {
		return cap
	}
	return freeSpaceGB
}
