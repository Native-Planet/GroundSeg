package system

// for retrieving hw info and managing host

import (
	"fmt"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"goseg/config"
	"io/ioutil"
	"strconv"
	"strings"
	"time"
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
		// config.Logger.Error(errmsg)
		return 0
	}
	tempStr := strings.TrimSpace(string(data))
	temp, err := strconv.Atoi(tempStr)
	if err != nil {
		errmsg := fmt.Sprintf("Error converting temperature to integer:", err)
		config.Logger.Error(errmsg)
		return 0
	}
	return float64(temp) / 1000.0
}

// return 0 for no 1 for yes(?)
func HasSwap() int {
	data, err := ioutil.ReadFile("/proc/swaps")
	if err != nil {
		errmsg := fmt.Sprintf("Error reading swap status:", err)
		config.Logger.Error(errmsg)
		return 0
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) > 1 {
		return 1
	} else {
		return 0
	}
}
