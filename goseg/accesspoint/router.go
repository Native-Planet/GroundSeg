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
	preStart()
	if err := executeRouterCommand("ip link set " + wlan + " up"); err != nil {
		return fmt.Errorf("start wlan interface: %w", err)
	}
	if err := executeRouterCommand("ip addr add " + ip + "/24 dev " + wlan); err != nil {
		return fmt.Errorf("assign AP IP address: %w", err)
	}
	zap.L().Debug("created interface")

	zap.L().Debug(fmt.Sprintf("wait.."))
	routerSleepFn(2 * time.Second)

	ipParts := ip[:strings.LastIndex(ip, ".")]

	zap.L().Debug(fmt.Sprintf("enabling forward in sysctl."))
	if err := executeRouterCommand("sysctl -w net.ipv4.ip_forward=1"); err != nil {
		return fmt.Errorf("enable ipv4 forwarding: %w", err)
	}

	if inet != "" {
		zap.L().Debug(fmt.Sprintf("creating NAT using iptables: %s <-> %s", wlan, inet))
		if err := executeRouterCommand("iptables -P FORWARD ACCEPT"); err != nil {
			return fmt.Errorf("set forward policy: %w", err)
		}
		if err := executeRouterCommand("iptables --table nat --delete-chain"); err != nil {
			return fmt.Errorf("delete nat chain: %w", err)
		}
		if err := executeRouterCommand("iptables --table nat -F"); err != nil {
			return fmt.Errorf("flush nat table: %w", err)
		}
		if err := executeRouterCommand("iptables --table nat -X"); err != nil {
			return fmt.Errorf("delete nat chains: %w", err)
		}
		if err := executeRouterCommand("iptables -t nat -A POSTROUTING -o " + wlan + " -j MASQUERADE"); err != nil {
			return fmt.Errorf("configure masquerade: %w", err)
		}
		if err := executeRouterCommand("iptables -A FORWARD -i " + wlan + " -o " + wlan + " -j ACCEPT -m state --state RELATED,ESTABLISHED"); err != nil {
			return fmt.Errorf("configure forward established allow: %w", err)
		}
		if err := executeRouterCommand("iptables -A FORWARD -i " + wlan + " -o " + wlan + " -j ACCEPT"); err != nil {
			return fmt.Errorf("configure forward accept: %w", err)
		}
	}

	if err := executeRouterCommand("iptables -A OUTPUT --out-interface " + wlan + " -j ACCEPT"); err != nil {
		return fmt.Errorf("allow output on AP interface: %w", err)
	}
	if err := executeRouterCommand("iptables -A INPUT --in-interface " + wlan + " -j ACCEPT"); err != nil {
		return fmt.Errorf("allow input on AP interface: %w", err)
	}

	cmd := "dnsmasq --dhcp-authoritative --interface=" + wlan + " --dhcp-range=" + ipParts + ".20," + ipParts + ".100," + netmask + ",4h"
	zap.L().Debug(fmt.Sprintf("running dnsmasq"))
	if err := executeRouterCommand(cmd); err != nil {
		return fmt.Errorf("start dnsmasq: %w", err)
	}

	cmd = "hostapd -B " + hostapdConfigPath
	zap.L().Debug(fmt.Sprintf("running hostapd"))
	zap.L().Debug(fmt.Sprintf("wait.."))
	if err := executeRouterCommand("sleep 2"); err != nil {
		return fmt.Errorf("hostapd warmup delay: %w", err)
	}
	if err := executeRouterCommand(cmd); err != nil {
		return fmt.Errorf("start hostapd: %w", err)
	}

	zap.L().Debug(fmt.Sprintf("hotspot is running."))
	return nil
}

// stopRouter stops the router
func stopRouter() error {
	// Bring down the interface
	cmd := "ip link set " + wlan + " down"
	if err := executeRouterCommand(cmd); err != nil {
		return fmt.Errorf("stop wlan interface: %w", err)
	}

	// Stop hostapd
	log.Println("stopping hostapd")
	if err := executeRouterCommand("pkill hostapd"); err != nil {
		return fmt.Errorf("stop hostapd: %w", err)
	}

	// Stop dnsmasq
	log.Println("stopping dnsmasq")
	if err := executeRouterCommand("killall dnsmasq"); err != nil {
		return fmt.Errorf("stop dnsmasq: %w", err)
	}

	// Disable forwarding in iptables
	log.Println("disabling forward rules in iptables.")
	if err := executeRouterCommand("iptables -P FORWARD DROP"); err != nil {
		return fmt.Errorf("drop forward policy: %w", err)
	}

	// Delete iptables rules that were added for wlan traffic
	if wlan != "" {
		if err := executeRouterCommand("iptables -D OUTPUT --out-interface " + wlan + " -j ACCEPT"); err != nil {
			return fmt.Errorf("remove output firewall rule: %w", err)
		}
		if err := executeRouterCommand("iptables -D INPUT --in-interface " + wlan + " -j ACCEPT"); err != nil {
			return fmt.Errorf("remove input firewall rule: %w", err)
		}
	}
	if err := executeRouterCommand("iptables --table nat --delete-chain"); err != nil {
		return fmt.Errorf("remove nat chain: %w", err)
	}
	if err := executeRouterCommand("iptables --table nat -F"); err != nil {
		return fmt.Errorf("flush nat table: %w", err)
	}
	if err := executeRouterCommand("iptables --table nat -X"); err != nil {
		return fmt.Errorf("delete nat table: %w", err)
	}

	// Disable forwarding in sysctl
	log.Println("disabling forward in sysctl.")
	if err := executeRouterCommand("sysctl -w net.ipv4.ip_forward=0"); err != nil {
		return fmt.Errorf("disable ipv4 forwarding: %w", err)
	}

	log.Println("hotspot has stopped.")
	return nil
}

func preStart() {
	var res string
	var err error
	// execute 'killall wpa_supplicant'
	res, err = executeRouterShellFn("killall wpa_supplicant")
	// execute 'nmcli radio wifi off' and check for errors
	res, err = executeRouterShellFn("nmcli radio wifi off")
	zap.L().Debug(fmt.Sprintf("res: %s, err: %v", res, err))
	if strings.Contains(strings.ToLower(res), "error") {
		res, err = executeRouterShellFn("nmcli nm wifi off")
		zap.L().Debug(fmt.Sprintf("res: %s, err: %v", res, err))
	}
	// execute 'rfkill unblock wlan'
	res, err = executeRouterShellFn("rfkill unblock wlan")
	zap.L().Debug(fmt.Sprintf("res: %s, err: %v", res, err))
	// execute 'sleep 1'
	res, err = executeRouterShellFn("sleep 1")
	zap.L().Debug(fmt.Sprintf("res: %s, err: %v", res, err))
}

func executeRouterCommand(cmd string) error {
	res, err := executeRouterShellFn(cmd)
	zap.L().Debug(fmt.Sprintf("res: %s, err: %v", res, err))
	if err != nil {
		zap.L().Warn(fmt.Sprintf("accesspoint command failed: %s: %v", cmd, err))
		return err
	}
	return nil
}
