package accesspoint

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

var (
	wlan              = "wlan0"
	inet              = ""
	ip                = "192.168.45.1"
	netmask           = "255.255.255.0"
	ssid              = "NativePlanetConnect"
	password          = "nativeplanet"
	rootDir           = "/etc/accesspoint/"
	hostapdConfigPath = filepath.Join(rootDir, "hostapd.config")
)

func init() {
	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		os.Mkdir(rootDir, os.ModePerm)
	}
}

func Start(dev string) error {
	zap.L().Info(fmt.Sprintf("Starting router on %v", dev))
	wlan = dev
	// make sure dependencies are met
	if err := checkDependencies(); err != nil {
		return err
	}
	// make sure params are set (maybe not needed)
	if err := checkParameters(); err != nil {
		return err
	}
	// check if AP already running
	running, err := isRunning()
	if err != nil {
		return err
	}
	if running {
		zap.L().Info("Accesspoint already started")
	}
	// dump config to file
	if err := writeHostapdConfig(); err != nil {
		return err
	}
	// start the router
	startRouter()
	return nil
}

func Stop(dev string) error {
	zap.L().Info(fmt.Sprintf("Stopping router on %v", dev))
	wlan = dev
	// make sure params are set (maybe not needed)
	if err := checkParameters(); err != nil {
		return err
	}
	// check if AP is running
	running, err := isRunning()
	if err != nil {
		return err
	}
	// stop the router
	if running {
		stopRouter()
	} else {
		zap.L().Info("Accesspoint already stopped")
	}
	return nil
}

// general internal functions

// checks if either 'hostapd' or 'dnsmasq' processes are running
func isRunning() (bool, error) {
	// Run 'pgrep' command to find processes by name
	out, err := exec.Command("pgrep", "-af", "'hostapd|dnsmasq'").Output()
	if err != nil {
		// If err is not nil, pgrep didn't find the processes, which is not an error in our case
		return false, nil
	}
	// Convert output to string and check if 'hostapd' or 'dnsmasq' is in it
	processOutput := string(out)
	if strings.Contains(processOutput, "hostapd") || strings.Contains(processOutput, "dnsmasq") {
		return true, nil
	}
	return false, nil
}

func checkDependencies() error {
	if _, err := exec.LookPath("hostapd"); err != nil {
		return err
	}
	if _, err := exec.LookPath("dnsmasq"); err != nil {
		return err
	}
	return nil
}

// ExecuteShell executes a shell command and returns its output
func executeShell(commandString string) (string, error) {
	zap.L().Debug(fmt.Sprintf("%v", commandString))
	// Initialize the command
	cmd := exec.Command("sh", "-c", commandString)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute the command
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("command failed: %s", stderr.String())
	}

	// Decode the result
	zap.L().Debug(stdout.String())
	return stdout.String(), nil
}
