package main

// NativePlanet GroundSeg: Go Edition (goseg)
// 🄯 2023 ~nallux-dozryl & ~sitful-hatred
// This is a Golang rewrite of GroundSeg that serves the v2 json
// object via websocket.
// The v2 rewrite decouples the frontend and backend, which makes it
// straightforward to implement alternative backends.
//
// Under development: reimplementing all pyseg functionality.
// Advantages:
// - Really, really fast
// - Event-driven
// - First-class support for concurrent operations
// - Very good golang Docker libraries

import (
	"fmt"
	"goseg/config"
	"goseg/docker"
	"goseg/rectify"
	"goseg/startram"
	"goseg/ws"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

var (
	DevMode = false
)

func loadService(loadFunc func() error, errMsg string) {
	go func() {
		if err := loadFunc(); err != nil {
			config.Logger.Error(fmt.Sprintf("%s %v", errMsg, err))
		}
	}()
}

func main() {
	// global SysConfig var is managed through config package
	conf := config.Conf()
	internetAvailable := config.NetCheck("1.1.1.1:53")
	availMsg := fmt.Sprintf("Internet available: %t", internetAvailable)
	config.Logger.Info(availMsg)
	// async operation to retrieve version info if updates are on
	versionUpdateChannel := make(chan bool)
	remoteVersion := false
	if conf.UpdateMode == "auto" {
		remoteVersion = true
		// get version info from remote server
		go func() {
			_, versionUpdate := config.CheckVersion()
			if versionUpdate {
				config.Logger.Info("Version info retrieved")
			}
			versionUpdateChannel <- versionUpdate
		}()
		// otherwise use cached if possible, or save hardcoded defaults and use that
	} else {
		versionStruct := config.LocalVersion()
		releaseChannel := conf.UpdateBranch
		targetChan := versionStruct.Groundseg[releaseChannel]
		config.VersionInfo = targetChan
	}
	// infinite version check loop
	go config.CheckVersionLoop()
	// listen to docker daemon
	go docker.DockerListener()
	// digest docker events from eventbus
	go rectify.DockerSubscriptionHandler()
	// digest urbit transition events
	go rectify.UrbitTransitionHandler()
	// just making sure we can parse (debug)
	if len(conf.Piers) > 0 {
		pierList := strings.Join(conf.Piers, ", ")
		config.Logger.Info(fmt.Sprintf("Loaded piers: %s", pierList))
	}
	// get the startram config from server
	if conf.WgRegistered == true {
		_, err := startram.Retrieve()
		if err != nil {
			config.Logger.Warn(fmt.Sprintf("Could not retrieve StarTram/Anchor config: %v", err))
		}
	}
	// block until version info returns
	if remoteVersion == true {
		select {
		case <-versionUpdateChannel:
			config.Logger.Info("Version info retrieved")
		case <-time.After(10 * time.Second):
			config.Logger.Warn("Could not retrieve version info after 10 seconds!")
			versionStruct := config.LocalVersion()
			releaseChannel := conf.UpdateBranch
			targetChan := versionStruct.Groundseg[releaseChannel]
			config.VersionInfo = targetChan
		}
	}
	if conf.WgRegistered == true {
		// Load Wireguard
		loadService(docker.LoadWireguard, "Unable to load Wireguard!")
		// Load MC
		loadService(docker.LoadMC, "Unable to load MinIO Client!")
		// Load MinIOs
		loadService(docker.LoadMinIOs, "Unable to load MinIO containers!")
	}
	// Load Netdata
	loadService(docker.LoadNetdata, "Unable to load Netdata!")
	// Load Urbits
	loadService(docker.LoadUrbits, "Unable to load Urbit ships!")

	// Websocket
	r := mux.NewRouter()
	r.HandleFunc("/ws", ws.WsHandler)
	http.ListenAndServe(":3000", r)
}
