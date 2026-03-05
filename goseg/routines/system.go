package routines

import (
	"context"
	"fmt"
	"groundseg/config"
	"groundseg/structs"
	"groundseg/system"
	maintenanceapt "groundseg/system/maintenance/apt"
	"net"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
	"go.uber.org/zap"
)

var (
	LocalDomain = "nativeplanet.local"

	updateCheckForAptLoop          = maintenanceapt.UpdateCheck
	configForSystemRoutine         = config.Config
	netInterfacesForSystemRoutine  = net.Interfaces
	interfaceAddrsForSystemRoutine = func(i net.Interface) ([]net.Addr, error) {
		return i.Addrs()
	}
)

func StartMDNSServer() {
	go mDNSServer()
}

func AptUpdateLoop() {
	_ = AptUpdateLoopWithContext(context.Background())
}

func AptUpdateLoopWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	updateCheckForAptLoop()
	conf := configForSystemRoutine()
	checkInterval := aptUpdateCheckInterval(conf)
	if len(conf.Runtime.LastKnownMDNS) > 0 {
		system.SetLocalUrl(conf.Runtime.LastKnownMDNS)
	} else {
		system.SetLocalUrl(LocalDomain)
	}
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			updateCheckForAptLoop()
		}
	}
}

func aptUpdateCheckInterval(conf structs.SysConfig) time.Duration {
	val := time.Duration(conf.Runtime.LinuxUpdates.Value)
	if val <= 0 {
		val = 1
	}
	var interval time.Duration
	if interv := conf.Runtime.LinuxUpdates.Interval; interv == "week" {
		interval = 7 * (time.Hour * 24)
	} else if interv == "day" {
		interval = time.Hour * 24
	} else {
		interval = 30 * (time.Hour * 24)
	}
	return val * interval
}

func mDNSServer() {
	conf := config.Config()
	if !conf.Runtime.GracefulExit && (len(conf.Runtime.LastKnownMDNS) > 0) {
		system.SetLocalUrl(conf.Runtime.LastKnownMDNS)
	} else {
		domains, err := mDNSDiscovery()
		if err != nil {
			zap.L().Warn("Couldn't discover mDNS servers on LAN -- defaulting to nativeplanet.local")
		} else {
			// check if there's already a nativeplanet.local
			counter := 2
			domainBase := strings.Split(LocalDomain, ".")[0]
			for contains(domains, domainBase) {
				LocalDomain = fmt.Sprintf("nativeplanet%d.local", counter)
				zap.L().Info(fmt.Sprintf("Incrementing to %v...", LocalDomain))
				counter++
				domainBase = strings.Split(LocalDomain, ".")[0]
			}
		}
		system.SetLocalUrl(LocalDomain)
	}
	// advertise the http server on loop
	// we use RegisterProxy so we can spoof the hostname
	for {
		ips, err := getAllIPs()
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		localUrl := system.LocalUrl()
		zap.L().Info(fmt.Sprintf("Announcing %v for %v", localUrl, ips))
		server, err := zeroconf.RegisterProxy(
			strings.Split(localUrl, ".")[0],
			"_http._tcp",
			"local.",
			80,
			strings.Split(localUrl, ".")[0],
			ips,
			[]string{"txtv=0", "lo=1", "la=2"},
			nil,
		)
		if err != nil {
			zap.L().Error(fmt.Sprintf("Failed to announce mDNS server: %v", err))
		} else {
			zap.L().Info(fmt.Sprintf("Caching %v", localUrl))
			if err = config.UpdateConfigTyped(
				config.WithGracefulExit(false),
				config.WithLastKnownMDNS(localUrl),
			); err != nil {
				zap.L().Error(fmt.Sprintf("Couldn't update mdns cache: %v", err))
			}
		}
		time.Sleep(120 * time.Second)
		server.Shutdown()
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
		domainParts := strings.Split(entry.ServiceInstanceName(), ".")
		if len(domainParts) > 0 {
			hosts = append(hosts, domainParts[0])
		}
	}
	return hosts, nil
}

func getAllIPs() ([]string, error) {
	var ips []string
	interfaces, err := netInterfacesForSystemRoutine()
	if err != nil {
		return nil, err
	}
	for _, i := range interfaces {
		addrs, err := interfaceAddrsForSystemRoutine(i)
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
