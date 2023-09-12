package system

// for retrieving hw info and managing host

import (
	"fmt"
	"goseg/defaults"
	"goseg/logger"
	"io/ioutil"
	"strconv"
	"strings"
	"time"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/robfig/cron/v3"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
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

// get used/avail disk in bytes
func GetDisk() (uint64, uint64) {
	d, _ := disk.Usage("/")
	return d.Used, d.Free
}

// get cpu temp (may not work on some devices)
func GetTemp() float64 {
	data, err := ioutil.ReadFile("/sys/class/thermal/thermal_zone0/temp")
	if err != nil {
		// errmsg := fmt.Sprintf("Error reading temperature:", err) // ignore for vps testing
		// logger.Logger.Error(errmsg)
		return 0
	}
	tempStr := strings.TrimSpace(string(data))
	temp, err := strconv.Atoi(tempStr)
	if err != nil {
		errmsg := fmt.Sprintf("Error converting temperature to integer:", err)
		logger.Logger.Error(errmsg)
		return 0
	}
	return float64(temp) / 1000.0
}

// return 0 for no 1 for yes(?)
func HasSwap() int {
	data, err := ioutil.ReadFile("/proc/swaps")
	if err != nil {
		errmsg := fmt.Sprintf("Error reading swap status:", err)
		logger.Logger.Error(errmsg)
		return 0
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) > 1 {
		return 1
	} else {
		return 0
	}
}

func IsNPBox(basePath string) bool {
	filePath := filepath.Join(basePath, "nativeplanet")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}
	return true
}

// set up auto-reinstall script
func FixerScript(basePath string) error {
	// check if it's one of our boxes
	if IsNPBox(basePath) {
		// Create fixer.sh
		fixer := filepath.Join(basePath, "fixer.sh")
		if _, err := os.Stat(fixer); os.IsNotExist(err) {
			fmt.Println("Config: Update fixer script not detected. Creating!")
			err := ioutil.WriteFile(fixer, []byte(defaults.Fixer), 0755)
			if err != nil {
				return err
			}
		}
		//make it a cron
		if !cronExists(fixer) {
			fmt.Println("Config: Updater cron job not found. Creating!")
			c := cron.New()
			_, err := c.AddFunc("@every 5m", func() {
				exec.Command("/bin/sh", fixer).Run()
			})
			if err != nil {
				return err
			}
			c.Start()
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
