package system

// for retrieving hw info and managing host

import (
	"bufio"
	"fmt"
	"goseg/defaults"
	"goseg/logger"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

var (
	C2CStoredSSIDs []string
)

// get memory total/used in bytes
func GetMemory() (uint64, uint64) {
	v, _ := mem.VirtualMemory()
	return v.Total, v.Used
}

// get cpu usage as %
func GetCPU() int {
	percent, _ := cpu.Percent(time.Second, false)
	return int(percent[0])
}

// get used/avail disk in bytes with labels
func GetDisk() (map[string][2]uint64, error) {
	diskUsageMap := make(map[string][2]uint64)
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return diskUsageMap, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 {
			device := fields[0]
			mountPoint := fields[1]
			if !strings.HasPrefix(device, "/dev/") {
				continue
			}
			var stat syscall.Statfs_t
			if err := syscall.Statfs(mountPoint, &stat); err != nil {
				return diskUsageMap, err
			}
			all := stat.Blocks * uint64(stat.Bsize)
			free := stat.Bfree * uint64(stat.Bsize)
			used := all - free
			diskUsageMap[device] = [2]uint64{used, all}
		}
	}
	if err := scanner.Err(); err != nil {
		return diskUsageMap, err
	}
	return diskUsageMap, nil
}

// get cpu temp (may not work on non-intel devices)
func GetTemp() float64 {
	basePath := "/sys/class/hwmon/"
	hwmons, err := ioutil.ReadDir(basePath)
	if err != nil {
		fmt.Printf("Error reading the hwmon directory: %v\n", err)
		return 0
	}
	var totalTemp float64
	var tempCount int
	for _, hwmon := range hwmons {
		path := filepath.Join(basePath, hwmon.Name())
		devicePath := filepath.Join(path, "name")
		device, err := ioutil.ReadFile(devicePath)
		if err != nil {
			continue
		}
		if strings.Contains(strings.ToLower(string(device)), "coretemp") {
			tempInputs, _ := filepath.Glob(filepath.Join(path, "temp*_input"))
			for _, tempInput := range tempInputs {
				temp, err := ioutil.ReadFile(tempInput)
				if err != nil {
					fmt.Printf("Error reading temperature from %s: %v\n", tempInput, err)
					continue
				}
				tempValue, err := strconv.Atoi(strings.TrimSpace(string(temp)))
				if err != nil {
					fmt.Printf("Error converting temperature: %s\n", temp)
					continue
				}
				totalTemp += float64(tempValue)
				tempCount++
			}
		}
	}
	if tempCount > 0 {
		return totalTemp / float64(tempCount) / 1000.0
	} else {
		return 0
	}
}

func IsNPBox(basePath string) bool {
	filePath := filepath.Join(basePath, "nativeplanet")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}
	logger.Logger.Info("Thank you for supporting Native Planet!")
	return true
}

// set up auto-reinstall script
func FixerScript(basePath string) error {
	// check if it's one of our boxes
	if IsNPBox(basePath) {
		// Create fixer.sh
		fixer := filepath.Join(basePath, "fixer.sh")
		if _, err := os.Stat(fixer); os.IsNotExist(err) {
			logger.Logger.Info("Fixer script not detected, creating")
			err := ioutil.WriteFile(fixer, []byte(defaults.Fixer), 0755)
			if err != nil {
				return err
			}
		}
		//make it a cron
		if !cronExists(fixer) {
			logger.Logger.Info("Fixer cron not found, creating")
			cronJob := fmt.Sprintf("*/5 * * * * /bin/bash %s\n", fixer)
			err := addCron(cronJob)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func cronExists(fixerPath string) bool {
	out, err := exec.Command("crontab", "-l").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), fixerPath)
}

func addCron(job string) error {
	tmpfile, err := ioutil.TempFile("", "cron")
	if err != nil {
		return err
	}
	defer os.Remove(tmpfile.Name())
	out, _ := exec.Command("crontab", "-l").Output()
	tmpfile.WriteString(string(out))
	tmpfile.WriteString(job)
	tmpfile.Close()
	cmd := exec.Command("crontab", tmpfile.Name())
	return cmd.Run()
}
