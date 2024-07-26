package accesspoint

import (
	"fmt"
	"log"
	"strings"
	"time"

	"go.uber.org/zap"
)

func startRouter() bool {
	var res string
	var err error
	preStart()
	cmd := "ip link set " + wlan + " up"
	res, err = executeShell(cmd)
	zap.L().Debug(fmt.Sprintf("res: %s, err: %v", res, err))
	cmd = "ip addr add " + ip + "/24 dev " + wlan
	res, err = executeShell(cmd)
	zap.L().Debug(fmt.Sprintf("res: %s, err: %v", res, err))
	zap.L().Debug("created interface")

	zap.L().Debug(fmt.Sprintf("wait.."))
	time.Sleep(2 * time.Second)

	ipParts := ip[:strings.LastIndex(ip, ".")]

	zap.L().Debug(fmt.Sprintf("enabling forward in sysctl."))
	res, err = executeShell("sysctl -w net.ipv4.ip_forward=1")
	zap.L().Debug(fmt.Sprintf("res: %s, err: %v", res, err))

	if inet != "" {
		zap.L().Debug(fmt.Sprintf("creating NAT using iptables: %s <-> %s", wlan, inet))
		res, err = executeShell("iptables -P FORWARD ACCEPT")
		zap.L().Debug(fmt.Sprintf("res: %s, err: %v", res, err))
		res, err = executeShell("iptables --table nat --delete-chain")
		zap.L().Debug(fmt.Sprintf("res: %s, err: %v", res, err))
		res, err = executeShell("iptables --table nat -F")
		zap.L().Debug(fmt.Sprintf("res: %s, err: %v", res, err))
		res, err = executeShell("iptables --table nat -X")
		zap.L().Debug(fmt.Sprintf("res: %s, err: %v", res, err))
		res, err = executeShell("iptables -t nat -A POSTROUTING -o " + wlan + " -j MASQUERADE")
		zap.L().Debug(fmt.Sprintf("res: %s, err: %v", res, err))
		res, err = executeShell("iptables -A FORWARD -i " + wlan + " -o " + wlan + " -j ACCEPT -m state --state RELATED,ESTABLISHED")
		zap.L().Debug(fmt.Sprintf("res: %s, err: %v", res, err))
		res, err = executeShell("iptables -A FORWARD -i " + wlan + " -o " + wlan + " -j ACCEPT")
		zap.L().Debug(fmt.Sprintf("res: %s, err: %v", res, err))
	}

	res, err = executeShell("iptables -A OUTPUT --out-interface " + wlan + " -j ACCEPT")
	zap.L().Debug(fmt.Sprintf("res: %s, err: %v", res, err))
	res, err = executeShell("iptables -A INPUT --in-interface " + wlan + " -j ACCEPT")
	zap.L().Debug(fmt.Sprintf("res: %s, err: %v", res, err))

	cmd = "dnsmasq --dhcp-authoritative --interface=" + wlan + " --dhcp-range=" + ipParts + ".20," + ipParts + ".100," + netmask + ",4h"
	zap.L().Debug(fmt.Sprintf("running dnsmasq"))
	res, err = executeShell(cmd)
	zap.L().Debug(fmt.Sprintf("res: %s, err: %v", res, err))

	cmd = "hostapd -B " + hostapdConfigPath
	zap.L().Debug(fmt.Sprintf("running hostapd"))
	zap.L().Debug(fmt.Sprintf("wait.."))
	res, err = executeShell("sleep 2")
	zap.L().Debug(fmt.Sprintf("res: %s, err: %v", res, err))
	res, err = executeShell(cmd)
	zap.L().Debug(fmt.Sprintf("res: %s, err: %v", res, err))

	zap.L().Debug(fmt.Sprintf("hotspot is running."))
	return true
}

// stopRouter stops the router
func stopRouter() bool {
	var res string
	var err error
	// Bring down the interface
	cmd := "ip link set " + wlan + " down"
	res, err = executeShell(cmd)
	zap.L().Debug(fmt.Sprintf("res: %s, err: %v", res, err))

	// Stop hostapd
	log.Println("stopping hostapd")
	res, err = executeShell("pkill hostapd")

	// Stop dnsmasq
	log.Println("stopping dnsmasq")
	res, err = executeShell("killall dnsmasq")

	// Disable forwarding in iptables
	log.Println("disabling forward rules in iptables.")
	res, err = executeShell("iptables -P FORWARD DROP")

	// Delete iptables rules that were added for wlan traffic
	if wlan != "" {
		res, err = executeShell("iptables -D OUTPUT --out-interface " + wlan + " -j ACCEPT")
		res, err = executeShell("iptables -D INPUT --in-interface " + wlan + " -j ACCEPT")
	}
	res, err = executeShell("iptables --table nat --delete-chain")
	res, err = executeShell("iptables --table nat -F")
	res, err = executeShell("iptables --table nat -X")

	// Disable forwarding in sysctl
	log.Println("disabling forward in sysctl.")
	res, err = executeShell("sysctl -w net.ipv4.ip_forward=0")

	log.Println("hotspot has stopped.")
	return true
}

func preStart() {
	var res string
	var err error
	// execute 'killall wpa_supplicant'
	res, err = executeShell("killall wpa_supplicant")
	// execute 'nmcli radio wifi off' and check for errors
	res, err = executeShell("nmcli radio wifi off")
	zap.L().Debug(fmt.Sprintf("res: %s, err: %v", res, err))
	if strings.Contains(strings.ToLower(res), "error") {
		res, err = executeShell("nmcli nm wifi off")
		zap.L().Debug(fmt.Sprintf("res: %s, err: %v", res, err))
	}
	// execute 'rfkill unblock wlan'
	res, err = executeShell("rfkill unblock wlan")
	zap.L().Debug(fmt.Sprintf("res: %s, err: %v", res, err))
	// execute 'sleep 1'
	res, err = executeShell("sleep 1")
	zap.L().Debug(fmt.Sprintf("res: %s, err: %v", res, err))
}
