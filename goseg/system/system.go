package system

// for retrieving hw info and managing host

import (
	"fmt"
	"goseg/defaults"
	"goseg/logger"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
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

// get used/avail disk in bytes
func GetDisk() (uint64, uint64) {
	d, _ := disk.Usage("/")
	return d.Total, d.Used
}

// get cpu temp (may not work on some devices)
func GetTemp() float64 {
	// Run the 'sensors' command
	cmd := exec.Command("sensors")
	// Capture stdout
	out, err := cmd.Output()
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Failed to get sensor data: %v", err))
		return 0
	}
	keyword := "Package id 0:"
	for _, ln := range strings.Split(string(out), "\n") {
		if strings.Contains(ln, keyword) {
			// Use regex to find the first temperature
			re := regexp.MustCompile(`\+([0-9]+\.[0-9]+)`)
			match := re.FindStringSubmatch(ln)

			// Convert to float
			if len(match) > 1 {
				temp, err := strconv.ParseFloat(match[1], 64)
				if err == nil {
					return temp
				} else {
					logger.Logger.Error(fmt.Sprintf("Unable to parse float for CPU temperature: %v", err))
					return 0
				}
			} else {
				return 0
			}
		}
	}
	return 0
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
