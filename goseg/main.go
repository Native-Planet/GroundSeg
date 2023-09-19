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
	"goseg/exporter"
	"goseg/logger"
	"goseg/rectify"
	"goseg/routines"
	"goseg/startram"
	"goseg/system"
	"goseg/ws"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

var (
	DevMode = false
)

func loadService(loadFunc func() error, errMsg string) {
	go func() {
		if err := loadFunc(); err != nil {
			logger.Logger.Error(fmt.Sprintf("%s %v", errMsg, err))
		}
	}()
}

func main() {
	// global SysConfig var is managed through config package
	r := mux.NewRouter()
	conf := config.Conf()
	internetAvailable := config.NetCheck("1.1.1.1:53")
	logger.Logger.Info(fmt.Sprintf("Internet available: %t", internetAvailable))
	// c2c mode
	if !internetAvailable {
		logger.Logger.Info("Entering C2C mode")
		if err := system.C2cMode(); err != nil {
			logger.Logger.Error(fmt.Sprintf("Error running C2C mode: %v", err))
			panic("Couldn't run C2C mode!")
		}
	}
	// async operation to retrieve version info if updates are on
	versionUpdateChannel := make(chan bool)
	remoteVersion := false
	if conf.UpdateMode == "auto" {
		remoteVersion = true
		// get version info from remote server
		go func() {
			_, versionUpdate := config.CheckVersion()
			versionUpdateChannel <- versionUpdate
		}()
		// otherwise use cached if possible, or save hardcoded defaults and use that
	} else {
		versionStruct := config.LocalVersion()
		releaseChannel := conf.UpdateBranch
		targetChan := versionStruct.Groundseg[releaseChannel]
		config.VersionInfo = targetChan
	}
	// routines/version.go
	go routines.CheckVersionLoop() // infinite version check loop
	go routines.AptUpdateLoop()    // check for base OS updates
	// routines/docker.go
	go routines.DockerListener()            // listen to docker daemon
	go routines.DockerSubscriptionHandler() // digest docker events from eventbus

	// digest urbit transition events
	go rectify.UrbitTransitionHandler()
	// digest system transition events
	go rectify.SystemTransitionHandler()
	// digest new ship transition events
	go rectify.NewShipTransitionHandler()
	// digest retrieve data
	go rectify.RectifyUrbit()
	// get the startram config from server
	if conf.WgRegistered == true {
		_, err := startram.Retrieve()
		if err != nil {
			logger.Logger.Warn(fmt.Sprintf("Could not retrieve StarTram/Anchor config: %v", err))
		}
	}
	// log manager routine
	go routines.LogEvent()
	// block until version info returns
	if remoteVersion == true {
		select {
		case <-versionUpdateChannel:
			logger.Logger.Info("Version info retrieved")
		case <-time.After(10 * time.Second):
			logger.Logger.Warn("Could not retrieve version info after 10 seconds!")
			versionStruct := config.LocalVersion()
			releaseChannel := conf.UpdateBranch
			targetChan := versionStruct.Groundseg[releaseChannel]
			config.VersionInfo = targetChan
		}
	}

	// grab wg now cause its huge
	if wgConf, err := docker.GetLatestContainerInfo("wireguard"); err != nil {
		logger.Logger.Warn(fmt.Sprintf("Error getting WG container: %v", err))
	} else {
		logger.Logger.Info("Downloading Wireguard image")
		if _, err := docker.PullImageIfNotExist("wireguard", wgConf); err != nil {
			logger.Logger.Warn(fmt.Sprintf("Error getting WG container: %v", err))
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
	r.HandleFunc("/ws", ws.WsHandler)
	r.HandleFunc("/export/{container}", exporter.ExportHandler)
	http.ListenAndServe(":3000", r)
}
