package system

// for retrieving hw info and managing host

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"groundseg/defaults"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"go.uber.org/zap"
)

var (
	C2CStoredSSIDs []string
)

// get memory total/used in bytes
func GetMemory() (uint64, uint64, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return 0, 0, fmt.Errorf("read virtual memory stats: %w", err)
	}
	return v.Total, v.Used, nil
}

// get cpu usage as %
func GetCPU() (int, error) {
	percent, err := cpu.Percent(time.Second, false)
	if err != nil {
		return 0, fmt.Errorf("read CPU usage: %w", err)
	}
	if len(percent) == 0 {
		return 0, fmt.Errorf("read CPU usage: empty CPU percentage response")
	}
	return int(percent[0]), nil
}

// get used/avail disk in bytes with labels
func GetDisk() (map[string][2]uint64, error) {
	diskUsageMap := make(map[string][2]uint64)
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return diskUsageMap, err
	}
	defer file.Close()
	getDiskLabel := func(device string) (string, string) {
		labelDir := "/dev/disk/by-label/"
		files, err := ioutil.ReadDir(labelDir)
		if err != nil {
			return "", ""
		}
		for _, f := range files {
			fullPath := filepath.Join(labelDir, f.Name())
			resolvedPath, err := os.Readlink(fullPath)
			if err != nil {
				zap.L().Warn(fmt.Sprintf("Unable to read disk label source for %s: %v", fullPath, err))
				continue
			}
			if strings.HasSuffix(resolvedPath, device) {
				label, err := url.QueryUnescape(f.Name())
				if err != nil {
					zap.L().Warn(fmt.Sprintf("Couldn't decode encoded disk label name %s: %v", f.Name(), err))
					return device, ""
				}
				label, err = octalToAscii(label)
				if err != nil {
					zap.L().Warn(fmt.Sprintf("Couldn't decode octal in disk label: %v", err))
					return device, ""
				}
				return device, label
			}
		}
		return device, ""
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 {
			device := fields[0]
			mountPoint, _ := octalToAscii(fields[1])
			if !strings.HasPrefix(device, "/dev/") || strings.HasPrefix(device, "/dev/loop") {
				continue
			}
			var stat syscall.Statfs_t
			if err := syscall.Statfs(mountPoint, &stat); err != nil {
				return diskUsageMap, fmt.Errorf("%s: %w", mountPoint, err)
			}
			all := stat.Blocks * uint64(stat.Bsize)
			free := stat.Bfree * uint64(stat.Bsize)
			used := all - free
			_, label := getDiskLabel(device)
			key := label
			if label == "" {
				key = device
			}
			diskUsageMap[key] = [2]uint64{used, all}
		}
	}
	if err := scanner.Err(); err != nil {
		return diskUsageMap, err
	}
	return diskUsageMap, nil
}

func octalToAscii(s string) (string, error) {
	re := regexp.MustCompile(`\\[0-7]{3}`)
	replaceFunc := func(match string) string {
		i, err := strconv.ParseInt(match[1:], 8, 64)
		if err != nil {
			return match
		}
		return string(rune(i))
	}
	return re.ReplaceAllStringFunc(s, replaceFunc), nil
}

// get cpu temp (may not work on non-intel devices)
func GetTemp() (float64, error) {
	basePath := "/sys/class/hwmon/"
	hwmons, err := ioutil.ReadDir(basePath)
	if err != nil {
		return 0, fmt.Errorf("Error reading the hwmon directory: %w", err)
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
					zap.L().Warn(fmt.Sprintf("Error reading temperature from %s: %v\n", tempInput, err))
					continue
				}
				tempValue, err := strconv.Atoi(strings.TrimSpace(string(temp)))
				if err != nil {
					zap.L().Warn(fmt.Sprintf("Error converting temperature: %s\n", temp))
					continue
				}
				totalTemp += float64(tempValue)
				tempCount++
			}
		}
	}
	if tempCount > 0 {
		return totalTemp / float64(tempCount) / 1000.0, nil
	}
	return 0, fmt.Errorf("no CPU temperature readings found")
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
		zap.L().Info("Thank you for supporting Native Planet!")
		fixer := filepath.Join(basePath, "fixer.sh")
		if _, err := os.Stat(fixer); os.IsNotExist(err) {
			zap.L().Info("Fixer script not detected, creating")
			err := ioutil.WriteFile(fixer, []byte(defaults.Fixer), 0755)
			if err != nil {
				return err
			}
		}
		//make it a cron
		if !cronExists(fixer) {
			zap.L().Info("Fixer cron not found, creating")
			cronJob := fmt.Sprintf("*/5 * * * * /bin/bash %s\n", fixer)
			err := addCron(cronJob)
			if err != nil {
				return err
			}
		} else {
			zap.L().Info("Fixer cron found. Doing nothing")
		}
	}
	return nil
}

func cronExists(fixerPath string) bool {
	out, err := exec.Command("crontab", "-l").Output()
	if err != nil {
		return false
	}
	outStr := string(out)
	return strings.Contains(outStr, fixerPath) && strings.Contains(outStr, "/bin/bash")
}

func addCron(job string) error {
	tmpfile, err := ioutil.TempFile("", "cron")
	if err != nil {
		return err
	}
	defer os.Remove(tmpfile.Name())
	out, err := exec.Command("crontab", "-l").Output()
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) || !strings.Contains(string(exitErr.Stderr), "no crontab for") {
			return fmt.Errorf("read existing crontab: %w", err)
		}
		out = []byte{}
	}
	if _, err := tmpfile.WriteString(string(out)); err != nil {
		return fmt.Errorf("write existing crontab to temp file: %w", err)
	}
	if _, err := tmpfile.WriteString(job); err != nil {
		return fmt.Errorf("append cron job to temp file: %w", err)
	}
	if err := tmpfile.Close(); err != nil {
		return fmt.Errorf("close crontab temp file: %w", err)
	}
	cmd := exec.Command("crontab", tmpfile.Name())
	return cmd.Run()
}

func runCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	return out.String(), err
}
