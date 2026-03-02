package accesspoint

import (
	"fmt"
	"log"
	"strings"
	"time"

	"go.uber.org/zap"
)

var (
	executeRouterShellFn = executeShell
	routerSleepFn        = time.Sleep
)

func startRouter() error {
	return startRouterWithRuntime(accessPointRuntime())
}

func stopRouter() error {
	return stopRouterWithRuntime(accessPointRuntime())
}

func startRouterWithRuntime(rt AccessPointRuntime) error {
	if err := preStart(); err != nil {
		return fmt.Errorf("pre-start configuration: %w", err)
	}
	if err := executeRouterCommand("ip", "link", "set", rt.Wlan, "up"); err != nil {
		return fmt.Errorf("start wlan interface: %w", err)
	}
	if err := executeRouterCommand("ip", "addr", "add", rt.IP+"/24", "dev", rt.Wlan); err != nil {
		return fmt.Errorf("assign AP IP address: %w", err)
	}
	zap.L().Debug("created interface")

	zap.L().Debug(fmt.Sprintf("wait.."))
	routerSleepFn(2 * time.Second)

	ipParts := rt.IP[:strings.LastIndex(rt.IP, ".")]

	zap.L().Debug(fmt.Sprintf("enabling forward in sysctl."))
	if err := executeRouterCommand("sysctl -w net.ipv4.ip_forward=1"); err != nil {
		return fmt.Errorf("enable ipv4 forwarding: %w", err)
	}

	if rt.Inet != "" {
		zap.L().Debug(fmt.Sprintf("creating NAT using iptables: %s <-> %s", rt.Wlan, rt.Inet))
		if err := executeRouterCommand("iptables -P FORWARD ACCEPT"); err != nil {
			return fmt.Errorf("set forward policy: %w", err)
		}
		if err := executeRouterCommand("iptables", "--table", "nat", "--delete-chain"); err != nil {
			return fmt.Errorf("delete nat chain: %w", err)
		}
		if err := executeRouterCommand("iptables", "--table", "nat", "-F"); err != nil {
			return fmt.Errorf("flush nat table: %w", err)
		}
		if err := executeRouterCommand("iptables", "--table", "nat", "-X"); err != nil {
			return fmt.Errorf("delete nat chains: %w", err)
		}
		if err := executeRouterCommand("iptables", "-t", "nat", "-A", "POSTROUTING", "-o", rt.Wlan, "-j", "MASQUERADE"); err != nil {
			return fmt.Errorf("configure masquerade: %w", err)
		}
		if err := executeRouterCommand("iptables", "-A", "FORWARD", "-i", rt.Wlan, "-o", rt.Wlan, "-j", "ACCEPT", "-m", "state", "--state", "RELATED,ESTABLISHED"); err != nil {
			return fmt.Errorf("configure forward established allow: %w", err)
		}
		if err := executeRouterCommand("iptables", "-A", "FORWARD", "-i", rt.Wlan, "-o", rt.Wlan, "-j", "ACCEPT"); err != nil {
			return fmt.Errorf("configure forward accept: %w", err)
		}
	}

	if err := executeRouterCommand("iptables", "-A", "OUTPUT", "--out-interface", rt.Wlan, "-j", "ACCEPT"); err != nil {
		return fmt.Errorf("allow output on AP interface: %w", err)
	}
	if err := executeRouterCommand("iptables", "-A", "INPUT", "--in-interface", rt.Wlan, "-j", "ACCEPT"); err != nil {
		return fmt.Errorf("allow input on AP interface: %w", err)
	}

	zap.L().Debug(fmt.Sprintf("running dnsmasq"))
	if err := executeRouterCommand("dnsmasq", "--dhcp-authoritative", fmt.Sprintf("--interface=%s", rt.Wlan), fmt.Sprintf("--dhcp-range=%s.20,%s.100,%s,4h", ipParts, ipParts, rt.Netmask)); err != nil {
		return fmt.Errorf("start dnsmasq: %w", err)
	}

	zap.L().Debug(fmt.Sprintf("running hostapd"))
	zap.L().Debug(fmt.Sprintf("wait.."))
	if err := executeRouterCommand("sleep", "2"); err != nil {
		return fmt.Errorf("hostapd warmup delay: %w", err)
	}
	if err := executeRouterCommand("hostapd", "-B", rt.HostapdConfigPath); err != nil {
		return fmt.Errorf("start hostapd: %w", err)
	}

	zap.L().Debug(fmt.Sprintf("hotspot is running."))
	return nil
}

// stopRouter stops the router
func stopRouterWithRuntime(rt AccessPointRuntime) error {
	// Bring down the interface
	if err := executeRouterCommand("ip", "link", "set", rt.Wlan, "down"); err != nil {
		return fmt.Errorf("stop wlan interface: %w", err)
	}

	// Stop hostapd
	log.Println("stopping hostapd")
	if err := executeRouterCommand("pkill", "hostapd"); err != nil {
		return fmt.Errorf("stop hostapd: %w", err)
	}

	// Stop dnsmasq
	log.Println("stopping dnsmasq")
	if err := executeRouterCommand("killall", "dnsmasq"); err != nil {
		return fmt.Errorf("stop dnsmasq: %w", err)
	}

	// Disable forwarding in iptables
	log.Println("disabling forward rules in iptables.")
	if err := executeRouterCommand("iptables", "-P", "FORWARD", "DROP"); err != nil {
		return fmt.Errorf("drop forward policy: %w", err)
	}

	// Delete iptables rules that were added for wlan traffic
	if rt.Wlan != "" {
		if err := executeRouterCommand("iptables", "-D", "OUTPUT", "--out-interface", rt.Wlan, "-j", "ACCEPT"); err != nil {
			return fmt.Errorf("remove output firewall rule: %w", err)
		}
		if err := executeRouterCommand("iptables", "-D", "INPUT", "--in-interface", rt.Wlan, "-j", "ACCEPT"); err != nil {
			return fmt.Errorf("remove input firewall rule: %w", err)
		}
	}
	if err := executeRouterCommand("iptables", "--table", "nat", "--delete-chain"); err != nil {
		return fmt.Errorf("remove nat chain: %w", err)
	}
	if err := executeRouterCommand("iptables", "--table", "nat", "-F"); err != nil {
		return fmt.Errorf("flush nat table: %w", err)
	}
	if err := executeRouterCommand("iptables", "--table", "nat", "-X"); err != nil {
		return fmt.Errorf("delete nat table: %w", err)
	}

	// Disable forwarding in sysctl
	log.Println("disabling forward in sysctl.")
	if err := executeRouterCommand("sysctl", "-w", "net.ipv4.ip_forward=0"); err != nil {
		return fmt.Errorf("disable ipv4 forwarding: %w", err)
	}

	log.Println("hotspot has stopped.")
	return nil
}

func preStart() error {
	var res string
	var err error
	// execute 'killall wpa_supplicant'
	res, err = executeRouterShellCommand("killall", "wpa_supplicant")
	if err != nil {
		return fmt.Errorf("stop wpa_supplicant: %w", err)
	}
	// execute 'nmcli radio wifi off' and check for errors
	res, err = executeRouterShellCommand("nmcli", "radio", "wifi", "off")
	if err != nil {
		return fmt.Errorf("disable wifi radio (nmcli): %w", err)
	}
	zap.L().Debug(fmt.Sprintf("res: %s, err: %v", res, err))
	if strings.Contains(strings.ToLower(res), "error") {
		res, err = executeRouterShellCommand("nmcli", "nm", "wifi", "off")
		if err != nil {
			return fmt.Errorf("disable wifi radio (nmcli legacy): %w", err)
		}
		zap.L().Debug(fmt.Sprintf("res: %s, err: %v", res, err))
		if strings.Contains(strings.ToLower(res), "error") {
			return fmt.Errorf("legacy nmcli wifi disable returned error output: %s", strings.TrimSpace(res))
		}
	}
	// execute 'rfkill unblock wlan'
	res, err = executeRouterShellCommand("rfkill", "unblock", "wlan")
	if err != nil {
		return fmt.Errorf("unblock wlan interface: %w", err)
	}
	zap.L().Debug(fmt.Sprintf("res: %s, err: %v", res, err))
	// execute 'sleep 1'
	res, err = executeRouterShellCommand("sleep", "1")
	if err != nil {
		return fmt.Errorf("delay before startup: %w", err)
	}
	zap.L().Debug(fmt.Sprintf("res: %s, err: %v", res, err))
	return nil
}

func executeRouterShellCommand(name string, args ...string) (string, error) {
	cmd := formatShellCommand(name, args...)
	return executeRouterShellFn(cmd)
}

func executeRouterCommand(name string, args ...string) error {
	cmd := formatShellCommand(name, args...)
	res, err := executeRouterShellFn(cmd)
	zap.L().Debug(fmt.Sprintf("res: %s, err: %v", res, err))
	if err != nil {
		zap.L().Warn(fmt.Sprintf("accesspoint command failed: %s: %v", cmd, err))
		return err
	}
	return nil
}

func formatShellCommand(name string, args ...string) string {
	if len(args) == 0 {
		return name
	}
	return name + " " + strings.Join(args, " ")
}
