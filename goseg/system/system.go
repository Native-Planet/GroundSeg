package system

// for retrieving hw info and managing host

import (
	"context"
	"fmt"
	"goseg/defaults"
	"goseg/logger"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
)

var (
	LocalDomain = "nativeplanet.local"
)

func init() {
	go mDNSServer()
}

func mDNSServer() {
	domains, err := mDNSDiscovery()
	if err != nil {
		logger.Logger.Warn("Couldn't discover mDNS servers on LAN -- defaulting to nativeplanet.local")
	} else {
		// check if there's already a nativeplanet.local
		counter := 2
		for contains(domains, LocalDomain) {
			LocalDomain = fmt.Sprintf("nativeplanet%d.local", counter)
			counter++
		}
	}
	_, err = zeroconf.Register(
		strings.Split(LocalDomain, ".")[0],
		"_http._tcp",
		"local.",
		80,
		[]string{"txtv=0", "lo=1", "la=2"},
		nil,
	)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Failed to register service: %v", err))
		return
	}
	logger.Logger.Info(fmt.Sprintf("Registered %v mDNS domain", LocalDomain))
	// infinite blocking
	select {}
}

// return a slice of all discovered .local domains
func mDNSDiscovery() ([]string, error) {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, err
	}
	entries := make(chan *zeroconf.ServiceEntry)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	go func() {
		err = resolver.Browse(ctx, "_http._tcp", "local.", entries)
		if err != nil {
			close(entries)
		}
	}()
	var hosts []string
	for entry := range entries {
		hosts = append(hosts, entry.ServiceInstanceName())
	}
	return hosts, nil
}

func contains(slice []string, item string) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}

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
