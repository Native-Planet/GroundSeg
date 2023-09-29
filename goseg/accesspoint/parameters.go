package accesspoint

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

// validateIP validates the IP address format.
func validateIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	return parsedIP != nil
}

func checkParameters() error {
	// get interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		return err
	}
	// Convert to a list of interface names for easier lookup
	var interfaceNames []string
	for _, iface := range interfaces {
		interfaceNames = append(interfaceNames, iface.Name)
	}

	// Check wlan interface
	if inet != "" {
		if !strings.Contains(strings.Join(interfaceNames, ","), wlan) {
			return errors.New(fmt.Sprintf("Wlan %s interface was not found", wlan))
		}
	}

	// Check inet interface
	if inet != "" && !strings.Contains(strings.Join(interfaceNames, ","), inet) {
		return errors.New(fmt.Sprintf("Inet %s interface was not found", inet))
	}

	// Validate IP
	if !validateIP(ip) {
		return errors.New(fmt.Sprintf("Wrong ip %s", ip))
	}

	// Check SSID
	if ssid == "" {
		return errors.New("SSID must not be empty")
	}

	// Check password
	if password == "" {
		return errors.New("Password must not be empty")
	}
	return nil
}
