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
	"context"
	"embed"
	"fmt"
	"goseg/config"
	"goseg/docker"
	"goseg/exporter"
	"goseg/handler"
	"goseg/logger"
	"goseg/rectify"
	"goseg/routines"
	"goseg/startram"
	"goseg/system"
	"goseg/ws"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
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
	//go:embed web/captive/*
	capContent    embed.FS
	capFs         = http.FS(capContent)
	capFileServer = http.FileServer(capFs)
	DevMode       = false
	shutdownChan  = make(chan struct{})
)

func ServerControl() {
	activeServer := startMainServer()
	// var activeServer *http.Server
	for {
		select {
		case useC2C := <-system.C2cChan:
			if activeServer != nil {
				close(shutdownChan)
				shutdownChan = make(chan struct{})
				activeServer.Shutdown(context.Background())
			}
			if useC2C {
				activeServer = startC2CServer()
			} else {
				activeServer = startMainServer()
			}
		}
	}
}

func C2cLoop() {
	c2cActive := false
	for {
		internetAvailable := config.NetCheck("1.1.1.1:53")
		if !internetAvailable && !c2cActive {
			if err := system.C2cMode(); err != nil {
				logger.Logger.Error(fmt.Sprintf("Error activating C2C mode:", err))
			} else {
				logger.Logger.Info("No connection -- entering C2C mode")
				c2cActive = true
				system.C2cChan <- true
			}
		} else if internetAvailable && c2cActive {
			if err := system.TeardownHostAPD(); err != nil {
				logger.Logger.Error(fmt.Sprintf("Error deactivating C2C mode:", err))
			} else {
				logger.Logger.Info("Connection detected -- exiting C2C mode")
				c2cActive = false
				system.C2cChan <- false
			}
		}
		time.Sleep(30 * time.Second)
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

func startC2CServer() *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/", capFileServer)
	mux.HandleFunc("/api", system.CaptiveAPI)
	server := &http.Server{
		Addr:    ":80",
		Handler: mux,
	}
	go func() {
		select {
		case <-shutdownChan:
			server.Shutdown(context.Background())
		}
	}()
	go server.ListenAndServe()
	logger.Logger.Info("C2C web server serving")
	return server
}

func startMainServer() *http.Server {
	r := mux.NewRouter()
	// r.PathPrefix("/").Handler(ContentTypeSetter(fileServer))
	r.PathPrefix("/").Handler(ContentTypeSetter(http.HandlerFunc(fallbackToIndex(http.FS(webContent)))))
	server := &http.Server{
		Addr:    ":80",
		Handler: r,
	}
	w := mux.NewRouter()
	w.HandleFunc("/ws", ws.WsHandler)
	w.HandleFunc("/export/{container}", exporter.ExportHandler)
	w.HandleFunc("/export/{uploadSession}/{patp}", handler.UploadHandler)
	wsServer := &http.Server{
		Addr:    ":3000",
		Handler: w,
	}
	go func() {
		select {
		case <-shutdownChan:
			server.Shutdown(context.Background())
			wsServer.Shutdown(context.Background())
		}
	}()
	go server.ListenAndServe()
	go wsServer.ListenAndServe()
	// http.ListenAndServe(":80", r)
	logger.Logger.Info("GroundSeg web UI serving")
	return server
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
	// push error messages to fe
	go rectify.ErrorMessageHandler()
	// global SysConfig var is managed through config package
	conf := config.Conf()
	internetAvailable := config.NetCheck("1.1.1.1:53")
	logger.Logger.Info(fmt.Sprintf("Internet available: %t", internetAvailable))
	// ongoing connectivity check
	go C2cLoop()
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

	// load the appropriate HTTP server forever
	for {
		ServerControl()
	}
}
