package accesspoint

import (
	"fmt"
	"net"
)

var netInterfacesFn = net.Interfaces

// validateIP validates the IP address format.
func validateIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	return parsedIP != nil
}

func hasInterface(interfaceNames []string, target string) bool {
	for _, name := range interfaceNames {
		if name == target {
			return true
		}
	}
	return false
}

func checkParameters() error {
	// get interfaces
	interfaces, err := netInterfacesFn()
	if err != nil {
		return err
	}
	// Convert to a list of interface names for easier lookup
	var interfaceNames []string
	for _, iface := range interfaces {
		interfaceNames = append(interfaceNames, iface.Name)
	}

	// Check wlan interface
	if wlan != "" && !hasInterface(interfaceNames, wlan) {
		return fmt.Errorf("Wlan %s interface was not found", wlan)
	}

	// Check inet interface
	if inet != "" && !hasInterface(interfaceNames, inet) {
		return fmt.Errorf("Inet %s interface was not found", inet)
	}

	// Validate IP
	if !validateIP(ip) {
		return fmt.Errorf("Wrong ip %s", ip)
	}

	// Check SSID
	if ssid == "" {
		return fmt.Errorf("SSID must not be empty")
	}

	// Check password
	if password == "" {
		return fmt.Errorf("Password must not be empty")
	}
	return nil
}
