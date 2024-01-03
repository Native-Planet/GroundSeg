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
	"embed"
	"fmt"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/exporter"
	"groundseg/importer"
	"groundseg/leak"
	"groundseg/logger"
	"groundseg/rectify"
	"groundseg/routines"
	"groundseg/startram"
	"groundseg/system"
	"groundseg/ws"
	"io/fs"
	"mime"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

var (
	//go:embed web/*
	//go:embed web/_app/*
	//go:embed web/_app/immutable/assets/_*
	//go:embed web/_app/immutable/chunks/_*
	//go:embed web/_app/immutable/entry/_*
	// we need to explicitly embed stuff starting with underscore
	content       embed.FS
	webContent, _ = fs.Sub(content, "web")
	fileServer    = http.FileServer(http.FS(webContent))
	capContent    embed.FS
	capFs         = http.FS(capContent)
	capFileServer = http.FileServer(capFs)
	DevMode       = false
	shutdownChan  = make(chan struct{})
	HttpPort      = 80
)

// test for internet connectivity and interrupt ServerControl if we need to switch
func C2cCheck() {
	conf := config.Conf()
	isNPBox := system.IsNPBox(config.BasePath)
	internetAvailable := connCheck()
	if !internetAvailable && system.Device != "" && isNPBox {
		if err := system.C2CMode(); err != nil {
			logger.Logger.Error(fmt.Sprintf("Error activating C2C mode: %v", err))
		} else {
			logger.Logger.Info("GroundSeg is in C2C Mode")
			system.SetC2CMode(true)
			// start killswitch timer in another routine if c2cInterval in system.json is greater than 0
			if conf.C2cInterval > 0 {
				go killSwitch()
			}
		}
	}
}

func killSwitch() {
	for {
		conf := config.Conf()
		time.Sleep(time.Duration(conf.C2cInterval) * time.Second)
		logger.Logger.Info("Graceful reboot from C2C mode...")
		routines.GracefulShipExit()
		if config.DebugMode {
			logger.Logger.Debug(fmt.Sprintf("DebugMode detected, skipping shutdown. Exiting program."))
			os.Exit(0)
		} else {
			logger.Logger.Info(fmt.Sprintf("Rebooting device.."))
			cmd := exec.Command("reboot")
			cmd.Run()
		}
	}
}

func connCheck() bool {
	internetAvailable := config.NetCheck("1.1.1.1:53")
	if internetAvailable {
		return true
	}
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if config.NetCheck("1.1.1.1:53") {
				return true
			}
		case <-timeout:
			return false
		}
	}
}

func loadService(loadFunc func() error, errMsg string) {
	go func() {
		if err := loadFunc(); err != nil {
			logger.Logger.Error(fmt.Sprintf("%s %v", errMsg, err))
		}
	}()
}

// autodetect mime type
func ContentTypeSetter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || strings.HasSuffix(r.URL.Path, "/") {
			next.ServeHTTP(w, r)
			return
		}
		ext := filepath.Ext(r.URL.Path)
		mimeType := mime.TypeByExtension(ext)
		if mimeType != "" {
			w.Header().Set("Content-Type", mimeType)
		}
		next.ServeHTTP(w, r)
	})
}

func startServer() { // *http.Server {
	r := mux.NewRouter()
	// r.PathPrefix("/").Handler(ContentTypeSetter(fileServer))
	r.PathPrefix("/").Handler(ContentTypeSetter(http.HandlerFunc(fallbackToIndex(http.FS(webContent)))))
	httpPort := fmt.Sprintf(":%d", HttpPort)
	server := &http.Server{
		Addr:    httpPort,
		Handler: r,
	}
	w := mux.NewRouter()
	w.HandleFunc("/ws", ws.WsHandler)
	w.HandleFunc("/export/{container}", exporter.ExportHandler)
	w.HandleFunc("/import/{uploadSession}/{patp}", importer.HTTPUploadHandler)
	wsServer := &http.Server{
		Addr:    ":3000",
		Handler: w,
	}
	go server.ListenAndServe()
	wsServer.ListenAndServe()
	//go wsServer.ListenAndServe()
	// http.ListenAndServe(":80", r)
	//logger.Logger.Info("GroundSeg web UI serving")
	//return server
}

func fallbackToIndex(fs http.FileSystem) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file, err := fs.Open(r.URL.Path)
		if err != nil {
			r.URL.Path = "/index.html"
		} else {
			defer file.Close()
		}
		http.FileServer(fs).ServeHTTP(w, r)
	}
}

func main() {
	// make sure resolved was reenabled
	if err := system.EnableResolved(); err != nil {
		logger.Logger.Error(fmt.Sprintf("Unable to enabled systemd-resolved: %v", err))
	}
	// push error messages to fe
	go rectify.ErrorMessageHandler()
	// global SysConfig var is managed through config package
	conf := config.Conf()
	internetAvailable := config.NetCheck("1.1.1.1:53")
	logger.Logger.Info(fmt.Sprintf("Internet available: %t", internetAvailable))
	// ongoing connectivity check
	go C2cCheck()
	// async operation to retrieve version info if updates are on
	versionUpdateChannel := make(chan bool)
	remoteVersion := false
	// debug mode
	for _, arg := range os.Args[1:] {
		// trigger dev mode with `./groundseg dev`
		if arg == "dev" {
			logger.Logger.Info("Starting pprof (:6060)")
			go http.ListenAndServe("0.0.0.0:6060", nil)
		}
		// set non-default port like `--http-port=8060`
		if strings.HasPrefix(arg, "--http-port=") {
			portStr := strings.TrimPrefix(arg, "--http-port=")
			port, err := strconv.Atoi(portStr)
			if err != nil {
				logger.Logger.Error(fmt.Sprintf("Invalid port number: %s -- defaulting to 80", portStr))
			} else {
				HttpPort = port
			}
			logger.Logger.Info(fmt.Sprintf("Running HTTP server on port %d", HttpPort))
		}
	}
	// setup swap
	logger.Logger.Info(fmt.Sprintf("Setting up swap %v for %vG", conf.SwapFile, conf.SwapVal))
	if err := system.ConfigureSwap(conf.SwapFile, conf.SwapVal); err != nil {
		logger.Logger.Error(fmt.Sprintf("Unable to set swap: %v", err))
	}
	// update mode
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
	// digest imported ship transition events
	go rectify.ImportShipTransitionHandler()
	// digest retrieve data
	go rectify.RectifyUrbit()
	// get the startram config from server
	if conf.WgRegistered == true {
		_, err := startram.Retrieve()
		if err != nil {
			logger.Logger.Warn(fmt.Sprintf("Could not retrieve StarTram/Anchor config: %v", err))
		}
	}
	// gallseg
	go leak.StartLeak()
	// pack scheduler
	go routines.PackScheduleLoop()
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
		if err := docker.LoadWireguard(); err != nil {
			logger.Logger.Error(fmt.Sprintf("Unable to load Wireguard: %v", err))
		} else {
			// Load MC
			loadService(docker.LoadMC, "Unable to load MinIO Client!")
			// Load MinIOs
			loadService(docker.LoadMinIOs, "Unable to load MinIO containers!")
		}
	}
	// Load Netdata
	loadService(docker.LoadNetdata, "Unable to load Netdata!")
	// Load Urbits
	loadService(docker.LoadUrbits, "Unable to load Urbit ships!")
	// Load Penpai
	loadService(docker.LoadLlama, "Unable to load Llama GPT!")

	startServer()
}
