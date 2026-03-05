package metrics

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
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
	cpuPercentFn      = cpu.Percent
	openFn            = os.Open
	readDirFn         = ioutil.ReadDir
	readlinkFn        = os.Readlink
	readFileFn        = ioutil.ReadFile
	globFn            = filepath.Glob
	statfsFn          = syscall.Statfs
	queryUnescapeFn   = url.QueryUnescape
	mountsPath        = "/proc/mounts"
	diskLabelBasePath = "/dev/disk/by-label/"
	hwmonBasePath     = "/sys/class/hwmon/"
)

// GetMemory returns total and used memory in bytes.
func GetMemory() (uint64, uint64, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return 0, 0, fmt.Errorf("read virtual memory stats: %w", err)
	}
	return v.Total, v.Used, nil
}

// GetCPU returns the current CPU utilization percentage.
func GetCPU() (int, error) {
	percent, err := cpuPercentFn(time.Second, false)
	if err != nil {
		return 0, fmt.Errorf("read CPU usage: %w", err)
	}
	if len(percent) == 0 {
		return 0, fmt.Errorf("read CPU usage: empty CPU percentage response")
	}
	return int(percent[0]), nil
}

// GetDisk returns used/available bytes per mounted disk label.
func GetDisk() (map[string][2]uint64, error) {
	diskUsageMap := make(map[string][2]uint64)
	file, err := openFn(mountsPath)
	if err != nil {
		return diskUsageMap, fmt.Errorf("open /proc/mounts: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 {
			device := fields[0]
			mountPoint, err := octalToAscii(fields[1])
			if err != nil {
				zap.L().Warn(fmt.Sprintf("Skipping mount point %s: %v", fields[1], err))
				continue
			}
			if !strings.HasPrefix(device, "/dev/") || strings.HasPrefix(device, "/dev/loop") {
				continue
			}
			var stat syscall.Statfs_t
			if err := statfsFn(mountPoint, &stat); err != nil {
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
		return diskUsageMap, fmt.Errorf("read /proc/mounts: %w", err)
	}
	return diskUsageMap, nil
}

func getDiskLabel(device string) (string, string) {
	files, err := readDirFn(diskLabelBasePath)
	if err != nil {
		return "", ""
	}
	for _, f := range files {
		fullPath := filepath.Join(diskLabelBasePath, f.Name())
		resolvedPath, err := readlinkFn(fullPath)
		if err != nil {
			zap.L().Warn(fmt.Sprintf("Unable to read disk label source for %s: %v", fullPath, err))
			continue
		}
		if strings.HasSuffix(resolvedPath, device) {
			label, err := queryUnescapeFn(f.Name())
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

func octalToAscii(s string) (string, error) {
	re := regexp.MustCompile(`\\[0-7]{3}`)
	var parseErr error
	replaceFunc := func(match string) string {
		i, err := strconv.ParseInt(match[1:], 8, 64)
		if err != nil {
			parseErr = err
			return match
		}
		return string(rune(i))
	}
	decoded := re.ReplaceAllStringFunc(s, replaceFunc)
	if parseErr != nil {
		return "", parseErr
	}
	return decoded, nil
}

// GetTemp returns average core temperature in degrees C when available.
func GetTemp() (float64, error) {
	hwmons, err := readDirFn(hwmonBasePath)
	if err != nil {
		return 0, fmt.Errorf("Error reading the hwmon directory: %w", err)
	}
	var totalTemp float64
	var tempCount int
	for _, hwmon := range hwmons {
		path := filepath.Join(hwmonBasePath, hwmon.Name())
		devicePath := filepath.Join(path, "name")
		device, err := readFileFn(devicePath)
		if err != nil {
			continue
		}
		if strings.Contains(strings.ToLower(string(device)), "coretemp") {
			tempInputs, _ := globFn(filepath.Join(path, "temp*_input"))
			for _, tempInput := range tempInputs {
				temp, err := readFileFn(tempInput)
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
