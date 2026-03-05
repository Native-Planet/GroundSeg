package accesspoint

import (
	"fmt"
	"net"
	"strings"
)

// validateIP validates the IP address format.
func validateIP(ip string) bool {
	parsedIP := net.ParseIP(strings.TrimSpace(ip))
	if parsedIP == nil {
		return false
	}
	return parsedIP.To4() != nil
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
	return checkParametersWithContext(accessPointRuntime())
}

func checkParametersWithContext(rt AccessPointRuntime) error {
	// get interfaces
	if rt.NetInterfacesFn == nil {
		return fmt.Errorf("net interfaces function is not configured")
	}
	interfaces, err := rt.NetInterfacesFn()
	if err != nil {
		return err
	}
	// Convert to a list of interface names for easier lookup
	var interfaceNames []string
	for _, iface := range interfaces {
		interfaceNames = append(interfaceNames, iface.Name)
	}

	// Check wlan interface
	if rt.Wlan != "" && !hasInterface(interfaceNames, rt.Wlan) {
		return fmt.Errorf("Wlan %s interface was not found", rt.Wlan)
	}

	// Check inet interface
	if rt.Inet != "" && !hasInterface(interfaceNames, rt.Inet) {
		return fmt.Errorf("Inet %s interface was not found", rt.Inet)
	}

	// Validate IP
	if !validateIP(rt.IP) {
		return fmt.Errorf("Wrong ip %s", rt.IP)
	}

	// Check SSID
	if rt.SSID == "" {
		return fmt.Errorf("SSID must not be empty")
	}

	// Check password
	if rt.Password == "" {
		return fmt.Errorf("Password must not be empty")
	}
	return nil
}
