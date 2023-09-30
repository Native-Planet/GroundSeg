package routines

import (
	"context"
	"fmt"
	"goseg/config"
	"goseg/logger"
	"goseg/system"
	"net"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
)

var (
	LocalDomain = "nativeplanet.local"
)

func init() {
	go mDNSServer()
}

func AptUpdateLoop() {
	system.UpdateCheck()
	conf := config.Conf()
	val := time.Duration(conf.LinuxUpdates.Value)

	var interval time.Duration
	if interv := conf.LinuxUpdates.Interval; interv == "week" {
		interval = 7 * (time.Hour * 24)
	} else if interv == "day" {
		interval = time.Hour * 24
	} else {
		interval = 30 * (time.Hour * 24)
	}
	checkInterval := val * interval
	ticker := time.NewTicker(checkInterval)
	for {
		select {
		case <-ticker.C:
			system.UpdateCheck()
		}
	}
}

func mDNSServer() {
	domains, err := mDNSDiscovery()
	if err != nil {
		logger.Logger.Warn("Couldn't discover mDNS servers on LAN -- defaulting to nativeplanet.local")
	} else {
		// check if there's already a nativeplanet.local
		counter := 2
		for contains(domains, strings.Split(LocalDomain, ".")[0]) {
			LocalDomain = fmt.Sprintf("nativeplanet%d.local", counter)
			logger.Logger.Info(fmt.Sprintf("Incrementing to %v...", LocalDomain))
			counter++
		}
	}
	system.LocalUrl = LocalDomain
	// advertise the http server on loop
	// we use RegisterProxy so we can spoof the hostname
	for {
		ips, err := getAllIPs()
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		logger.Logger.Debug(fmt.Sprintf("Announcing %v for %v", system.LocalUrl, ips))
		server, err := zeroconf.RegisterProxy(
			strings.Split(system.LocalUrl, ".")[0],
			"_http._tcp",
			"local.",
			80,
			strings.Split(system.LocalUrl, ".")[0],
			ips,
			[]string{"txtv=0", "lo=1", "la=2"},
			nil,
		)
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("Failed to announce mDNS server: %v", err))
		}
		server.Shutdown()
		time.Sleep(120 * time.Second)
	}
	// reannounce every 2 minutes
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

func getAllIPs() ([]string, error) {
	var ips []string
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, i := range interfaces {
		addrs, err := i.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip.To4() == nil {
				continue // skip ipv6
			}
			ipStr := ip.String()
			if strings.HasPrefix(ipStr, "127") || strings.HasPrefix(ipStr, "172.1") || strings.HasPrefix(ipStr, "172.2") {
				continue // skip local-only IPs
			}
			ips = append(ips, ipStr)
		}
	}
	return ips, nil
}
