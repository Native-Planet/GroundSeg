package routines

import (
	"context"
	"fmt"
	"goseg/config"
	"goseg/system"
	"goseg/logger"
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
			counter++
		}
	}
	// advertise the http server
	_, err = zeroconf.RegisterProxy(
		strings.Split(LocalDomain, ".")[0],
		"_http._tcp",
		"local.",
		80,
		[]string{"txtv=0", "lo=1", "la=2"},
		nil,
	)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Failed to register mDNS server: %v", err))
		return
	}
	// also spoof the np hostname
	_, err = zeroconf.Register(
		strings.Split(LocalDomain, ".")[0],
		"_workstation._tcp",
		"local.",
		42069,
		nil,
		nil,
	)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Failed to advertise mDNS host: %v", err))
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